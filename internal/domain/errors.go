package domain

import "errors"

var (
	ErrProjectNotFound = errors.New("project not found")
	ErrSessionNotFound = errors.New("session not found")
	ErrNoEnvFile       = errors.New("no .env file found")
)

type DefaultSetting struct {
	Key   string
	Value string
}
