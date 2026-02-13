package sqlite

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tursodatabase/go-libsql"
)

type TursoDB struct {
	DB        *sql.DB
	connector *libsql.Connector
}

func OpenTurso(dbPath, tursoURL, authToken string) (*TursoDB, error) {
	if tursoURL == "" || authToken == "" {
		return openTursoLocal(dbPath)
	}

	connector, err := openReplicaConnector(dbPath, tursoURL, authToken)
	if err != nil && strings.Contains(err.Error(), "metadata file does not") {
		// DB was created locally without Turso — remove stale files and retry
		removeStaleDB(dbPath)
		connector, err = openReplicaConnector(dbPath, tursoURL, authToken)
	}
	if err != nil {
		return nil, fmt.Errorf("turso connector at %s: %w", filepath.Dir(dbPath), err)
	}

	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(1)

	if err := migrateProjects(db); err != nil {
		_ = db.Close()
		_ = connector.Close()
		return nil, fmt.Errorf("migrate projects: %w", err)
	}

	return &TursoDB{DB: db, connector: connector}, nil
}

func openReplicaConnector(dbPath, tursoURL, authToken string) (*libsql.Connector, error) {
	return libsql.NewEmbeddedReplicaConnector(
		dbPath,
		tursoURL,
		libsql.WithAuthToken(authToken),
		libsql.WithSyncInterval(0),
	)
}

func removeStaleDB(dbPath string) {
	for _, suffix := range []string{"", "-shm", "-wal"} {
		_ = os.Remove(dbPath + suffix)
	}
}

func openTursoLocal(path string) (*TursoDB, error) {
	db, err := OpenLocal(path)
	if err != nil {
		return nil, err
	}
	if err := migrateProjects(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate projects: %w", err)
	}
	return &TursoDB{DB: db}, nil
}

func (t *TursoDB) Sync() error {
	if t.connector == nil {
		return fmt.Errorf("turso cloud sync not configured (set AUTOENV_TURSO_DATABASE_URL and AUTOENV_TURSO_AUTH_TOKEN)")
	}
	_, err := t.connector.Sync()
	return err
}

func (t *TursoDB) Close() error {
	_ = t.DB.Close()
	if t.connector != nil {
		_ = t.connector.Close()
	}
	return nil
}
