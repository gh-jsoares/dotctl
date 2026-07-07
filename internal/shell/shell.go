package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gh-jsoares/dotctl/internal/config"
)

func Generate(shellName string) (string, error) {
	cfg, err := config.Load()
	if err != nil {
		return "", err
	}

	switch shellName {
	case "zsh":
		return zshInit(cfg.Guards), nil
	case "bash":
		return bashInit(cfg.Guards), nil
	default:
		return "", fmt.Errorf("unsupported shell: %s (supported: zsh, bash)", shellName)
	}
}

func Install() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	dir := installDir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(dir, "init.zsh"), []byte(zshInit(cfg.Guards)), 0o644); err != nil {
		return err
	}

	return nil
}

func installDir() string {
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share", "dotctl")
}

func generateGuards(guards []config.GuardConfig) string {
	if len(guards) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("# --- Context-guarded commands ---\n")

	for _, g := range guards {
		msg := g.Message
		if msg == "" {
			msg = fmt.Sprintf("Running %s outside '%s' context.", g.Command, g.Context)
		}
		safeMsg := strings.ReplaceAll(msg, "'", "'\\''")

		b.WriteString(fmt.Sprintf(`function %s() {
  local current=""
  [[ -f "$DOTCTL_STATE_DIR/current-context" ]] && current="$(<"$DOTCTL_STATE_DIR/current-context")"

  if [[ "$current" != %q ]]; then
    printf '⚠ %s\n'
    printf 'Current context: %%s. Continue? [y/N] ' "${current:-unset}"
    local reply
    read -r reply
    if [[ "$reply" != [yY] ]]; then
      printf 'Aborted.\n'
      return 1
    fi
  fi

  command %s "$@"
}

`, g.Command, g.Context, safeMsg, g.Command))
	}

	return b.String()
}

func zshInit(guards []config.GuardConfig) string {
	base := `# dotctl shell integration for zsh
# Source this file in .zshrc or use: eval "$(dotctl shell-init zsh)"

# --- PATH setup ---
[[ -d /opt/homebrew/bin ]] && eval "$(/opt/homebrew/bin/brew shellenv)"
[[ -d /run/current-system/sw/bin ]] && export PATH="/run/current-system/sw/bin:$PATH"

# --- Context env vars (file source, no subprocess) ---
typeset -g DOTCTL_STATE_DIR="${XDG_STATE_HOME:-$HOME/.local/state}/dotctl"
[[ -f "$DOTCTL_STATE_DIR/env" ]] && source "$DOTCTL_STATE_DIR/env"

# --- ctx wrapper (sources env after switch) ---
function ctx() {
  if [[ $# -eq 0 ]]; then
    command dotctl ctx
    return $?
  fi

  command dotctl ctx "$@"
  local ret=$?

  if [[ $ret -eq 0 && -f "$DOTCTL_STATE_DIR/env" ]]; then
    source "$DOTCTL_STATE_DIR/env"
  fi

  return $ret
}

# --- chdir hook for project context detection ---
function _dotctl_chdir_hook() {
  local dotctx=""

  # Walk up to find .dotctx (max 10 levels to avoid slow traversal)
  local i=0
  local check="$PWD"
  while [[ $i -lt 10 && "$check" != "/" ]]; do
    if [[ -f "$check/.dotctx" ]]; then
      dotctx="$check/.dotctx"
      break
    fi
    check="${check:h}"
    ((i++))
  done

  if [[ -n "$dotctx" && -f "$DOTCTL_STATE_DIR/current-context" ]]; then
    local current preferred
    current="$(<"$DOTCTL_STATE_DIR/current-context")"
    preferred="$(command grep -m1 '^context' "$dotctx" 2>/dev/null | command sed 's/.*=[ ]*["'\'']*\([^"'\'']*\)["'\'']*$/\1/')"

    if [[ -n "$preferred" && "$preferred" != "$current" ]]; then
      printf '%s\n' "⚠ This repo prefers context '$preferred'. Current: '$current'. Run: ctx $preferred"
    fi
  fi
}

autoload -Uz add-zsh-hook
add-zsh-hook chpwd _dotctl_chdir_hook

`
	return base + generateGuards(guards)
}

func bashInit(guards []config.GuardConfig) string {
	base := `# dotctl shell integration for bash
# Source this file in .bashrc or use: eval "$(dotctl shell-init bash)"

export DOTCTL_STATE_DIR="${XDG_STATE_HOME:-$HOME/.local/state}/dotctl"
[[ -f "$DOTCTL_STATE_DIR/env" ]] && source "$DOTCTL_STATE_DIR/env"

# --- ctx wrapper ---
function ctx() {
  if [[ $# -eq 0 ]]; then
    command dotctl ctx
    return $?
  fi

  command dotctl ctx "$@"
  local ret=$?

  if [[ $ret -eq 0 && -f "$DOTCTL_STATE_DIR/env" ]]; then
    source "$DOTCTL_STATE_DIR/env"
  fi

  return $ret
}

`
	return base + generateGuards(guards)
}
