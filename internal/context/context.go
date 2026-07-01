package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/BurntSushi/toml"
	"github.com/gh-jsoares/dotctl/internal/config"
)

type ContextDef struct {
	Identity IdentityConfig        `toml:"identity"`
	Symlinks map[string]string     `toml:"symlinks"`
	Env      map[string]string     `toml:"env"`
	Lazy map[string]string `toml:"env.lazy"`
}

type IdentityConfig struct {
	GitConfig string `toml:"git_config"`
	SSHKey    string `toml:"ssh_key"`
}

type Manager struct {
	cfg      *config.Config
	stateDir string
}

func NewManager() (*Manager, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, fmt.Errorf("loading config: %w", err)
	}

	stateDir := config.StateDir()
	if err := os.MkdirAll(stateDir, 0o755); err != nil {
		return nil, fmt.Errorf("creating state dir: %w", err)
	}

	return &Manager{cfg: cfg, stateDir: stateDir}, nil
}

func (m *Manager) Switch(name string) error {
	ctx, err := m.Load(name)
	if err != nil {
		return err
	}

	current, _ := m.Current()
	if current == name {
		return nil
	}

	if err := m.applySymlinks(ctx); err != nil {
		return fmt.Errorf("applying symlinks: %w", err)
	}

	if err := m.writeCurrentContext(name); err != nil {
		return fmt.Errorf("writing current context: %w", err)
	}

	if err := m.generateEnvFile(name, ctx); err != nil {
		return fmt.Errorf("generating env file: %w", err)
	}

	m.updateTmuxEnv(ctx)

	return nil
}

func (m *Manager) SetDefault(name string) error {
	if _, err := m.Load(name); err != nil {
		return err
	}

	path := filepath.Join(m.stateDir, "default-context")
	return os.WriteFile(path, []byte(name), 0o644)
}

func (m *Manager) Current() (string, error) {
	path := filepath.Join(m.stateDir, "current-context")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return m.defaultContext()
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func (m *Manager) EnvFilePath() string {
	return filepath.Join(m.stateDir, "env")
}

func (m *Manager) Load(name string) (*ContextDef, error) {
	path := filepath.Join(m.cfg.Dotfiles.Path, "contexts", name+".toml")
	var ctx ContextDef
	if _, err := toml.DecodeFile(path, &ctx); err != nil {
		return nil, fmt.Errorf("loading context %q: %w", name, err)
	}
	return &ctx, nil
}

func (m *Manager) applySymlinks(ctx *ContextDef) error {
	home, _ := os.UserHomeDir()

	for link, target := range ctx.Symlinks {
		link = expandHome(link, home)
		target = expandHome(target, home)

		if _, err := os.Lstat(link); err == nil {
			if err := os.Remove(link); err != nil {
				return fmt.Errorf("removing existing symlink %s: %w", link, err)
			}
		}

		if err := os.Symlink(target, link); err != nil {
			return fmt.Errorf("creating symlink %s -> %s: %w", link, target, err)
		}
	}

	if ctx.Identity.GitConfig != "" {
		gitDir := filepath.Join(home, ".config", "git")
		link := filepath.Join(gitDir, "config-current")
		target := filepath.Join(gitDir, ctx.Identity.GitConfig)

		if _, err := os.Lstat(link); err == nil {
			os.Remove(link)
		}
		if err := os.Symlink(target, link); err != nil {
			return fmt.Errorf("creating git config symlink: %w", err)
		}
	}

	return nil
}

func (m *Manager) generateEnvFile(name string, ctx *ContextDef) error {
	home, _ := os.UserHomeDir()
	var b strings.Builder

	b.WriteString(fmt.Sprintf("export DOTCTL_CONTEXT=%q\n", name))
	for k, v := range ctx.Env {
		v = expandHome(v, home)
		b.WriteString(fmt.Sprintf("export %s=%q\n", k, v))
	}

	return os.WriteFile(m.EnvFilePath(), []byte(b.String()), 0o644)
}

func (m *Manager) writeCurrentContext(name string) error {
	path := filepath.Join(m.stateDir, "current-context")
	return os.WriteFile(path, []byte(name), 0o644)
}

func (m *Manager) updateTmuxEnv(ctx *ContextDef) {
	// Only update if we're inside tmux
	if os.Getenv("TMUX") == "" {
		return
	}

	home, _ := os.UserHomeDir()
	for k, v := range ctx.Env {
		v = expandHome(v, home)
		// tmux set-environment updates the server env for new panes
		runTmuxSetEnv(k, v)
	}
	current, _ := m.Current()
	runTmuxSetEnv("DOTCTL_CONTEXT", current)
}

func (m *Manager) defaultContext() (string, error) {
	path := filepath.Join(m.stateDir, "default-context")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return "", nil
		}
		return "", err
	}
	return strings.TrimSpace(string(data)), nil
}

func expandHome(path, home string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}
