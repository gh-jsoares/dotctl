package plugin

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/gh-jsoares/dotctl/internal/config"
)

func Execute(p *Plugin, hook string, cfg *config.Config, currentContext string) error {
	var script string
	switch hook {
	case "sync":
		script = p.Hooks.Sync
	case "bootstrap":
		script = p.Hooks.Bootstrap
	case "doctor":
		script = p.Hooks.Doctor
	default:
		return fmt.Errorf("unknown hook %q", hook)
	}

	if script == "" {
		return nil
	}

	scriptPath := filepath.Join(p.Dir, script)

	var cmd *exec.Cmd
	if p.Options.Timeout > 0 {
		ctx, cancel := context.WithTimeout(context.Background(), time.Duration(p.Options.Timeout)*time.Second)
		defer cancel()
		if p.Options.Sudo {
			cmd = exec.CommandContext(ctx, "sudo", scriptPath)
		} else {
			cmd = exec.CommandContext(ctx, scriptPath)
		}
	} else {
		if p.Options.Sudo {
			cmd = exec.Command("sudo", scriptPath)
		} else {
			cmd = exec.Command(scriptPath)
		}
	}

	cmd.Dir = resolveWorkdir(p, cfg)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	cmd.Env = buildEnv(p, cfg, currentContext, hook)

	return cmd.Run()
}

func resolveWorkdir(p *Plugin, cfg *config.Config) string {
	if p.Options.Workdir == "" {
		return p.Dir
	}
	return resolvePath(p.Options.Workdir, cfg.Dotfiles.Path)
}

func buildEnv(p *Plugin, cfg *config.Config, currentContext, hook string) []string {
	env := os.Environ()
	env = append(env,
		"DOTCTL_DOTFILES_PATH="+cfg.Dotfiles.Path,
		"DOTCTL_PLUGIN_DIR="+p.Dir,
		"DOTCTL_CONTEXT="+currentContext,
		"DOTCTL_MACHINE="+cfg.Machine,
		"DOTCTL_HOOK="+hook,
	)
	for k, v := range p.Options.Env {
		home, _ := os.UserHomeDir()
		if len(v) > 1 && v[:2] == "~/" {
			v = filepath.Join(home, v[2:])
		}
		env = append(env, k+"="+v)
	}
	return env
}
