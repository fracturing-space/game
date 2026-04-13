package internalutil

import (
	"database/sql"
	"fmt"
	"io/fs"
	"sort"
	"strings"
	"time"
)

// ApplyMigrations applies embedded SQL migrations in lexicographic order and
// records them in the local schema_migrations table.
func ApplyMigrations(db *sql.DB, migrationFS fs.FS, scope string, now func() time.Time) error {
	if db == nil {
		return fmt.Errorf("sqlite db is required")
	}
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return fmt.Errorf("migration scope is required")
	}
	if now == nil {
		now = time.Now
	}

	if _, err := db.Exec(`
CREATE TABLE IF NOT EXISTS schema_migrations (
	scope TEXT NOT NULL,
	name TEXT NOT NULL,
	applied_at_ns INTEGER NOT NULL,
	PRIMARY KEY (scope, name)
);
`); err != nil {
		return fmt.Errorf("ensure schema migrations table: %w", err)
	}

	entries, err := fs.ReadDir(migrationFS, ".")
	if err != nil {
		return fmt.Errorf("read migrations: %w", err)
	}
	sort.Slice(entries, func(i, j int) bool {
		return entries[i].Name() < entries[j].Name()
	})

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".sql") {
			continue
		}
		applied, err := migrationApplied(db, scope, entry.Name())
		if err != nil {
			return err
		}
		if applied {
			continue
		}

		sqlText, err := fs.ReadFile(migrationFS, entry.Name())
		if err != nil {
			return fmt.Errorf("read migration %s: %w", entry.Name(), err)
		}
		if err := applyMigration(db, scope, entry.Name(), string(sqlText), now().UTC()); err != nil {
			return err
		}
	}

	return nil
}

func migrationApplied(db *sql.DB, scope, name string) (bool, error) {
	var count int64
	if err := db.QueryRow(`
SELECT COUNT(1)
FROM schema_migrations
WHERE scope = ? AND name = ?
`, scope, name).Scan(&count); err != nil {
		return false, fmt.Errorf("query schema migration %s/%s: %w", scope, name, err)
	}
	return count > 0, nil
}

func applyMigration(db *sql.DB, scope, name, sqlText string, appliedAt time.Time) (err error) {
	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("begin migration %s/%s: %w", scope, name, err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	if _, err = tx.Exec(sqlText); err != nil {
		return fmt.Errorf("run migration %s/%s: %w", scope, name, err)
	}
	if _, err = tx.Exec(`
INSERT INTO schema_migrations (scope, name, applied_at_ns)
VALUES (?, ?, ?)
`, scope, name, appliedAt.UnixNano()); err != nil {
		return fmt.Errorf("record migration %s/%s: %w", scope, name, err)
	}
	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit migration %s/%s: %w", scope, name, err)
	}
	return nil
}
