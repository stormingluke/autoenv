package store

import (
	"database/sql"
)

type Session struct {
	ShellPID     int
	ProjectPath  string
	EnvFileMtime int64
	LoadedAt     string
}

type SessionKey struct {
	ShellPID int
	KeyName  string
	KeyHash  string
}

type SessionRepo struct {
	db *sql.DB
}

func NewSessionRepo(db *sql.DB) *SessionRepo {
	return &SessionRepo{db: db}
}

func (r *SessionRepo) Get(shellPID int) (*Session, error) {
	row := r.db.QueryRow(
		`SELECT shell_pid, project_path, env_file_mtime, loaded_at FROM sessions WHERE shell_pid = ?`,
		shellPID,
	)
	s := &Session{}
	if err := row.Scan(&s.ShellPID, &s.ProjectPath, &s.EnvFileMtime, &s.LoadedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return s, nil
}

func (r *SessionRepo) Upsert(shellPID int, projectPath string, envFileMtime int64) error {
	_, err := r.db.Exec(
		`INSERT INTO sessions (shell_pid, project_path, env_file_mtime)
		 VALUES (?, ?, ?)
		 ON CONFLICT(shell_pid) DO UPDATE SET
		   project_path = excluded.project_path,
		   env_file_mtime = excluded.env_file_mtime,
		   loaded_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		shellPID, projectPath, envFileMtime,
	)
	return err
}

func (r *SessionRepo) Delete(shellPID int) error {
	_, err := r.db.Exec(`DELETE FROM sessions WHERE shell_pid = ?`, shellPID)
	return err
}

func (r *SessionRepo) GetKeys(shellPID int) ([]SessionKey, error) {
	rows, err := r.db.Query(
		`SELECT shell_pid, key_name, key_hash FROM session_keys WHERE shell_pid = ?`,
		shellPID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []SessionKey
	for rows.Next() {
		var k SessionKey
		if err := rows.Scan(&k.ShellPID, &k.KeyName, &k.KeyHash); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	return keys, rows.Err()
}

func (r *SessionRepo) SetKeys(shellPID int, keys map[string]string) error {
	tx, err := r.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(`DELETE FROM session_keys WHERE shell_pid = ?`, shellPID)
	if err != nil {
		return err
	}

	stmt, err := tx.Prepare(
		`INSERT INTO session_keys (shell_pid, key_name, key_hash) VALUES (?, ?, ?)`,
	)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for name, hash := range keys {
		if _, err := stmt.Exec(shellPID, name, hash); err != nil {
			return err
		}
	}

	return tx.Commit()
}
