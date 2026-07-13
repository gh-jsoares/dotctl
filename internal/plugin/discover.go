package plugin

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/gh-jsoares/dotctl/internal/config"
)

func Discover(cfg *config.Config) ([]*Plugin, error) {
	disabled := make(map[string]bool)
	for _, name := range cfg.Plugins.Disabled {
		disabled[name] = true
	}

	// Discover builtins
	builtinPlugins, err := discoverFromDir(extractedBuiltinsDir())
	if err != nil {
		builtinPlugins = nil
	}

	// Discover user plugins
	userDir := filepath.Join(cfg.Dotfiles.Path, ".dotctl", "plugins")
	userPlugins, err := discoverFromDir(userDir)
	if err != nil {
		return nil, err
	}

	// Merge: user plugins override builtins with same name
	merged := make(map[string]*Plugin)
	for _, p := range builtinPlugins {
		p.Builtin = true
		merged[p.Name] = p
	}
	for _, p := range userPlugins {
		merged[p.Name] = p
	}

	// Filter disabled and collect
	var plugins []*Plugin
	for _, p := range merged {
		if disabled[p.Name] {
			continue
		}
		plugins = append(plugins, p)
	}

	sort.Slice(plugins, func(i, j int) bool {
		return plugins[i].Ordering.Priority < plugins[j].Ordering.Priority
	})

	return plugins, nil
}

func discoverFromDir(dir string) ([]*Plugin, error) {
	if dir == "" {
		return nil, nil
	}
	if _, err := os.Stat(dir); err != nil {
		return nil, nil
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, fmt.Errorf("reading plugins dir: %w", err)
	}

	var plugins []*Plugin
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}

		manifest := filepath.Join(dir, e.Name(), "plugin.toml")
		if _, err := os.Stat(manifest); err != nil {
			continue
		}

		p, err := Parse(manifest)
		if err != nil {
			return nil, fmt.Errorf("parsing plugin %q: %w", e.Name(), err)
		}

		p.Dir = filepath.Join(dir, e.Name())
		if p.Name == "" {
			p.Name = e.Name()
		}

		plugins = append(plugins, p)
	}

	return plugins, nil
}

// extractedBuiltinsDir extracts embedded builtins to cache and returns the path.
func extractedBuiltinsDir() string {
	dir, err := extractBuiltins()
	if err != nil {
		return ""
	}
	return dir
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
