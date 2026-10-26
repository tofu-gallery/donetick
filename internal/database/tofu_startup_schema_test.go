// Tofu (2026-10-26): this file is Tofu's addition; it is not part of upstream
// Donetick. It is licensed under the AGPL-3.0, like the rest of this server.

package database

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"donetick.com/core/config"
)

type fakePgError struct{ code string }

func (e fakePgError) Error() string    { return "ERROR: fake (SQLSTATE " + e.code + ")" }
func (e fakePgError) SQLState() string { return e.code }

func TestIsDuplicateObjectError(t *testing.T) {
	for code, want := range map[string]bool{"42P07": true, "42710": true, "23505": true, "42P01": false, "55P03": false} {
		err := fmt.Errorf("%w handling x.sql", fakePgError{code})
		if got := isDuplicateObjectError(err); got != want {
			t.Errorf("isDuplicateObjectError(%s) = %v, want %v", code, got, want)
		}
	}
	if isDuplicateObjectError(errors.New("relation already exists")) {
		t.Error("an error without a SQLSTATE counted as a duplicate")
	}
}

// Five servers starting at the same moment against one empty PostgreSQL database must
// all get through the schema step. Runs only with TOFU_TEST_POSTGRES_URL set to an
// EMPTY database, for example postgresql://user:password@127.0.0.1:5432/scratch?sslmode=disable.
func TestStartupSchemaConcurrentStarts(t *testing.T) {
	url := os.Getenv("TOFU_TEST_POSTGRES_URL")
	if url == "" {
		t.Skip("TOFU_TEST_POSTGRES_URL is not set")
	}
	cfg := &config.Config{}
	cfg.Database.Type = "postgres"
	open := func() *gorm.DB {
		db, err := gorm.Open(postgres.Open(poolerSafeDSN(url)), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
		if err != nil {
			t.Fatal(err)
		}
		return db
	}
	for round := 1; round <= 2; round++ {
		var wg sync.WaitGroup
		errs := make([]error, 5)
		for i := range errs {
			db := open()
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				errs[i] = StartupSchema(context.Background(), db, cfg)
			}(i)
		}
		wg.Wait()
		for i, err := range errs {
			if err != nil {
				t.Fatalf("round %d, start %d: %v", round, i+1, err)
			}
		}
	}
	db := open()
	var tables, sqlMigrations, dataMigrations int64
	if err := db.Raw(`SELECT count(*) FROM pg_class c
		JOIN pg_namespace n ON n.oid = c.relnamespace WHERE n.nspname = current_schema() AND c.relkind = 'r'`).Row().Scan(&tables); err != nil {
		t.Fatal(err)
	}
	if tables != 41 {
		t.Errorf("%d tables; want 41", tables)
	}
	if err := db.Raw(`SELECT (SELECT count(*) FROM gorp_migrations), (SELECT count(*) FROM migrations)`).Row().Scan(&sqlMigrations, &dataMigrations); err != nil {
		t.Fatal(err)
	}
	if sqlMigrations != 3 || dataMigrations != 10 {
		t.Errorf("%d SQL and %d data migrations recorded; want 3 and 10", sqlMigrations, dataMigrations)
	}
}
