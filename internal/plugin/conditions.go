package plugin

import (
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gh-jsoares/dotctl/internal/config"
)

func EvaluateConditions(plugins []*Plugin, cfg *config.Config, currentContext string) []*Plugin {
	var enabled []*Plugin
	for _, p := range plugins {
		if checkConditions(p, cfg, currentContext) {
			enabled = append(enabled, p)
		}
	}
	return enabled
}

func checkConditions(p *Plugin, cfg *config.Config, currentContext string) bool {
	c := p.Conditions

	for _, path := range c.PathsExist {
		resolved := resolvePath(path, cfg.Dotfiles.Path)
		if _, err := os.Stat(resolved); err != nil {
			return false
		}
	}

	for _, bin := range c.BinariesExist {
		if _, err := exec.LookPath(bin); err != nil {
			return false
		}
	}

	for _, bin := range c.BinariesAbsent {
		if _, err := exec.LookPath(bin); err == nil {
			return false
		}
	}

	if len(c.Contexts) > 0 {
		found := false
		for _, ctx := range c.Contexts {
			if ctx == currentContext {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if c.Check != "" {
		cmd := exec.Command("sh", "-c", c.Check)
		cmd.Dir = cfg.Dotfiles.Path
		if err := cmd.Run(); err != nil {
			return false
		}
	}

	return true
}

func resolvePath(path, dotfilesPath string) string {
	if filepath.IsAbs(path) {
		return path
	}
	if len(path) > 1 && path[:2] == "~/" {
		home, _ := os.UserHomeDir()
		return filepath.Join(home, path[2:])
	}
	return filepath.Join(dotfilesPath, path)
}
