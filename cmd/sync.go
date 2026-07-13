package cmd

import (
	"fmt"
	"os"

	"github.com/gh-jsoares/dotctl/internal/config"
	"github.com/gh-jsoares/dotctl/internal/context"
	"github.com/gh-jsoares/dotctl/internal/orchestrator"
	"github.com/gh-jsoares/dotctl/internal/plugin"
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
			ID:   "submodule-update",
			Name: "submodule update",
			Run:  orchestrator.SubmoduleUpdate,
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
	for _, step := range steps {
		if !step.Enabled(cfg) {
			fmt.Fprintf(os.Stdout, "⊘ %s (skipped)\n", step.Name)
			continue
		}

		fmt.Fprintf(os.Stdout, "▸ %s\n", step.Name)
		if err := step.Run(cfg); err != nil {
			return fmt.Errorf("%s failed: %w", step.Name, err)
		}
		fmt.Fprintf(os.Stdout, "✓ %s\n", step.Name)
	}

	// Run plugins
	if err := runPlugins(cfg, "sync"); err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, "\n✓ Sync complete.")
	return nil
}

func runPlugins(cfg *config.Config, hook string) error {
	plugins, err := plugin.Discover(cfg)
	if err != nil {
		return fmt.Errorf("discovering plugins: %w", err)
	}
	if len(plugins) == 0 {
		return nil
	}

	if err := plugin.Validate(plugins); err != nil {
		return fmt.Errorf("validating plugins: %w", err)
	}

	filtered := plugin.FilterByHook(plugins, hook)
	if len(filtered) == 0 {
		return nil
	}

	var currentContext string
	if mgr, err := context.NewManager(); err == nil {
		currentContext, _ = mgr.Current()
	}

	enabled := plugin.EvaluateConditions(filtered, cfg, currentContext)

	ordered, err := plugin.ResolveOrder(enabled)
	if err != nil {
		return fmt.Errorf("resolving plugin order: %w", err)
	}

	for _, p := range ordered {
		fmt.Fprintf(os.Stdout, "▸ %s\n", p.Name)
		if err := plugin.Execute(p, hook, cfg, currentContext); err != nil {
			if p.Options.ContinueOnError {
				fmt.Fprintf(os.Stderr, "  ⚠ %s failed: %v (continuing)\n", p.Name, err)
				continue
			}
			return fmt.Errorf("plugin %s failed: %w", p.Name, err)
		}
		fmt.Fprintf(os.Stdout, "✓ %s\n", p.Name)
	}

	return nil
}
