package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gh-jsoares/dotctl/internal/config"
)

func TestGenerateZshNoGuards(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	t.Setenv("DOTFILES_DIR", filepath.Join(dir, "dotfiles"))

	code, err := Generate("zsh")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(code, "function ctx()") {
		t.Error("missing ctx function")
	}
	if !strings.Contains(code, "_dotctl_chdir_hook") {
		t.Error("missing chdir hook")
	}
	if strings.Contains(code, "Context-guarded") {
		t.Error("should not have guard section with no guards configured")
	}
}

func TestGenerateZshWithGuards(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("DOTFILES_DIR", filepath.Join(dir, "dotfiles"))

	configDir := filepath.Join(dir, ".config", "dotctl")
	os.MkdirAll(configDir, 0o755)
	os.WriteFile(filepath.Join(configDir, "config.toml"), []byte(`
[[guards]]
command = "awscreds"
context = "work"
message = "writes to AWS config"

[[guards]]
command = "helm"
context = "work"
`), 0o644)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))

	code, err := Generate("zsh")
	if err != nil {
		t.Fatal(err)
	}

	if !strings.Contains(code, "function awscreds()") {
		t.Error("missing awscreds guard")
	}
	if !strings.Contains(code, "function helm()") {
		t.Error("missing helm guard")
	}
	if !strings.Contains(code, "writes to AWS config") {
		t.Error("missing custom message for awscreds")
	}
	if !strings.Contains(code, "Running helm outside") {
		t.Error("missing default message for helm")
	}
}

func TestGenerateUnsupportedShell(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	t.Setenv("DOTFILES_DIR", filepath.Join(dir, "dotfiles"))

	_, err := Generate("fish")
	if err == nil {
		t.Error("expected error for unsupported shell")
	}
}

func TestInstall(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	t.Setenv("XDG_CONFIG_HOME", filepath.Join(dir, ".config"))
	t.Setenv("DOTFILES_DIR", filepath.Join(dir, "dotfiles"))

	if err := Install(); err != nil {
		t.Fatal(err)
	}

	initFile := filepath.Join(dir, ".local", "share", "dotctl", "init.zsh")
	if _, err := os.Stat(initFile); err != nil {
		t.Errorf("init.zsh not created: %v", err)
	}
}

func TestGenerateGuardsOutput(t *testing.T) {
	guards := []config.GuardConfig{
		{Command: "test-cmd", Context: "work", Message: "custom msg"},
	}

	output := generateGuards(guards)

	if !strings.Contains(output, "function test-cmd()") {
		t.Error("missing function declaration")
	}
	if !strings.Contains(output, `"work"`) {
		t.Error("missing context check")
	}
	if !strings.Contains(output, "custom msg") {
		t.Error("missing custom message")
	}
	if !strings.Contains(output, "command test-cmd") {
		t.Error("missing command passthrough")
	}
}
