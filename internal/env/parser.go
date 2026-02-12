package env

import (
	"crypto/sha256"
	"fmt"
	"os"
	"path/filepath"

	"github.com/joho/godotenv"
)

type EnvFile struct {
	Path   string
	Mtime  int64
	Values map[string]string
}

// Load reads and parses a .env file from the given project directory.
// Returns nil if no .env file exists.
func Load(projectPath string) (*EnvFile, error) {
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

	return &EnvFile{
		Path:   envPath,
		Mtime:  info.ModTime().UnixNano(),
		Values: values,
	}, nil
}

// HashValue returns a SHA-256 hash of a value (for change detection without storing secrets).
func HashValue(value string) string {
	h := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", h[:8]) // 16-char hex, enough for change detection
}
