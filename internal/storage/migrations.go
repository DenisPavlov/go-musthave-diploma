package storage

import (
	"database/sql"
	"fmt"
	"log/slog"
	"os"

	"github.com/pressly/goose/v3"
)

func ApplyMigrations(db *sql.DB, log *slog.Logger) error {
	op := "storage.applyMigrations"

	goose.SetLogger(newLogger(log))

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set dialect for migration: %s %w", op, err)
	}
	if err := goose.Up(db, "migrations"); err != nil {
		return fmt.Errorf("failed to up migrations: %s %w", op, err)
	}
	return nil
}

type logger struct {
	log *slog.Logger
}

func newLogger(log *slog.Logger) *logger {
	return &logger{log: log.With("component", "migrations")}
}

func (l *logger) Printf(format string, v ...interface{}) {
	l.log.Info(fmt.Sprintf(format, v...))
}

func (l *logger) Fatalf(format string, v ...interface{}) {
	l.log.Error(fmt.Sprintf(format, v...))
	os.Exit(1)
}
