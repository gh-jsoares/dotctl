package plugin

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gh-jsoares/dotctl/internal/config"
)

func TestEvaluateConditions_NoConditions(t *testing.T) {
	cfg := &config.Config{Dotfiles: config.DotfilesConfig{Path: t.TempDir()}}
	plugins := []*Plugin{{Name: "foo"}}
	result := EvaluateConditions(plugins, cfg, "work")
	if len(result) != 1 {
		t.Fatalf("expected 1 enabled plugin, got %d", len(result))
	}
}

func TestEvaluateConditions_PathsExist_Pass(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "stow", "vim"), 0o755)

	cfg := &config.Config{Dotfiles: config.DotfilesConfig{Path: dir}}
	plugins := []*Plugin{{
		Name:       "foo",
		Conditions: Conditions{PathsExist: []string{"stow/vim"}},
	}}
	result := EvaluateConditions(plugins, cfg, "")
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
}

func TestEvaluateConditions_PathsExist_Fail(t *testing.T) {
	dir := t.TempDir()

	cfg := &config.Config{Dotfiles: config.DotfilesConfig{Path: dir}}
	plugins := []*Plugin{{
		Name:       "foo",
		Conditions: Conditions{PathsExist: []string{"nonexistent/path"}},
	}}
	result := EvaluateConditions(plugins, cfg, "")
	if len(result) != 0 {
		t.Fatalf("expected 0, got %d", len(result))
	}
}

func TestEvaluateConditions_BinariesExist_Pass(t *testing.T) {
	cfg := &config.Config{Dotfiles: config.DotfilesConfig{Path: t.TempDir()}}
	plugins := []*Plugin{{
		Name:       "foo",
		Conditions: Conditions{BinariesExist: []string{"sh"}},
	}}
	result := EvaluateConditions(plugins, cfg, "")
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
}

func TestEvaluateConditions_BinariesExist_Fail(t *testing.T) {
	cfg := &config.Config{Dotfiles: config.DotfilesConfig{Path: t.TempDir()}}
	plugins := []*Plugin{{
		Name:       "foo",
		Conditions: Conditions{BinariesExist: []string{"nonexistent_binary_xyz_123"}},
	}}
	result := EvaluateConditions(plugins, cfg, "")
	if len(result) != 0 {
		t.Fatalf("expected 0, got %d", len(result))
	}
}

func TestEvaluateConditions_BinariesAbsent_Pass(t *testing.T) {
	cfg := &config.Config{Dotfiles: config.DotfilesConfig{Path: t.TempDir()}}
	plugins := []*Plugin{{
		Name:       "foo",
		Conditions: Conditions{BinariesAbsent: []string{"nonexistent_binary_xyz_123"}},
	}}
	result := EvaluateConditions(plugins, cfg, "")
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
}

func TestEvaluateConditions_BinariesAbsent_Fail(t *testing.T) {
	cfg := &config.Config{Dotfiles: config.DotfilesConfig{Path: t.TempDir()}}
	plugins := []*Plugin{{
		Name:       "foo",
		Conditions: Conditions{BinariesAbsent: []string{"sh"}},
	}}
	result := EvaluateConditions(plugins, cfg, "")
	if len(result) != 0 {
		t.Fatalf("expected 0, got %d", len(result))
	}
}

func TestEvaluateConditions_Contexts_Match(t *testing.T) {
	cfg := &config.Config{Dotfiles: config.DotfilesConfig{Path: t.TempDir()}}
	plugins := []*Plugin{{
		Name:       "foo",
		Conditions: Conditions{Contexts: []string{"work", "personal"}},
	}}
	result := EvaluateConditions(plugins, cfg, "work")
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
}

func TestEvaluateConditions_Contexts_NoMatch(t *testing.T) {
	cfg := &config.Config{Dotfiles: config.DotfilesConfig{Path: t.TempDir()}}
	plugins := []*Plugin{{
		Name:       "foo",
		Conditions: Conditions{Contexts: []string{"work"}},
	}}
	result := EvaluateConditions(plugins, cfg, "personal")
	if len(result) != 0 {
		t.Fatalf("expected 0, got %d", len(result))
	}
}

func TestEvaluateConditions_Check_Pass(t *testing.T) {
	cfg := &config.Config{Dotfiles: config.DotfilesConfig{Path: t.TempDir()}}
	plugins := []*Plugin{{
		Name:       "foo",
		Conditions: Conditions{Check: "true"},
	}}
	result := EvaluateConditions(plugins, cfg, "")
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
}

func TestEvaluateConditions_Check_Fail(t *testing.T) {
	cfg := &config.Config{Dotfiles: config.DotfilesConfig{Path: t.TempDir()}}
	plugins := []*Plugin{{
		Name:       "foo",
		Conditions: Conditions{Check: "false"},
	}}
	result := EvaluateConditions(plugins, cfg, "")
	if len(result) != 0 {
		t.Fatalf("expected 0, got %d", len(result))
	}
}

func TestEvaluateConditions_MultipleConditions_AllMustPass(t *testing.T) {
	dir := t.TempDir()
	os.MkdirAll(filepath.Join(dir, "stow", "vim"), 0o755)

	cfg := &config.Config{Dotfiles: config.DotfilesConfig{Path: dir}}
	plugins := []*Plugin{{
		Name: "foo",
		Conditions: Conditions{
			PathsExist:    []string{"stow/vim"},
			BinariesExist: []string{"sh"},
			Contexts:      []string{"work"},
			Check:         "true",
		},
	}}

	// All pass
	result := EvaluateConditions(plugins, cfg, "work")
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}

	// Context doesn't match
	result = EvaluateConditions(plugins, cfg, "personal")
	if len(result) != 0 {
		t.Fatalf("expected 0 (context mismatch), got %d", len(result))
	}
}
