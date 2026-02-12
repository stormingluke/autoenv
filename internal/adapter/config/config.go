package config

import (
	"os"
	"path/filepath"
)

type Config struct {
	Dir            string
	ProjectsDBPath string
	SessionsDBPath string
	TursoURL       string
	TursoAuthToken string
}

func Load() *Config {
	dir := configDir()

	return &Config{
		Dir:            dir,
		ProjectsDBPath: filepath.Join(dir, "projects.db"),
		SessionsDBPath: filepath.Join(dir, "sessions.db"),
		TursoURL:       os.Getenv("AUTOENV_TURSO_URL"),
		TursoAuthToken: os.Getenv("AUTOENV_TURSO_AUTH_TOKEN"),
	}
}

func (c *Config) EnsureDir() error {
	return os.MkdirAll(c.Dir, 0o755)
}

func configDir() string {
	if d := os.Getenv("AUTOENV_CONFIG_DIR"); d != "" {
		return d
	}
	if d := os.Getenv("XDG_CONFIG_HOME"); d != "" {
		return filepath.Join(d, "autoenv")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "autoenv")
}
