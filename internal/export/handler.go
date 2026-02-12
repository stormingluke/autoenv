package export

import (
	"os"

	"github.com/stormingluke/autoenv/internal/env"
	"github.com/stormingluke/autoenv/internal/shell"
	"github.com/stormingluke/autoenv/internal/store"
)

type Handler struct {
	projects *store.ProjectRepo
	sessions *store.SessionRepo
}

func NewHandler(projects *store.ProjectRepo, sessions *store.SessionRepo) *Handler {
	return &Handler{projects: projects, sessions: sessions}
}

// Export is the hot-path function called on every directory change.
// It returns shell commands to eval.
func (h *Handler) Export(shellType string, shellPID int) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}

	// Find if current directory is inside a registered project
	project, err := h.projects.MatchCurrent(cwd)
	if err != nil {
		return "", err
	}

	// Get current session state
	session, err := h.sessions.Get(shellPID)
	if err != nil {
		return "", err
	}

	loadedKeys, err := h.sessions.GetKeys(shellPID)
	if err != nil {
		return "", err
	}

	// Case 1: Not in a project
	if project == nil {
		if session == nil {
			return "", nil // nothing loaded, nothing to do
		}
		// Left a project — unset everything
		output := shell.FormatUnsets(shellType, keyNames(loadedKeys))
		h.sessions.Delete(shellPID)
		return output, nil
	}

	// Case 2: In a project — load its .env
	envFile, err := env.Load(project.Path)
	if err != nil {
		return "", err
	}

	// If same project with unchanged .env, skip
	if session != nil && session.ProjectPath == project.Path && envFile != nil && session.EnvFileMtime == envFile.Mtime {
		return "", nil
	}

	// Compute diff
	diff := env.Diff(envFile, loadedKeys)

	// If switching projects, also unset old keys not in new .env
	if session != nil && session.ProjectPath != project.Path {
		for _, k := range loadedKeys {
			if envFile == nil {
				diff.Unset = append(diff.Unset, k.KeyName)
			} else if _, exists := envFile.Values[k.KeyName]; !exists {
				diff.Unset = append(diff.Unset, k.KeyName)
			}
		}
	}

	// Build output
	output := shell.FormatUnsets(shellType, diff.Unset) + shell.FormatExports(shellType, diff.Export)

	// Update session state
	if envFile != nil {
		h.sessions.Upsert(shellPID, project.Path, envFile.Mtime)
		h.sessions.SetKeys(shellPID, env.KeyHashes(envFile))
	} else {
		h.sessions.Delete(shellPID)
	}

	return output, nil
}

// LoadProject registers a project and returns export commands for its .env file.
func (h *Handler) LoadProject(shellType string, shellPID int, projectPath, name string) (string, error) {
	if err := h.projects.Upsert(projectPath, name); err != nil {
		return "", err
	}

	envFile, err := env.Load(projectPath)
	if err != nil {
		return "", err
	}
	if envFile == nil {
		return "", nil
	}

	output := shell.FormatExports(shellType, envFile.Values)

	h.sessions.Upsert(shellPID, projectPath, envFile.Mtime)
	h.sessions.SetKeys(shellPID, env.KeyHashes(envFile))

	return output, nil
}

// Clear returns unset commands for all loaded keys and cleans up the session.
func (h *Handler) Clear(shellType string, shellPID int) (string, error) {
	keys, err := h.sessions.GetKeys(shellPID)
	if err != nil {
		return "", err
	}

	output := shell.FormatUnsets(shellType, keyNames(keys))
	h.sessions.Delete(shellPID)
	return output, nil
}

func keyNames(keys []store.SessionKey) []string {
	names := make([]string, len(keys))
	for i, k := range keys {
		names[i] = k.KeyName
	}
	return names
}
