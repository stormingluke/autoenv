package app

import (
	"fmt"

	"github.com/stormingluke/autoenv/internal/domain"
	"github.com/stormingluke/autoenv/internal/port"
)

type ConfigureService struct {
	config port.ConfigStore
}

func (s *ConfigureService) Set(key, value string) error {
	if s.config == nil {
		return fmt.Errorf("config store not available")
	}
	return s.config.Set(key, value)
}

func (s *ConfigureService) Get(key string) (string, error) {
	if s.config == nil {
		return "", fmt.Errorf("config store not available")
	}
	return s.config.Get(key)
}

func (s *ConfigureService) List() ([]domain.DefaultSetting, error) {
	if s.config == nil {
		return nil, fmt.Errorf("config store not available")
	}
	return s.config.List()
}
