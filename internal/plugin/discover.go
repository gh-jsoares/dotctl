package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/gh-jsoares/dotctl/internal/config"
)

func Discover(cfg *config.Config) ([]*Plugin, error) {
	pluginsDir := filepath.Join(cfg.Dotfiles.Path, ".dotctl", "plugins")

	if _, err := os.Stat(pluginsDir); err != nil {
		return nil, nil
	}

	entries, err := os.ReadDir(pluginsDir)
	if err != nil {
		return nil, fmt.Errorf("reading plugins dir: %w", err)
	}

	var plugins []*Plugin
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		manifest := filepath.Join(pluginsDir, e.Name(), "plugin.toml")
		if _, err := os.Stat(manifest); err != nil {
			continue
		}

		p, err := Parse(manifest)
		if err != nil {
			return nil, fmt.Errorf("parsing plugin %q: %w", e.Name(), err)
		}

		p.Dir = filepath.Join(pluginsDir, e.Name())
		if p.Name == "" {
			p.Name = e.Name()
		}

		plugins = append(plugins, p)
	}

	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Ordering.Priority < plugins[j].Ordering.Priority
	})

	return plugins, nil
}

func FilterByHook(plugins []*Plugin, hook string) []*Plugin {
	var filtered []*Plugin
	for _, p := range plugins {
		var has bool
		switch hook {
		case "sync":
			has = p.Hooks.Sync != ""
		case "bootstrap":
			has = p.Hooks.Bootstrap != ""
		case "doctor":
			has = p.Hooks.Doctor != ""
		}
		if has {
			filtered = append(filtered, p)
		}
	}
	return filtered
}

func Validate(plugins []*Plugin) error {
	for _, p := range plugins {
		if p.Name == "" {
			return fmt.Errorf("plugin at %s: missing name", p.Dir)
		}
		scripts := map[string]string{
			"sync":      p.Hooks.Sync,
			"bootstrap": p.Hooks.Bootstrap,
			"doctor":    p.Hooks.Doctor,
		}
		for hook, script := range scripts {
			if script == "" {
				continue
			}
			path := filepath.Join(p.Dir, script)
			info, err := os.Stat(path)
			if err != nil {
				return fmt.Errorf("plugin %q: %s script %q not found", p.Name, hook, script)
			}
			if info.Mode()&0111 == 0 {
				return fmt.Errorf("plugin %q: %s script %q is not executable", p.Name, hook, script)
			}
		}
	}
	return nil
}
