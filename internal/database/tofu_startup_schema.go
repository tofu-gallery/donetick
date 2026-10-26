// Tofu (2026-10-26): this file is Tofu's addition; it is not part of upstream
// Donetick. It is licensed under the AGPL-3.0, like the rest of this server.

package database

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	migrate "github.com/rubenv/sql-migrate"
	"gorm.io/gorm"

	"donetick.com/core/config"
	"donetick.com/core/logging"
	"donetick.com/core/migrations"
)

// startupSchemaLockKey is the PostgreSQL advisory lock every start of this server
// takes before it touches the schema. Any fixed number works, as long as every start
// of the app uses the same one; this one is "donetick" in ASCII.
const startupSchemaLockKey int64 = 0x646f6e657469636b

// startupSchemaLockWait bounds how long a start waits for another start's schema
// step before it gives up and fails.
const startupSchemaLockWait = 3 * time.Minute

// sql-migrate records the SQL files of internal/database/migrations it has run in
// this table, which it creates with this statement (its defaults for PostgreSQL).
const sqlMigrationsTable = "gorp_migrations"
const createSQLMigrationsTable = `create table if not exists "gorp_migrations" ("id" text not null primary key, "applied_at" timestamp with time zone)`

// StartupSchema runs everything a start does to the database's schema: GORM's
// AutoMigrate, Donetick's data migrations (recorded in the migrations table), the SQL
// files in internal/database/migrations (recorded in gorp_migrations) and, on
// PostgreSQL, row-level security on every table the app makes.
//
// Several copies of the server may start at the same moment against the same database
// (a host that starts more than one container for a new deployment), and upstream's
// steps then race: on an empty database the starts create the same tables at once,
// and all but one crash. On PostgreSQL the whole step is therefore ONE transaction
// whose first statement takes a transaction-level advisory lock
// (pg_advisory_xact_lock): one start does the work while the others wait, then each of
// them takes the lock in turn, finds the work done and goes on. DDL is transactional
// in PostgreSQL, so nobody ever sees half a schema, and a failed step leaves the
// database as it was. This holds through a transaction pooler (Tofu's DATABASE_URL,
// port 6543), which keeps one server connection for a transaction from BEGIN to its
// end and never hands it to another client in between: the lock, the work and the
// commit all happen on that one connection, and the lock ends with the transaction.
// (A session-level lock, pg_advisory_lock, would not be safe there, because between
// transactions the pooler gives the server connection to other clients.)
//
// SQLite keeps upstream's steps exactly as they were: nothing else shares its file.
func StartupSchema(ctx context.Context, db *gorm.DB, cfg *config.Config) error {
	if db.Dialector.Name() != "postgres" {
		// Upstream ignored the first two errors, and so does this path.
		_ = Migration(db)
		_ = migrations.Run(ctx, db)
		return MigrationScripts(db, cfg)
	}

	log := logging.DefaultLogger()
	run := func() (string, error) {
		var outcome string
		err := db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
			var err error
			outcome, err = startupSchemaStep(ctx, tx)
			return err
		})
		return outcome, err
	}
	outcome, err := run()
	if err != nil && isDuplicateObjectError(err) {
		// Every start of this server takes the lock, so only something that does not
		// (an older version of the app, say) can get here. The failed transaction left
		// nothing behind; run the whole step once more.
		log.Warnf("start-up schema: %v; something else changed the schema at the same moment, so the whole step runs once more", err)
		outcome, err = run()
	}
	if err != nil {
		return fmt.Errorf("start-up schema step: %w", err)
	}
	log.Infof("start-up schema: %s", outcome)
	return nil
}

// startupSchemaStep is the body of the one transaction: the bounded wait for the
// lock, then the schema state read inside the lock, never assumed.
func startupSchemaStep(ctx context.Context, tx *gorm.DB) (string, error) {
	log := logging.DefaultLogger()
	if err := tx.Exec(fmt.Sprintf("SET LOCAL lock_timeout = '%dms'", startupSchemaLockWait.Milliseconds())).Error; err != nil {
		return "", err
	}
	waitStart := time.Now()
	if err := tx.Exec(fmt.Sprintf("SELECT pg_advisory_xact_lock(%d)", startupSchemaLockKey)).Error; err != nil {
		if sqlState(err) == "55P03" {
			return "", fmt.Errorf("another start has held the schema lock for %s: %w", startupSchemaLockWait, err)
		}
		return "", err
	}
	waited := time.Since(waitStart).Round(time.Millisecond)

	if err := Migration(tx); err != nil {
		return "", fmt.Errorf("auto-migrate: %w", err)
	}
	// Upstream goes on when a data migration fails, and so does this step: the
	// savepoint undoes this start's data migrations, which stay unrecorded and run
	// again at the next start, as they would upstream.
	if err := tx.SavePoint("tofu_data_migrations").Error; err != nil {
		return "", err
	}
	if err := migrations.Run(ctx, tx); err != nil {
		log.Errorf("start-up schema: a data migration failed, the server goes on without it: %v", err)
		if err := tx.RollbackTo("tofu_data_migrations").Error; err != nil {
			return "", err
		}
	}
	applied, err := runSQLMigrations(tx)
	if err != nil {
		return "", err
	}
	outcome := fmt.Sprintf("schema checked and brought up to date, %d SQL migrations applied (waited %s for the lock)", applied, waited)

	closed, err := enableRowLevelSecurity(tx)
	if err != nil {
		return "", fmt.Errorf("row-level security: %w", err)
	}
	if closed > 0 {
		outcome += fmt.Sprintf("; row-level security turned on for %d tables", closed)
	}
	return outcome, nil
}

// runSQLMigrations applies the SQL files of internal/database/migrations that
// gorp_migrations does not record yet, in this transaction, as MigrationScripts does
// through sql-migrate (which can only run them on connections of its own): the same
// files, split into statements by sql-migrate's own parser, in its order, recorded in
// its table, and refused, as it refuses, when the table records a file this server
// does not have.
func runSQLMigrations(tx *gorm.DB) (int, error) {
	source := &migrate.EmbedFileSystemMigrationSource{FileSystem: embeddedMigrations, Root: "migrations"}
	found, err := source.FindMigrations()
	if err != nil {
		return 0, err
	}
	if err := tx.Exec(createSQLMigrationsTable).Error; err != nil {
		return 0, err
	}
	var recorded []string
	if err := tx.Raw(`SELECT id FROM "` + sqlMigrationsTable + `"`).Scan(&recorded).Error; err != nil {
		return 0, err
	}
	known := make(map[string]bool, len(found))
	for _, m := range found {
		known[m.Id] = true
	}
	applied := make(map[string]bool, len(recorded))
	for _, id := range recorded {
		if !known[id] {
			return 0, fmt.Errorf("unknown migration in database: %s", id)
		}
		applied[id] = true
	}
	n := 0
	for _, m := range found {
		if applied[m.Id] {
			continue
		}
		if m.DisableTransactionUp {
			return n, fmt.Errorf("%s asks to run outside a transaction, which the start-up schema step cannot do", m.Id)
		}
		for _, statement := range m.Up {
			if err := tx.Exec(statement).Error; err != nil {
				return n, fmt.Errorf("%w handling %s", err, m.Id)
			}
		}
		if err := tx.Exec(`INSERT INTO "`+sqlMigrationsTable+`" ("id", "applied_at") VALUES (?, ?)`, m.Id, time.Now()).Error; err != nil {
			return n, err
		}
		n++
	}
	fmt.Printf("Applied %d migrations!\n", n)
	return n, nil
}

var plainTableName = regexp.MustCompile(`^[a-z_][a-z0-9_]*$`)

// appTables names every table the app makes: the tables of the models AutoMigrate
// creates (the data migrations' migrations table among them) with any join tables
// they declare, and sql-migrate's gorp_migrations.
func appTables(db *gorm.DB) ([]string, error) {
	seen := map[string]bool{}
	var names []string
	add := func(name string) {
		if !seen[name] {
			seen[name] = true
			names = append(names, name)
		}
	}
	for _, model := range autoMigrateModels {
		stmt := &gorm.Statement{DB: db}
		if err := stmt.Parse(model); err != nil {
			return nil, err
		}
		add(stmt.Schema.Table)
		for _, rel := range stmt.Schema.Relationships.Many2Many {
			if rel.JoinTable != nil {
				add(rel.JoinTable.Table)
			}
		}
	}
	add(sqlMigrationsTable)
	for _, name := range names {
		if !plainTableName.MatchString(name) {
			return nil, fmt.Errorf("unexpected table name %q", name)
		}
	}
	return names, nil
}

// enableRowLevelSecurity turns row-level security on for every table the app makes
// that does not have it yet, in one round trip, and returns how many it changed. On a
// database whose default privileges grant new tables in public to the roles of
// Supabase's Data API (anon, authenticated), as Tofu's databases made before
// 2026-10-30 do, the app's tables would otherwise be readable and writable through
// that API by anyone holding the project's public key. With row-level security on and
// no policy those roles get no row, while the server's own connection, which owns the
// tables, is not limited by it. Only tables this role owns are touched.
func enableRowLevelSecurity(tx *gorm.DB) (int, error) {
	names, err := appTables(tx)
	if err != nil {
		return 0, err
	}
	rows, err := tx.Raw(`SELECT n.nspname, c.relname
		FROM pg_class c JOIN pg_namespace n ON n.oid = c.relnamespace
		WHERE n.nspname = current_schema() AND c.relkind IN ('r', 'p') AND NOT c.relrowsecurity
			AND pg_has_role(c.relowner, 'USAGE') AND c.relname = ANY(string_to_array(?, ','))
		ORDER BY c.relname`, strings.Join(names, ",")).Rows()
	if err != nil {
		return 0, err
	}
	var statements strings.Builder
	n := 0
	for rows.Next() {
		var schema, table string
		if err := rows.Scan(&schema, &table); err != nil {
			rows.Close()
			return 0, err
		}
		fmt.Fprintf(&statements, "ALTER TABLE %s.%s ENABLE ROW LEVEL SECURITY;\n", quoteIdentifier(schema), quoteIdentifier(table))
		n++
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return 0, err
	}
	rows.Close()
	if n == 0 {
		return 0, nil
	}
	if err := tx.Exec(statements.String()).Error; err != nil {
		return 0, err
	}
	return n, nil
}

func quoteIdentifier(name string) string {
	return `"` + strings.ReplaceAll(name, `"`, `""`) + `"`
}

// sqlState is the SQLSTATE of a PostgreSQL error anywhere in err's chain, or "".
func sqlState(err error) string {
	var pgErr interface{ SQLState() string }
	if errors.As(err, &pgErr) {
		return pgErr.SQLState()
	}
	return ""
}

// isDuplicateObjectError says whether err is PostgreSQL refusing to create something
// that another session created at the same moment: a table, index, type, sequence,
// schema or function that already exists, or a duplicate key, the catalog's own
// (pg_type_typname_nsp_index) or a migration record's.
func isDuplicateObjectError(err error) bool {
	switch sqlState(err) {
	case "42P07", "42710", "42P06", "42723", "23505":
		return true
	}
	return false
}
