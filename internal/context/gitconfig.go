package context

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// GenerateGitConfigs writes ~/.config/git/config (with includeIf rules)
// and per-context config files with identity info.
// The first context listed becomes the default (no includeIf needed).
func (m *Manager) GenerateGitConfigs(defaultCtx string) error {
	names, err := m.List()
	if err != nil {
		return err
	}
	if len(names) == 0 {
		return nil
	}

	home, _ := os.UserHomeDir()
	gitDir := filepath.Join(home, ".config", "git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		return err
	}

	// Generate per-context config files
	for _, name := range names {
		ctx, err := m.Load(name)
		if err != nil {
			continue
		}
		if ctx.Identity.Email == "" {
			continue
		}
		if err := writeContextGitConfig(gitDir, name, ctx); err != nil {
			return err
		}
	}

	// Generate main config with includeIf rules
	return m.writeMainGitConfig(gitDir, names, defaultCtx)
}

func writeContextGitConfig(gitDir, name string, ctx *ContextDef) error {
	var b strings.Builder

	b.WriteString("[user]\n")
	if ctx.Identity.Name != "" {
		b.WriteString(fmt.Sprintf("    name = %s\n", ctx.Identity.Name))
	}
	b.WriteString(fmt.Sprintf("    email = %s\n", ctx.Identity.Email))
	if ctx.Identity.GPGKey != "" {
		b.WriteString(fmt.Sprintf("    signingkey = %s\n", ctx.Identity.GPGKey))
	}

	if ctx.Identity.GPGKey != "" {
		b.WriteString("\n[commit]\n    gpgsign = true\n")
		b.WriteString("\n[tag]\n    gpgsign = true\n")
	}

	path := filepath.Join(gitDir, "config-"+name)
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func (m *Manager) writeMainGitConfig(gitDir string, names []string, defaultCtx string) error {
	var b strings.Builder

	// Include shared settings (delta, core, etc.)
	b.WriteString("[include]\n    path = config-shared\n\n")

	// Default identity (included unconditionally)
	if defaultCtx != "" {
		b.WriteString(fmt.Sprintf("[include]\n    path = config-%s\n\n", defaultCtx))
	}

	// includeIf for non-default contexts based on PROJECTS_DIR
	home, _ := os.UserHomeDir()
	for _, name := range names {
		if name == defaultCtx {
			continue
		}
		ctx, err := m.Load(name)
		if err != nil || ctx.Identity.Email == "" {
			continue
		}
		projectsDir, ok := ctx.Env["PROJECTS_DIR"]
		if !ok {
			continue
		}
		projectsDir = expandHome(projectsDir, home)
		b.WriteString(fmt.Sprintf("[includeIf \"gitdir:%s/**\"]\n    path = config-%s\n\n", projectsDir, name))
	}

	mainPath := filepath.Join(gitDir, "config")
	return os.WriteFile(mainPath, []byte(b.String()), 0o644)
}
