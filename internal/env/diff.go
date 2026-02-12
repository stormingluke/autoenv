package env

import "github.com/stormingluke/autoenv/internal/store"

type DiffResult struct {
	Export map[string]string // keys to export with their values
	Unset  []string          // keys to unset
}

// Diff computes what needs to be exported and unset given the current .env file
// and the previously loaded session keys.
func Diff(envFile *EnvFile, loadedKeys []store.SessionKey) DiffResult {
	result := DiffResult{
		Export: make(map[string]string),
	}

	if envFile == nil {
		// No .env file — unset everything that was previously loaded
		for _, k := range loadedKeys {
			result.Unset = append(result.Unset, k.KeyName)
		}
		return result
	}

	loadedMap := make(map[string]string, len(loadedKeys))
	for _, k := range loadedKeys {
		loadedMap[k.KeyName] = k.KeyHash
	}

	// Export new or changed keys
	for key, value := range envFile.Values {
		newHash := HashValue(value)
		if oldHash, exists := loadedMap[key]; !exists || oldHash != newHash {
			result.Export[key] = value
		}
	}

	// Unset keys that were loaded but no longer in .env
	for _, k := range loadedKeys {
		if _, exists := envFile.Values[k.KeyName]; !exists {
			result.Unset = append(result.Unset, k.KeyName)
		}
	}

	return result
}

// KeyHashes returns a map of key name → hash for all values in the env file.
func KeyHashes(envFile *EnvFile) map[string]string {
	if envFile == nil {
		return nil
	}
	hashes := make(map[string]string, len(envFile.Values))
	for k, v := range envFile.Values {
		hashes[k] = HashValue(v)
	}
	return hashes
}
