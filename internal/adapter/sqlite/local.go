package sqlite

import (
	"database/sql"
	"fmt"
	"io"

	_ "github.com/tursodatabase/go-libsql"
)

type sessionsDB struct {
	db *sql.DB
}

func (s *sessionsDB) Close() error { return s.db.Close() }

func OpenLocal(path string) (*sql.DB, error) {
	db, err := sql.Open("libsql", fmt.Sprintf("file:%s", path))
	if err != nil {
		return nil, fmt.Errorf("open local db %s: %w", path, err)
	}
	db.SetMaxOpenConns(1)
	var mode string
	if err := db.QueryRow("PRAGMA journal_mode=WAL").Scan(&mode); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("set WAL mode: %w", err)
	}
	// libsql may not support this pragma; ignore error
	_, _ = db.Exec("PRAGMA foreign_keys = ON")
	return db, nil
}

func OpenSessionsDB(path string) (*sql.DB, io.Closer, error) {
	db, err := OpenLocal(path)
	if err != nil {
		return nil, nil, err
	}
	if err := migrateSessions(db); err != nil {
		_ = db.Close()
		return nil, nil, fmt.Errorf("migrate sessions: %w", err)
	}
	return db, &sessionsDB{db: db}, nil
}
