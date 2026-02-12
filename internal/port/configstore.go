package port

import "github.com/stormingluke/autoenv/internal/domain"

type ConfigStore interface {
	Get(key string) (string, error)
	Set(key, value string) error
	List() ([]domain.DefaultSetting, error)
}
