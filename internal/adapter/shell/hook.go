package shell

import "fmt"

func HookScript(shellType string) (string, error) {
	switch shellType {
	case "zsh":
		return zshHook, nil
	case "bash":
		return bashHook, nil
	default:
		return "", fmt.Errorf("unsupported shell: %s (supported: zsh, bash)", shellType)
	}
}

const zshHook = `_autoenv_hook() {
  if [[ -f .env ]] || [[ -n "$_AUTOENV_ACTIVE" ]]; then
    eval "$(autoenv export zsh)"
  fi
}
typeset -ag chpwd_functions
if [[ -z "${chpwd_functions[(r)_autoenv_hook]+1}" ]]; then
  chpwd_functions=(_autoenv_hook $chpwd_functions)
fi
_autoenv_hook
`

const bashHook = `_autoenv_hook() {
  local prev_exit=$?
  if [[ -f .env ]] || [[ -n "$_AUTOENV_ACTIVE" ]]; then
    eval "$(autoenv export bash)"
  fi
  return $prev_exit
}
if [[ ";${PROMPT_COMMAND[*]:-};" != *";_autoenv_hook;"* ]]; then
  PROMPT_COMMAND="_autoenv_hook${PROMPT_COMMAND:+;$PROMPT_COMMAND}"
fi
_autoenv_hook
`
