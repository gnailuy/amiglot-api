package db

import (
	"database/sql"
	"fmt"

	_ "github.com/lib/pq"

	"github.com/gnailuy/amiglot-api/internal/config"
)

// New opens a database connection if DATABASE_URL is set.
// Returns (nil, nil) when no URL is provided.
func New(cfg config.Config) (*sql.DB, error) {
	if cfg.DatabaseURL == "" {
		return nil, nil
	}

	conn, err := sql.Open("postgres", cfg.DatabaseURL)
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	if err := conn.Ping(); err != nil {
		_ = conn.Close()
		return nil, fmt.Errorf("ping db: %w", err)
	}

	return conn, nil
}
