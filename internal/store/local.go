package store

import (
	"database/sql"
	"fmt"

	_ "github.com/tursodatabase/go-libsql"
)

func OpenLocal(path string) (*sql.DB, error) {
	db, err := sql.Open("libsql", fmt.Sprintf("file:%s", path))
	if err != nil {
		return nil, fmt.Errorf("open local db %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	var mode string
	if err := db.QueryRow("PRAGMA journal_mode=WAL").Scan(&mode); err != nil {
		db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	if _, err := db.Exec("PRAGMA foreign_keys = ON"); err != nil {
		// libsql may not support this pragma; ignore
	}
	return db, nil
}

func OpenSessionsDB(path string) (*sql.DB, error) {
	db, err := OpenLocal(path)
	if err != nil {
		return nil, err
	}
	if err := migrateSessions(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate sessions: %w", err)
	}
	return db, nil
}
