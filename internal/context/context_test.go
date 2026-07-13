package context

import (
	"os"
	"path/filepath"
	"testing"
)

func setupTestEnv(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Set HOME and DOTFILES_DIR to isolate from real system
	t.Setenv("HOME", dir)
	t.Setenv("DOTFILES_DIR", filepath.Join(dir, "dotfiles"))

	// Create directory structure
	dirs := []string{
		filepath.Join(dir, "dotfiles", "contexts"),
		filepath.Join(dir, ".aws-work"),
		filepath.Join(dir, ".aws-personal"),
		filepath.Join(dir, ".kube-work"),
		filepath.Join(dir, ".kube-personal"),
		filepath.Join(dir, ".docker-work"),
		filepath.Join(dir, ".docker-personal"),
		filepath.Join(dir, ".config", "git"),
		filepath.Join(dir, ".local", "state", "dotctl"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Create test context files
	workCtx := `[identity]
name = "Work User"
email = "work@example.com"
git_config = "config-work"
ssh_key = "id_ed25519_work"

[symlinks]
"~/.aws" = "~/.aws-work"
"~/.kube" = "~/.kube-work"

[env]
DOCKER_CONFIG = "~/.docker-work"
PROJECTS_DIR = "~/projects/work"
NPM_CONFIG_REGISTRY = "https://nexus.company.com/repository/npm/"
`
	personalCtx := `[identity]
name = "Personal User"
email = "personal@example.com"
git_config = "config-personal"
ssh_key = "id_ed25519_personal"

[symlinks]
"~/.aws" = "~/.aws-personal"
"~/.kube" = "~/.kube-personal"

[env]
DOCKER_CONFIG = "~/.docker-personal"
PROJECTS_DIR = "~/projects/personal"
`
	os.WriteFile(filepath.Join(dir, "dotfiles", "contexts", "work.toml"), []byte(workCtx), 0o644)
	os.WriteFile(filepath.Join(dir, "dotfiles", "contexts", "personal.toml"), []byte(personalCtx), 0o644)

	// Create git config files
	os.WriteFile(filepath.Join(dir, ".config", "git", "config-work"), []byte("[user]\nname = Work User\n"), 0o644)
	os.WriteFile(filepath.Join(dir, ".config", "git", "config-personal"), []byte("[user]\nname = Personal User\n"), 0o644)

	return dir
}

func TestSwitchContext(t *testing.T) {
	dir := setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatalf("NewManager: %v", err)
	}

	// Switch to work
	if err := mgr.Switch("work"); err != nil {
		t.Fatalf("Switch(work): %v", err)
	}

	// Verify current context
	current, err := mgr.Current()
	if err != nil {
		t.Fatalf("Current: %v", err)
	}
	if current != "work" {
		t.Errorf("expected current=work, got %q", current)
	}

	// Verify symlinks
	awsTarget, err := os.Readlink(filepath.Join(dir, ".aws"))
	if err != nil {
		t.Fatalf("readlink .aws: %v", err)
	}
	if awsTarget != filepath.Join(dir, ".aws-work") {
		t.Errorf("expected .aws -> .aws-work, got %q", awsTarget)
	}

	kubeTarget, err := os.Readlink(filepath.Join(dir, ".kube"))
	if err != nil {
		t.Fatalf("readlink .kube: %v", err)
	}
	if kubeTarget != filepath.Join(dir, ".kube-work") {
		t.Errorf("expected .kube -> .kube-work, got %q", kubeTarget)
	}

	// Verify generated git config
	gitCfg, err := os.ReadFile(filepath.Join(dir, ".config", "git", "config-work"))
	if err != nil {
		t.Fatalf("reading config-work: %v", err)
	}
	if !contains(string(gitCfg), "work@example.com") {
		t.Errorf("config-work missing email:\n%s", gitCfg)
	}

	// Verify env file
	envData, err := os.ReadFile(mgr.EnvFilePath())
	if err != nil {
		t.Fatalf("reading env file: %v", err)
	}
	envStr := string(envData)
	if !contains(envStr, `DOTCTL_CONTEXT="work"`) {
		t.Errorf("env file missing DOTCTL_CONTEXT=work:\n%s", envStr)
	}
	if !contains(envStr, `DOCKER_CONFIG=`) {
		t.Errorf("env file missing DOCKER_CONFIG:\n%s", envStr)
	}

	// Switch to personal
	if err := mgr.Switch("personal"); err != nil {
		t.Fatalf("Switch(personal): %v", err)
	}

	awsTarget, _ = os.Readlink(filepath.Join(dir, ".aws"))
	if awsTarget != filepath.Join(dir, ".aws-personal") {
		t.Errorf("after switch: expected .aws -> .aws-personal, got %q", awsTarget)
	}

	current, _ = mgr.Current()
	if current != "personal" {
		t.Errorf("expected current=personal, got %q", current)
	}
}

func TestSwitchIdempotent(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}

	if err := mgr.Switch("work"); err != nil {
		t.Fatal(err)
	}
	// Second switch should be a no-op
	if err := mgr.Switch("work"); err != nil {
		t.Fatalf("idempotent switch failed: %v", err)
	}
}

func TestSetDefault(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}

	if err := mgr.SetDefault("personal"); err != nil {
		t.Fatal(err)
	}

	// With no current context, Current() should return the default
	current, err := mgr.Current()
	if err != nil {
		t.Fatal(err)
	}
	if current != "personal" {
		t.Errorf("expected default=personal, got %q", current)
	}
}

func TestInvalidContext(t *testing.T) {
	setupTestEnv(t)

	mgr, err := NewManager()
	if err != nil {
		t.Fatal(err)
	}

	if err := mgr.Switch("nonexistent"); err == nil {
		t.Error("expected error for nonexistent context")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsStr(s, substr))
}

func containsStr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
