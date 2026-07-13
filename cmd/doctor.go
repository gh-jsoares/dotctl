package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gh-jsoares/dotctl/internal/config"
	"github.com/gh-jsoares/dotctl/internal/context"
	"github.com/gh-jsoares/dotctl/internal/plugin"
	"github.com/gh-jsoares/dotctl/internal/ui"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Validate environment health",
	Long:  "Check that all symlinks, tools, contexts, and configurations are consistent and healthy.",
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

type check struct {
	name   string
	fn     func() error
}

func runDoctor(cmd *cobra.Command, args []string) error {
	checks := []check{
		{"state directory exists", checkStateDir},
		{"context is set", checkContextSet},
		{"env file exists", checkEnvFile},
		{"symlinks are valid", checkSymlinks},
		{"required tools installed", checkTools},
	}

	ui.Section("Health Checks")
	failed := 0
	for _, c := range checks {
		st := ui.StepStart(c.name)
		if err := c.fn(); err != nil {
			st.Fail(err)
			failed++
		} else {
			st.Success()
		}
	}

	// Run plugin doctor hooks
	cfg, err := config.Load()
	if err == nil {
		pluginFailed := runPluginDoctor(cfg)
		failed += pluginFailed
	}

	if failed > 0 {
		return fmt.Errorf("%d check(s) failed", failed)
	}
	return nil
}

func runPluginDoctor(cfg *config.Config) int {
	plugins, err := plugin.Discover(cfg)
	if err != nil || len(plugins) == 0 {
		return 0
	}

	filtered := plugin.FilterByHook(plugins, "doctor")
	if len(filtered) == 0 {
		return 0
	}

	var currentContext string
	if mgr, err := context.NewManager(); err == nil {
		currentContext, _ = mgr.Current()
	}

	enabled := plugin.EvaluateConditions(filtered, cfg, currentContext)
	if len(enabled) == 0 {
		return 0
	}

	ui.Section("Plugin Checks")
	failed := 0
	for _, p := range enabled {
		st := ui.StepStart(p.Name)
		if err := plugin.Execute(p, "doctor", cfg, currentContext); err != nil {
			st.Fail(err)
			failed++
		} else {
			st.Success()
		}
	}
	return failed
}

func checkStateDir() error {
	dir := config.StateDir()
	if _, err := os.Stat(dir); err != nil {
		return fmt.Errorf("missing %s", dir)
	}
	return nil
}

func checkContextSet() error {
	mgr, err := context.NewManager()
	if err != nil {
		return err
	}
	current, err := mgr.Current()
	if err != nil {
		return err
	}
	if current == "" {
		return fmt.Errorf("no context set (run: dotctl ctx <name> or dotctl ctx default <name>)")
	}
	return nil
}

func checkEnvFile() error {
	mgr, err := context.NewManager()
	if err != nil {
		return err
	}
	if _, err := os.Stat(mgr.EnvFilePath()); err != nil {
		return fmt.Errorf("missing %s (run: dotctl ctx <name>)", mgr.EnvFilePath())
	}
	return nil
}

func checkSymlinks() error {
	home, _ := os.UserHomeDir()
	links := []string{
		filepath.Join(home, ".aws"),
		filepath.Join(home, ".kube"),
		filepath.Join(home, ".config", "git", "config-current"),
	}

	for _, link := range links {
		info, err := os.Lstat(link)
		if err != nil {
			if os.IsNotExist(err) {
				continue // not all symlinks are required
			}
			return err
		}
		if info.Mode()&os.ModeSymlink == 0 {
			return fmt.Errorf("%s exists but is not a symlink", link)
		}
		// Verify target exists
		if _, err := os.Stat(link); err != nil {
			return fmt.Errorf("%s is a broken symlink", link)
		}
	}
	return nil
}

func checkTools() error {
	tools := []string{"nix", "darwin-rebuild", "stow", "mise", "tmux", "git", "op"}
	missing := []string{}

	for _, tool := range tools {
		if _, err := findExecutable(tool); err != nil {
			missing = append(missing, tool)
		}
	}

	if len(missing) > 0 {
		return fmt.Errorf("missing: %v", missing)
	}
	return nil
}

func findExecutable(name string) (string, error) {
	return exec.LookPath(name)
}
