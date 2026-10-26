// Package storage implements the transfer ports with SQLite.
package storage

import (
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	_ "modernc.org/sqlite" // pure Go SQLite driver, nothing to install
)

// Open opens a fresh SQLite database and runs schema.sql on it.
//
//   - dbPath == "": the database lives in memory and disappears when the app stops (default).
//   - dbPath != "": the database is a file in WAL mode, so it can be opened with a SQLite viewer
//     while the app runs. The file is deleted on every start, so each run still begins clean.
//
// The pool is limited to one connection. For the in-memory database this is required,
// because the data lives inside that one connection and must never be closed.
func Open(dbPath, schemaPath string) (*sql.DB, error) {
	dsn := "file::memory:?_pragma=foreign_keys(1)"
	if dbPath != "" {
		if err := removeDatabaseFiles(dbPath); err != nil {
			return nil, err
		}
		// WAL mode lets a viewer (DBeaver, DB Browser) read the file while the app writes to it.
		// Without it, a viewer's open read transaction blocks every write with SQLITE_BUSY.
		dsn = "file:" + filepath.ToSlash(dbPath) +
			"?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)"
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	db.SetMaxOpenConns(1)
	db.SetMaxIdleConns(1)
	db.SetConnMaxLifetime(0)
	db.SetConnMaxIdleTime(0)

	if err := applySchema(db, schemaPath); err != nil {
		db.Close()
		return nil, err
	}
	return db, nil
}

func applySchema(db *sql.DB, schemaPath string) error {
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read %s: %w", schemaPath, err)
	}
	if _, err := db.Exec(string(schema)); err != nil {
		return fmt.Errorf("apply %s: %w", schemaPath, err)
	}
	return nil
}

// removeDatabaseFiles deletes the database file and SQLite's side files from an earlier run.
func removeDatabaseFiles(dbPath string) error {
	for _, path := range []string{dbPath, dbPath + "-wal", dbPath + "-shm", dbPath + "-journal"} {
		if err := os.Remove(path); err != nil && !errors.Is(err, fs.ErrNotExist) {
			return fmt.Errorf("remove old database %s: %w", path, err)
		}
	}
	return nil
}
