package state

import (
	"database/sql"
	"embed"
	"fmt"

	"github.com/pressly/goose/v3"

	_ "github.com/lib/pq"
)

//go:embed migrations/*.sql
var embedMigrations embed.FS

type PostgresStateStore struct {
	db *sql.DB
}

func NewPostgresStateStore(connectionString string) (*PostgresStateStore, error) {
	db, err := sql.Open("postgres", connectionString)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	if err := db.Ping(); err != nil {
		return nil, fmt.Errorf("failed to connect to database: %w", err)
	}

	store := &PostgresStateStore{db: db}

	if err := store.runMigrations(); err != nil {
		return nil, fmt.Errorf("failed to run migrations: %w", err)
	}

	return store, nil
}

func (s *PostgresStateStore) Close() error {
	if s.db != nil {
		return s.db.Close()
	}
	return nil
}

func (s *PostgresStateStore) DB() *sql.DB {
	return s.db
}

func (s *PostgresStateStore) runMigrations() error {
	goose.SetBaseFS(embedMigrations)

	if err := goose.SetDialect("postgres"); err != nil {
		return fmt.Errorf("failed to set goose dialect: %w", err)
	}

	if err := goose.Up(s.db, "migrations"); err != nil {
		return fmt.Errorf("failed to run migrations: %w", err)
	}

	return nil
}

func nullString(s string) sql.NullString {
	if s == "" {
		return sql.NullString{Valid: false}
	}
	return sql.NullString{String: s, Valid: true}
}

func nullBytes(b []byte) interface{} {
	if b == nil {
		return nil
	}
	return b
}
