package port

type ShellRenderer interface {
	FormatExports(shellType string, vars map[string]string) string
	FormatUnsets(shellType string, keys []string) string
}
