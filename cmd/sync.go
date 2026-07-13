package cmd

import (
	"fmt"
	"os"
	"time"

	"github.com/gh-jsoares/dotctl/internal/config"
	"github.com/gh-jsoares/dotctl/internal/context"
	"github.com/gh-jsoares/dotctl/internal/orchestrator"
	"github.com/gh-jsoares/dotctl/internal/plugin"
	"github.com/gh-jsoares/dotctl/internal/ui"
	"github.com/spf13/cobra"
)

var syncNoPull bool
var syncDotfilesOnly bool

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync environment to desired state",
	Long:  "Rebuild nix-darwin, re-stow dotfiles, and run plugins. Skips steps that aren't applicable.",
	RunE:  runSync,
}

func init() {
	syncCmd.Flags().BoolVar(&syncNoPull, "no-pull", false, "skip git pull before syncing")
	syncCmd.Flags().BoolVar(&syncDotfilesOnly, "dotfiles-only", false, "only pull, stow, and reload (skip nix, sheldon, mise)")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	syncStart := time.Now()

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	steps := []orchestrator.Step{
		{
			ID:   "git-pull",
			Name: "git pull",
			Run:  orchestrator.GitPull,
			Enabled: func(cfg *config.Config) bool {
				return !syncNoPull && orchestrator.HasGitRepo(cfg)
			},
		},
		{
			ID:      "submodule-update",
			Name:    "submodule update",
			Run:     orchestrator.SubmoduleUpdate,
			Enabled: orchestrator.HasSubmodules,
		},
		{
			ID:   "nix-darwin",
			Name: "nix-darwin switch",
			Run:  orchestrator.NixDarwinSwitch,
			Enabled: func(cfg *config.Config) bool {
				return !syncDotfilesOnly && orchestrator.HasFlake(cfg)
			},
		},
		{
			ID:   "commit-flake-lock",
			Name: "commit flake.lock",
			Run:  orchestrator.CommitLockfile,
			Enabled: func(cfg *config.Config) bool {
				return !syncDotfilesOnly && orchestrator.HasDirtyLockfile(cfg)
			},
		},
		{
			ID:      "stow",
			Name:    "stow dotfiles",
			Run:     orchestrator.StowAll,
			Enabled: orchestrator.HasStow,
		},
		{
			ID:   "sheldon",
			Name: "sheldon lock",
			Run:  orchestrator.SheldonLock,
			Enabled: func(cfg *config.Config) bool {
				return !syncDotfilesOnly && orchestrator.HasSheldon(cfg)
			},
		},
		{
			ID:   "mise",
			Name: "mise install",
			Run:  orchestrator.MiseInstall,
			Enabled: func(cfg *config.Config) bool {
				return !syncDotfilesOnly && orchestrator.HasMise(cfg)
			},
		},
	}

	// Run core steps
	ui.Section("Core")
	totalSteps := 0
	failedSteps := 0

	for _, step := range steps {
		if !step.Enabled(cfg) {
			st := ui.StepStart(step.Name)
			st.Skip("not applicable")
			continue
		}

		totalSteps++
		st := ui.StepStart(step.Name)
		if err := step.Run(cfg); err != nil {
			st.Fail(err)
			failedSteps++
			ui.Summary(totalSteps, failedSteps, time.Since(syncStart))
			return fmt.Errorf("%s failed: %w", step.Name, err)
		}
		st.Success()
	}

	// Run plugins
	pluginFailed, err := runPluginsUI(cfg, "sync")
	if err != nil {
		failedSteps += pluginFailed
		ui.Summary(totalSteps+pluginFailed, failedSteps, time.Since(syncStart))
		return err
	}
	totalSteps += pluginFailed

	ui.Summary(totalSteps, failedSteps, time.Since(syncStart))
	return nil
}

func runPlugins(cfg *config.Config, hook string) error {
	_, err := runPluginsUI(cfg, hook)
	return err
}

func runPluginsUI(cfg *config.Config, hook string) (int, error) {
	plugins, err := plugin.Discover(cfg)
	if err != nil {
		return 0, fmt.Errorf("discovering plugins: %w", err)
	}
	if len(plugins) == 0 {
		return 0, nil
	}

	if err := plugin.Validate(plugins); err != nil {
		return 0, fmt.Errorf("validating plugins: %w", err)
	}

	filtered := plugin.FilterByHook(plugins, hook)
	if len(filtered) == 0 {
		return 0, nil
	}

	var currentContext string
	if mgr, err := context.NewManager(); err == nil {
		currentContext, _ = mgr.Current()
	}

	enabled := plugin.EvaluateConditions(filtered, cfg, currentContext)
	if len(enabled) == 0 {
		return 0, nil
	}

	ordered, err := plugin.ResolveOrder(enabled)
	if err != nil {
		return 0, fmt.Errorf("resolving plugin order: %w", err)
	}

	ui.Section("Plugins")
	ran := 0

	for _, p := range ordered {
		ran++
		st := ui.StepStart(p.Name)

		spinner := ui.NewSpinner(os.Stderr, p.Name)
		spinner.Start()
		execErr := plugin.Execute(p, hook, cfg, currentContext)
		spinner.Stop()

		if execErr != nil {
			if p.Options.ContinueOnError {
				st.Warn(execErr)
				continue
			}
			st.Fail(execErr)
			return ran, fmt.Errorf("plugin %s failed: %w", p.Name, execErr)
		}
		st.Success()
	}

	return ran, nil
}
