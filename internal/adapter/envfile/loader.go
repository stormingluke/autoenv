package envfile

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
	"github.com/stormingluke/autoenv/internal/domain"
	"github.com/stormingluke/autoenv/internal/port"
)

var _ port.EnvLoader = (*Loader)(nil)

type Loader struct{}

func NewLoader() *Loader {
	return &Loader{}
}

func (l *Loader) Load(projectPath string) (*domain.EnvFile, error) {
	envPath := filepath.Join(projectPath, ".env")

	info, err := os.Stat(envPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, fmt.Errorf("stat %s: %w", envPath, err)
	}

	values, err := godotenv.Read(envPath)
	if err != nil {
		return nil, fmt.Errorf("parse %s: %w", envPath, err)
	}

	return &domain.EnvFile{
		Path:   envPath,
		Mtime:  info.ModTime().UnixNano(),
		Values: values,
	}, nil
}
