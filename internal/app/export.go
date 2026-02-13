package app

import (
	"github.com/stormingluke/autoenv/internal/domain"
	"github.com/stormingluke/autoenv/internal/port"
)

type ExportService struct {
	sessions  port.SessionRepository
	envLoader port.EnvLoader
	shell     port.ShellRenderer
}

func (s *ExportService) Export(shellType string, shellPID int, cwd string) (string, error) {
	// Load .env directly from cwd (no project DB lookup)
	envFile, err := s.envLoader.Load(cwd)
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

	// No .env in current directory
	if envFile == nil {
		if session == nil {
			return "", nil
		}
		output := s.shell.FormatUnsets(shellType, domain.KeyNames(loadedKeys))
		output += "unset _AUTOENV_ACTIVE\n"
		_ = s.sessions.Delete(shellPID)
		return output, nil
	}

	// Same directory, unchanged .env — skip
	if session != nil && session.ProjectPath == cwd && session.EnvFileMtime == envFile.Mtime {
		return "", nil
	}

	diff := domain.Diff(envFile, loadedKeys)

	// Switching directories — unset old keys not in new .env
	if session != nil && session.ProjectPath != cwd {
		for _, k := range loadedKeys {
			if _, exists := envFile.Values[k.KeyName]; !exists {
				diff.Unset = append(diff.Unset, k.KeyName)
			}
		}
	}

	output := s.shell.FormatUnsets(shellType, diff.Unset) + s.shell.FormatExports(shellType, diff.Export)
	output += "export _AUTOENV_ACTIVE=1\n"

	_ = s.sessions.Upsert(shellPID, cwd, envFile.Mtime)
	_ = s.sessions.SetKeys(shellPID, domain.KeyHashes(envFile))

	return output, nil
}
