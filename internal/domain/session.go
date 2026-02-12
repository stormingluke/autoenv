package domain

type Session struct {
	ShellPID     int
	ProjectPath  string
	EnvFileMtime int64
	LoadedAt     string
}

type SessionKey struct {
	ShellPID int
	KeyName  string
	KeyHash  string
}

func KeyNames(keys []SessionKey) []string {
	names := make([]string, len(keys))
	for i, k := range keys {
		names[i] = k.KeyName
	}
	return names
}
