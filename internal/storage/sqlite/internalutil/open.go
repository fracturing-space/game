package internalutil

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "modernc.org/sqlite"
)

// Open opens one SQLite connection with the adapter PRAGMA defaults applied.
func Open(path string) (*sql.DB, error) {
	if err := ensureDir(path); err != nil {
		return nil, err
	}
	db, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if _, err := db.Exec(`
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA busy_timeout = 5000;
PRAGMA foreign_keys = ON;
`); err != nil {
		_ = db.Close()
		return nil, err
	}
	return db, nil
}

func ensureDir(path string) error {
	path = strings.TrimSpace(path)
	if path == "" {
		return fmt.Errorf("sqlite path is required")
	}
	dir := filepath.Dir(path)
	if dir == "." {
		return nil
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create storage dir: %w", err)
	}
	return nil
}
