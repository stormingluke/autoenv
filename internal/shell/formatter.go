package shell

import (
	"fmt"
	"sort"
	"strings"
)

// FormatExports returns shell commands to export the given key-value pairs.
func FormatExports(shellType string, vars map[string]string) string {
	if len(vars) == 0 {
		return ""
	}

	// Sort keys for deterministic output
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		b.WriteString(formatExport(shellType, k, vars[k]))
		b.WriteByte('\n')
	}
	return b.String()
}

// FormatUnsets returns shell commands to unset the given keys.
func FormatUnsets(shellType string, keys []string) string {
	if len(keys) == 0 {
		return ""
	}

	sorted := make([]string, len(keys))
	copy(sorted, keys)
	sort.Strings(sorted)

	var b strings.Builder
	for _, k := range sorted {
		fmt.Fprintf(&b, "unset %s\n", k)
	}
	return b.String()
}

func formatExport(shellType, key, value string) string {
	escaped := shellEscape(value)
	return fmt.Sprintf("export %s='%s'", key, escaped)
}

// shellEscape escapes single quotes for safe embedding in shell single-quoted strings.
func shellEscape(s string) string {
	return strings.ReplaceAll(s, "'", "'\\''")
}
