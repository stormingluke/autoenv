package app

import (
	"github.com/stormingluke/autoenv/internal/domain"
	"github.com/stormingluke/autoenv/internal/port"
)

type LoadService struct {
	projects  port.ProjectRepository
	sessions  port.SessionRepository
	envLoader port.EnvLoader
	shell     port.ShellRenderer
}

func (s *LoadService) LoadProject(shellType string, shellPID int, projectPath, name string) (string, error) {
	if err := s.projects.Upsert(projectPath, name); err != nil {
		return "", err
	}

	envFile, err := s.envLoader.Load(projectPath)
	if err != nil {
		return "", err
	}
	if envFile == nil {
		return "", nil
	}

	output := s.shell.FormatExports(shellType, envFile.Values)

	s.sessions.Upsert(shellPID, projectPath, envFile.Mtime)
	s.sessions.SetKeys(shellPID, domain.KeyHashes(envFile))

	return output, nil
}
