package internalutil

import (
	"io/fs"
	"path/filepath"
	"testing"
	"testing/fstest"
	"time"
)

var fixedRecordTime = time.Date(2026, 4, 4, 18, 0, 0, 0, time.UTC)

func TestOpenCreatesDirectoriesAndRejectsBlankPath(t *testing.T) {
	t.Parallel()

	if _, err := Open("   "); err == nil {
		t.Fatal("Open(blank) error = nil, want failure")
	}

	path := filepath.Join(t.TempDir(), "nested", "store.db")
	db, err := Open(path)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	if _, err := db.Exec(`CREATE TABLE IF NOT EXISTS smoke (id INTEGER PRIMARY KEY);`); err != nil {
		t.Fatalf("smoke exec error = %v", err)
	}
}

func TestApplyMigrationsAppliesOnceInOrder(t *testing.T) {
	t.Parallel()

	db, err := Open(filepath.Join(t.TempDir(), "migrations.db"))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	defer func() {
		if err := db.Close(); err != nil {
			t.Fatalf("Close() error = %v", err)
		}
	}()

	migrationFS := fstest.MapFS{
		"002_insert.sql": &fstest.MapFile{Data: []byte(`INSERT INTO sample (id, label) VALUES (1, 'later');`)},
		"001_create.sql": &fstest.MapFile{Data: []byte(`CREATE TABLE sample (id INTEGER PRIMARY KEY, label TEXT NOT NULL);`)},
		"notes.txt":      &fstest.MapFile{Data: []byte(`ignored`)},
	}

	if err := ApplyMigrations(db, fs.FS(migrationFS), "sample", func() time.Time { return fixedRecordTime }); err != nil {
		t.Fatalf("ApplyMigrations(first) error = %v", err)
	}
	if err := ApplyMigrations(db, fs.FS(migrationFS), "sample", func() time.Time { return fixedRecordTime.Add(time.Minute) }); err != nil {
		t.Fatalf("ApplyMigrations(second) error = %v", err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(1) FROM sample`).Scan(&count); err != nil {
		t.Fatalf("count sample rows: %v", err)
	}
	if got, want := count, 1; got != want {
		t.Fatalf("sample row count = %d, want %d", got, want)
	}

	if err := ApplyMigrations(db, fstest.MapFS{}, "   ", func() time.Time { return fixedRecordTime }); err == nil {
		t.Fatal("ApplyMigrations(blank scope) error = nil, want failure")
	}
}
