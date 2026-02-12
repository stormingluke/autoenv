package domain

import (
	"crypto/sha256"
	"fmt"
)

type EnvFile struct {
	Path   string
	Mtime  int64
	Values map[string]string
}

type DiffResult struct {
	Export map[string]string
	Unset  []string
}

func HashValue(value string) string {
	h := sha256.Sum256([]byte(value))
	return fmt.Sprintf("%x", h[:8])
}

func KeyHashes(ef *EnvFile) map[string]string {
	if ef == nil {
		return nil
	}
	hashes := make(map[string]string, len(ef.Values))
	for k, v := range ef.Values {
		hashes[k] = HashValue(v)
	}
	return hashes
}

func Diff(envFile *EnvFile, loadedKeys []SessionKey) DiffResult {
	result := DiffResult{Export: make(map[string]string)}

	if envFile == nil {
		for _, k := range loadedKeys {
			result.Unset = append(result.Unset, k.KeyName)
		}
		return result
	}

	loadedMap := make(map[string]string, len(loadedKeys))
	for _, k := range loadedKeys {
		loadedMap[k.KeyName] = k.KeyHash
	}

	for key, value := range envFile.Values {
		newHash := HashValue(value)
		if oldHash, exists := loadedMap[key]; !exists || oldHash != newHash {
			result.Export[key] = value
		}
	}

	for _, k := range loadedKeys {
		if _, exists := envFile.Values[k.KeyName]; !exists {
			result.Unset = append(result.Unset, k.KeyName)
		}
	}

	return result
}
