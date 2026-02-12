package port

import "github.com/stormingluke/autoenv/internal/domain"

type EnvLoader interface {
	Load(projectPath string) (*domain.EnvFile, error)
}
