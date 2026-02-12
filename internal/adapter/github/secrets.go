package github

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/stormingluke/autoenv/internal/port"
)

var _ port.SecretSyncer = (*SecretSyncer)(nil)

type SecretSyncer struct{}

func NewSecretSyncer() *SecretSyncer {
	return &SecretSyncer{}
}

func (s *SecretSyncer) Sync(repo string, secrets map[string]string) error {
	if _, err := exec.LookPath("gh"); err != nil {
		return fmt.Errorf("gh CLI not found: install from https://cli.github.com")
	}

	for key, value := range secrets {
		cmd := exec.Command("gh", "secret", "set", key,
			"--repo", repo,
			"--body", value,
		)
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("set secret %s: %s: %w", key, strings.TrimSpace(string(out)), err)
		}
	}

	return nil
}
