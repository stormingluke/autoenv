package port

import "github.com/stormingluke/autoenv/internal/domain"

type ProjectReader interface {
	MatchCurrent(dir string) (*domain.Project, error)
	ListAll() ([]domain.Project, error)
	FindByPath(path string) (*domain.Project, error)
}

type ProjectWriter interface {
	Upsert(path, name string) error
	Delete(path string) error
}

type ProjectRepository interface {
	ProjectReader
	ProjectWriter
}

type SessionRepository interface {
	Get(shellPID int) (*domain.Session, error)
	Upsert(shellPID int, projectPath string, envFileMtime int64) error
	Delete(shellPID int) error
	GetKeys(shellPID int) ([]domain.SessionKey, error)
	SetKeys(shellPID int, keys map[string]string) error
}
