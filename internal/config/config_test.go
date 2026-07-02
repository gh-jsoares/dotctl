package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaults(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("DOTFILES_DIR", "")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	t.Setenv("XDG_STATE_HOME", "")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	expected := filepath.Join(dir, ".dotfiles")
	if cfg.Dotfiles.Path != expected {
		t.Errorf("expected dotfiles.path=%q, got %q", expected, cfg.Dotfiles.Path)
	}
}

func TestLoadFromFile(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("DOTFILES_DIR", "")

	configDir := filepath.Join(dir, ".config", "dotctl")
	os.MkdirAll(configDir, 0o755)

	content := `
machine = "test-machine"

[dotfiles]
path = "~/my-dotfiles"
remote = "git@github.com:user/dotfiles.git"

[dotctl]
remote = "git@github.com:testuser/dotctl.git"

[[guards]]
command = "terraform"
context = "work"
message = "terraform targets work infra"
`
	os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(content), 0o644)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Machine != "test-machine" {
		t.Errorf("machine: got %q", cfg.Machine)
	}
	if cfg.Dotfiles.Path != filepath.Join(dir, "my-dotfiles") {
		t.Errorf("dotfiles.path: got %q", cfg.Dotfiles.Path)
	}
	if cfg.Dotctl.Remote != "git@github.com:testuser/dotctl.git" {
		t.Errorf("dotctl.remote: got %q", cfg.Dotctl.Remote)
	}
	if len(cfg.Guards) != 1 || cfg.Guards[0].Command != "terraform" {
		t.Errorf("guards: got %+v", cfg.Guards)
	}
}

func TestLoadDotfilesEnvOverride(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("DOTFILES_DIR", "/custom/path")
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.Dotfiles.Path != "/custom/path" {
		t.Errorf("expected DOTFILES_DIR override, got %q", cfg.Dotfiles.Path)
	}
}

func TestStateDirXDG(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/tmp/xdg-state")
	got := StateDir()
	expected := "/tmp/xdg-state/dotctl"
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}

func TestStateDirDefault(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_STATE_HOME", "")

	got := StateDir()
	expected := filepath.Join(dir, ".local", "state", "dotctl")
	if got != expected {
		t.Errorf("expected %q, got %q", expected, got)
	}
}
