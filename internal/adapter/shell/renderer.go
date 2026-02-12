package shell

import (
	"fmt"
	"sort"
	"strings"

	"github.com/stormingluke/autoenv/internal/port"
)

var _ port.ShellRenderer = (*Renderer)(nil)

type Renderer struct{}

func NewRenderer() *Renderer {
	return &Renderer{}
}

func (r *Renderer) FormatExports(shellType string, vars map[string]string) string {
	if len(vars) == 0 {
		return ""
	}

	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var b strings.Builder
	for _, k := range keys {
		escaped := strings.ReplaceAll(vars[k], "'", "'\\''")
		fmt.Fprintf(&b, "export %s='%s'\n", k, escaped)
	}
	return b.String()
}

func (r *Renderer) FormatUnsets(shellType string, keys []string) string {
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
