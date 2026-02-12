package app

import (
	"github.com/stormingluke/autoenv/internal/domain"
	"github.com/stormingluke/autoenv/internal/port"
)

type ListService struct {
	projects port.ProjectReader
}

func (s *ListService) ListProjects() ([]domain.Project, error) {
	return s.projects.ListAll()
}
