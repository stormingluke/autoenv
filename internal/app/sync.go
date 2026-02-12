package app

import (
	"fmt"
	"strings"

	"github.com/stormingluke/autoenv/internal/domain"
	"github.com/stormingluke/autoenv/internal/port"
)

type SyncService struct {
	projects  port.ProjectReader
	envLoader port.EnvLoader
	syncer    port.SecretSyncer
	config    port.ConfigStore
}

func (s *SyncService) SyncSecrets(projectPath, target string) error {
	if s.syncer == nil {
		return fmt.Errorf("secret sync not configured")
	}

	repo, err := s.resolveTarget(target)
	if err != nil {
		return err
	}

	envFile, err := s.envLoader.Load(projectPath)
	if err != nil {
		return err
	}
	if envFile == nil {
		return domain.ErrNoEnvFile
	}

	return s.syncer.Sync(repo, envFile.Values)
}

func (s *SyncService) resolveTarget(target string) (string, error) {
	// Strip github.com/ prefix if present
	target = strings.TrimPrefix(target, "github.com/")

	// If target already contains a slash, it's owner/repo
	if strings.Contains(target, "/") {
		return target, nil
	}

	// Bare repo name — prepend default owner
	if s.config == nil {
		return "", fmt.Errorf("no default owner configured; use full path (e.g., github.com/owner/repo) or run: autoenv configure set github.default_owner <owner>")
	}

	owner, err := s.config.Get("github.default_owner")
	if err != nil {
		return "", fmt.Errorf("no default owner configured; run: autoenv configure set github.default_owner <owner>")
	}

	return owner + "/" + target, nil
}
