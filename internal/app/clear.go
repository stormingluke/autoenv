package app

import (
	"github.com/stormingluke/autoenv/internal/domain"
	"github.com/stormingluke/autoenv/internal/port"
)

type ClearService struct {
	sessions port.SessionRepository
	shell    port.ShellRenderer
}

func (s *ClearService) Clear(shellType string, shellPID int) (string, error) {
	keys, err := s.sessions.GetKeys(shellPID)
	if err != nil {
		return "", err
	}

	output := s.shell.FormatUnsets(shellType, domain.KeyNames(keys))
	s.sessions.Delete(shellPID)
	return output, nil
}
