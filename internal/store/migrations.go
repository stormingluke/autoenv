package store

import "database/sql"

func migrateProjects(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS projects (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			path       TEXT NOT NULL UNIQUE,
			name       TEXT,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		)
	`)
	return err
}

func migrateSessions(db *sql.DB) error {
	_, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS sessions (
			shell_pid      INTEGER PRIMARY KEY,
			project_path   TEXT NOT NULL,
			env_file_mtime INTEGER NOT NULL,
			loaded_at      TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
		);

		CREATE TABLE IF NOT EXISTS session_keys (
			shell_pid  INTEGER NOT NULL REFERENCES sessions(shell_pid) ON DELETE CASCADE,
			key_name   TEXT NOT NULL,
			key_hash   TEXT NOT NULL,
			PRIMARY KEY (shell_pid, key_name)
		);
	`)
	return err
}
