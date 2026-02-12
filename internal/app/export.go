package app

import (
	"github.com/stormingluke/autoenv/internal/domain"
	"github.com/stormingluke/autoenv/internal/port"
)

type ExportService struct {
	projects  port.ProjectReader
	sessions  port.SessionRepository
	envLoader port.EnvLoader
	shell     port.ShellRenderer
}

func (s *ExportService) Export(shellType string, shellPID int, cwd string) (string, error) {
	project, err := s.projects.MatchCurrent(cwd)
	if err != nil {
		return "", err
	}

	session, err := s.sessions.Get(shellPID)
	if err != nil {
		return "", err
	}

	loadedKeys, err := s.sessions.GetKeys(shellPID)
	if err != nil {
		return "", err
	}

	// Not in a project
	if project == nil {
		if session == nil {
			return "", nil
		}
		output := s.shell.FormatUnsets(shellType, domain.KeyNames(loadedKeys))
		_ = s.sessions.Delete(shellPID)
		return output, nil
	}

	// In a project — load its .env
	envFile, err := s.envLoader.Load(project.Path)
	if err != nil {
		return "", err
	}

	// Same project, unchanged .env — skip
	if session != nil && session.ProjectPath == project.Path && envFile != nil && session.EnvFileMtime == envFile.Mtime {
		return "", nil
	}

	diff := domain.Diff(envFile, loadedKeys)

	// Switching projects — unset old keys not in new .env
	if session != nil && session.ProjectPath != project.Path {
		for _, k := range loadedKeys {
			if envFile == nil {
				diff.Unset = append(diff.Unset, k.KeyName)
			} else if _, exists := envFile.Values[k.KeyName]; !exists {
				diff.Unset = append(diff.Unset, k.KeyName)
			}
		}
	}

	output := s.shell.FormatUnsets(shellType, diff.Unset) + s.shell.FormatExports(shellType, diff.Export)

	if envFile != nil {
		_ = s.sessions.Upsert(shellPID, project.Path, envFile.Mtime)
		_ = s.sessions.SetKeys(shellPID, domain.KeyHashes(envFile))
	} else {
		_ = s.sessions.Delete(shellPID)
	}

	return output, nil
}
