package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

type Config struct {
	Dotfiles DotfilesConfig `toml:"dotfiles"`
	Dotctl   DotctlConfig   `toml:"dotctl"`
	SSH      SSHConfig      `toml:"ssh"`
	Machine  string         `toml:"machine"`
	Guards   []GuardConfig  `toml:"guards"`
}

type DotfilesConfig struct {
	Path   string `toml:"path"`
	Remote string `toml:"remote"`
}

type DotctlConfig struct {
	Path      string `toml:"path"`
	Remote    string `toml:"remote"`
	RepoOwner string `toml:"repo_owner"`
	RepoName  string `toml:"repo_name"`
}

type SSHConfig struct {
	Hosts map[string]string `toml:"hosts"`
}

type GuardConfig struct {
	Command string `toml:"command"`
	Context string `toml:"context"`
	Message string `toml:"message"`
}

func DefaultConfigPath() string {
	if xdg := os.Getenv("XDG_CONFIG_HOME"); xdg != "" {
		return filepath.Join(xdg, "dotctl", "config.toml")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".config", "dotctl", "config.toml")
}

func Load() (*Config, error) {
	path := DefaultConfigPath()

	cfg := &Config{
		Dotfiles: DotfilesConfig{
			Path: defaultDotfilesPath(),
		},
	}

	if _, err := os.Stat(path); err == nil {
		if _, err := toml.DecodeFile(path, cfg); err != nil {
			return nil, err
		}
	}

	cfg.Dotfiles.Path = expandPath(cfg.Dotfiles.Path)
	cfg.Dotctl.Path = expandPath(cfg.Dotctl.Path)
	return cfg, nil
}

func StateDir() string {
	if xdg := os.Getenv("XDG_STATE_HOME"); xdg != "" {
		return filepath.Join(xdg, "dotctl")
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "state", "dotctl")
}

func defaultDotfilesPath() string {
	if env := os.Getenv("DOTFILES_DIR"); env != "" {
		return env
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".dotfiles")
}

func expandPath(path string) string {
	if len(path) > 1 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return path
}
