package store

import (
	"database/sql"
	"fmt"
	"path/filepath"

	"github.com/stormingluke/autoenv/internal/config"
	"github.com/tursodatabase/go-libsql"
)

type TursoDB struct {
	DB        *sql.DB
	connector *libsql.Connector
}

func OpenTurso(cfg *config.Config) (*TursoDB, error) {
	if !cfg.TursoEnabled() {
		return openTursoLocal(cfg.ProjectsDBPath)
	}

	dbDir := filepath.Dir(cfg.ProjectsDBPath)
	connector, err := libsql.NewEmbeddedReplicaConnector(
		cfg.ProjectsDBPath,
		cfg.TursoURL,
		libsql.WithAuthToken(cfg.TursoAuthToken),
		libsql.WithSyncInterval(0), // manual sync only
	)
	if err != nil {
		return nil, fmt.Errorf("turso connector at %s: %w", dbDir, err)
	}

	db := sql.OpenDB(connector)
	db.SetMaxOpenConns(1)

	if err := migrateProjects(db); err != nil {
		db.Close()
		connector.Close()
		return nil, fmt.Errorf("migrate projects: %w", err)
	}

	return &TursoDB{DB: db, connector: connector}, nil
}

func openTursoLocal(path string) (*TursoDB, error) {
	db, err := OpenLocal(path)
	if err != nil {
		return nil, err
	}
	if err := migrateProjects(db); err != nil {
		db.Close()
		return nil, fmt.Errorf("migrate projects: %w", err)
	}
	return &TursoDB{DB: db}, nil
}

func (t *TursoDB) Sync() error {
	if t.connector == nil {
		return fmt.Errorf("turso cloud sync not configured (set AUTOENV_TURSO_URL and AUTOENV_TURSO_AUTH_TOKEN)")
	}
	_, err := t.connector.Sync()
	return err
}

func (t *TursoDB) Close() {
	t.DB.Close()
	if t.connector != nil {
		t.connector.Close()
	}
}
