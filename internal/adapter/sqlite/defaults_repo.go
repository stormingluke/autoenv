package sqlite

import (
	"database/sql"
	"fmt"

	"github.com/stormingluke/autoenv/internal/domain"
	"github.com/stormingluke/autoenv/internal/port"
)

var _ port.ConfigStore = (*DefaultsRepo)(nil)

type DefaultsRepo struct {
	db *sql.DB
}

func NewDefaultsRepo(db *sql.DB) *DefaultsRepo {
	return &DefaultsRepo{db: db}
}

func (r *DefaultsRepo) Get(key string) (string, error) {
	var value string
	err := r.db.QueryRow(`SELECT value FROM defaults WHERE key = ?`, key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", fmt.Errorf("no default set for %q", key)
	}
	return value, err
}

func (r *DefaultsRepo) Set(key, value string) error {
	_, err := r.db.Exec(
		`INSERT INTO defaults (key, value) VALUES (?, ?)
		 ON CONFLICT(key) DO UPDATE SET value = excluded.value,
		 updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')`,
		key, value,
	)
	return err
}

func (r *DefaultsRepo) List() ([]domain.DefaultSetting, error) {
	rows, err := r.db.Query(`SELECT key, value FROM defaults ORDER BY key`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	var settings []domain.DefaultSetting
	for rows.Next() {
		var s domain.DefaultSetting
		if err := rows.Scan(&s.Key, &s.Value); err != nil {
			return nil, err
		}
		settings = append(settings, s)
	}
	return settings, rows.Err()
}
