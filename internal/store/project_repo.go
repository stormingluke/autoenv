package store

import (
	"database/sql"
	"fmt"
	"path/filepath"
)

type Project struct {
	ID        int
	Path      string
	Name      string
	CreatedAt string
}

type ProjectRepo struct {
	db *sql.DB
}

func NewProjectRepo(db *sql.DB) *ProjectRepo {
	return &ProjectRepo{db: db}
}

func (r *ProjectRepo) Upsert(path, name string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	_, err = r.db.Exec(
		`INSERT INTO projects (path, name) VALUES (?, ?)
		 ON CONFLICT(path) DO UPDATE SET name = excluded.name`,
		abs, name,
	)
	return err
}

func (r *ProjectRepo) FindByPath(path string) (*Project, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}
	row := r.db.QueryRow(
		`SELECT id, path, name, created_at FROM projects WHERE path = ?`, abs,
	)
	p := &Project{}
	if err := row.Scan(&p.ID, &p.Path, &p.Name, &p.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (r *ProjectRepo) FindByName(name string) (*Project, error) {
	row := r.db.QueryRow(
		`SELECT id, path, name, created_at FROM projects WHERE name = ?`, name,
	)
	p := &Project{}
	if err := row.Scan(&p.ID, &p.Path, &p.Name, &p.CreatedAt); err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}
	return p, nil
}

func (r *ProjectRepo) ListAll() ([]Project, error) {
	rows, err := r.db.Query(
		`SELECT id, path, name, created_at FROM projects ORDER BY name, path`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Path, &p.Name, &p.CreatedAt); err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, rows.Err()
}

func (r *ProjectRepo) Delete(path string) error {
	abs, err := filepath.Abs(path)
	if err != nil {
		return fmt.Errorf("resolve path: %w", err)
	}
	_, err = r.db.Exec(`DELETE FROM projects WHERE path = ?`, abs)
	return err
}

// MatchCurrent finds a registered project that is a prefix of the given directory.
// Returns the longest (most specific) match.
func (r *ProjectRepo) MatchCurrent(dir string) (*Project, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return nil, fmt.Errorf("resolve path: %w", err)
	}

	rows, err := r.db.Query(
		`SELECT id, path, name, created_at FROM projects ORDER BY length(path) DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var p Project
		if err := rows.Scan(&p.ID, &p.Path, &p.Name, &p.CreatedAt); err != nil {
			return nil, err
		}
		// Check if current dir is within this project
		rel, err := filepath.Rel(p.Path, abs)
		if err != nil {
			continue
		}
		// rel must not start with ".." — that means abs is outside p.Path
		if len(rel) >= 2 && rel[:2] == ".." {
			continue
		}
		return &p, nil
	}
	return nil, rows.Err()
}
