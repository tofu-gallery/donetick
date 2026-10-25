package database

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"gorm.io/driver/postgres"
	"gorm.io/plugin/opentelemetry/tracing"

	// "gorm.io/driver/sqlite" // Sqlite driver based on CGO
	"gorm.io/gorm/logger"

	"donetick.com/core/config"
	"donetick.com/core/logging"
	"github.com/glebarez/sqlite" // Pure go SQLite driver, checkout https://github.com/glebarez/sqlite for details
	"gorm.io/gorm"
)

// convertLogLevelToGorm converts application log level to GORM log level
func convertLogLevelToGorm(appLogLevel string) logger.LogLevel {
	switch strings.ToLower(appLogLevel) {
	case "debug":
		return logger.Info // GORM's most verbose level for debugging
	case "info":
		return logger.Warn // Show warnings and errors, but not all queries
	case "warn", "warning", "error", "dpanic", "panic", "fatal":
		return logger.Error // Only show errors
	case "silent":
		return logger.Silent // No logging from GORM
	default:
		return logger.Error // Default to error level for production safety
	}
}

func NewDatabase(cfg *config.Config) (*gorm.DB, error) {
	var db *gorm.DB
	var err error

	// Use GORM logger level based on application log level from config
	gormLogLevel := convertLogLevelToGorm(cfg.Logging.Level)
	newLogger := logger.New(
		log.New(os.Stdout, "\r\n", log.LstdFlags),
		logger.Config{
			SlowThreshold:             200 * time.Millisecond, // Default Logger behavior
			LogLevel:                  gormLogLevel,
			IgnoreRecordNotFoundError: false,                        // Default Logger behavior
			Colorful:                  true,                         // Default Logger behavior
			ParameterizedQueries:      cfg.Logging.Level != "debug", // ParameterizedQueries=true => Don't show data in logs
		},
	)

	gormConfig := gorm.Config{
		Logger: newLogger,
	}

	switch cfg.Database.Type {
	case "postgres":
		dsn := fmt.Sprintf("host=%s port=%v user=%s password=%s dbname=%s sslmode=disable TimeZone=Asia/Shanghai", cfg.Database.Host, cfg.Database.Port, cfg.Database.User, cfg.Database.Password, cfg.Database.Name)
		// Tofu (2026-10-25): when the host hands the database over as DATABASE_URL and
		// DT_DATABASE_HOST is not set, connect with that URL instead of the fields above.
		// The URL keeps its own sslmode (Tofu's database takes TLS connections, which the
		// line above turns off), and poolerSafeDSN makes every query run on the unnamed
		// statement, because Tofu's DATABASE_URL goes through a transaction pooler that
		// keeps no named prepared statements, which are what pgx uses by default.
		if databaseURL := os.Getenv("DATABASE_URL"); databaseURL != "" && os.Getenv("DT_DATABASE_HOST") == "" {
			dsn = poolerSafeDSN(databaseURL)
		}
		for i := 0; i <= 30; i++ {
			db, err = gorm.Open(postgres.Open(dsn), &gormConfig)
			if err == nil {
				break
			}
			logging.DefaultLogger().Warnf("failed to open database: %v", err)
			time.Sleep(500 * time.Millisecond)
		}

	default:
		path := os.Getenv("DT_SQLITE_PATH")
		if path == "" {
			path = "donetick.db"
		}
		db, err = gorm.Open(sqlite.Open(path), &gormConfig)

	}

	if err != nil {
		return nil, err
	}

	// Add OpenTelemetry tracing to GORM
	if err := db.Use(tracing.NewPlugin()); err != nil {
		logging.DefaultLogger().Warnf("failed to enable OpenTelemetry tracing for GORM: %v", err)
	}

	return db, nil
}

// Tofu (2026-10-25): see NewDatabase. Adds default_query_exec_mode=cache_describe
// to a connection string that names no query exec mode of its own: pgx then caches
// each statement's description and runs it on the unnamed statement, which a
// transaction pooler (PgBouncer, Supavisor) can serve. A postgres:// URL takes it as
// a query parameter, a key=value string as one more pair.
func poolerSafeDSN(dsn string) string {
	if strings.Contains(dsn, "default_query_exec_mode=") {
		return dsn
	}
	if strings.HasPrefix(dsn, "postgres://") || strings.HasPrefix(dsn, "postgresql://") {
		separator := "?"
		if strings.Contains(dsn, "?") {
			separator = "&"
		}
		return dsn + separator + "default_query_exec_mode=cache_describe"
	}
	return dsn + " default_query_exec_mode=cache_describe"
}
