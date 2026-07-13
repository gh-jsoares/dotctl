package cmd

import (
	"fmt"
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
			RunW: orchestrator.GitPullW,
			Enabled: func(cfg *config.Config) bool {
				return !syncNoPull && orchestrator.HasGitRepo(cfg)
			},
		},
		{
			ID:      "submodule-update",
			Name:    "submodule update",
			RunW:    orchestrator.SubmoduleUpdateW,
			Enabled: orchestrator.HasSubmodules,
		},
		{
			ID:          "nix-darwin",
			Name:        "nix-darwin switch",
			RunW:        orchestrator.NixDarwinSwitchW,
			Interactive: true,
			Enabled: func(cfg *config.Config) bool {
				return !syncDotfilesOnly && orchestrator.HasFlake(cfg)
			},
		},
		{
			ID:   "commit-flake-lock",
			Name: "commit flake.lock",
			RunW: orchestrator.CommitLockfileW,
			Enabled: func(cfg *config.Config) bool {
				return !syncDotfilesOnly && orchestrator.HasDirtyLockfile(cfg)
			},
		},
		{
			ID:          "stow",
			Name:        "stow dotfiles",
			RunW:        orchestrator.StowAllW,
			Interactive: true,
			Enabled:     orchestrator.HasStow,
		},
		{
			ID:   "refresh-context",
			Name: "refresh context env",
			Run:  orchestrator.RefreshContext,
			Enabled: func(cfg *config.Config) bool {
				mgr, err := context.NewManager()
				if err != nil {
					return false
				}
				current, _ := mgr.Current()
				return current != ""
			},
		},
		{
			ID:   "sheldon",
			Name: "sheldon lock",
			RunW: orchestrator.SheldonLockW,
			Enabled: func(cfg *config.Config) bool {
				return !syncDotfilesOnly && orchestrator.HasSheldon(cfg)
			},
		},
		{
			ID:   "mise",
			Name: "mise install",
			RunW: orchestrator.MiseInstallW,
			Enabled: func(cfg *config.Config) bool {
				return !syncDotfilesOnly && orchestrator.HasMise(cfg)
			},
		},
	}

	// Count enabled steps for progress
	enabledSteps := []orchestrator.Step{}
	for _, step := range steps {
		if step.Enabled(cfg) {
			enabledSteps = append(enabledSteps, step)
		}
	}

	ui.SectionWithCount("Core", len(enabledSteps))
	totalSteps := 0
	failedSteps := 0
	coreIdx := 0

	for _, step := range steps {
		if !step.Enabled(cfg) {
			continue
		}

		coreIdx++
		totalSteps++
		st := ui.StepStartWithCounter(step.Name, coreIdx, len(enabledSteps))

		if step.Interactive {
			// Interactive steps get raw terminal — no spinner, no pipe
			fmt.Println()
			if err := orchestrator.RunStep(step, cfg, nil); err != nil {
				st.Fail(err)
				failedSteps++
				ui.Summary(totalSteps, failedSteps, time.Since(syncStart))
				return fmt.Errorf("%s failed: %w", step.Name, err)
			}
			st.Success()
		} else {
			st.StartSpin()
			pipe := ui.NewPipeCmd(st)
			if err := orchestrator.RunStep(step, cfg, pipe.Writer()); err != nil {
				pipe.Flush()
				st.Fail(err)
				failedSteps++
				ui.Summary(totalSteps, failedSteps, time.Since(syncStart))
				return fmt.Errorf("%s failed: %w", step.Name, err)
			}
			pipe.Flush()
			st.Success()
		}
	}

	// Run plugins
	pluginRan, pluginFailed, err := runPluginsUI(cfg, "sync")
	totalSteps += pluginRan
	failedSteps += pluginFailed

	ui.Summary(totalSteps, failedSteps, time.Since(syncStart))

	// Check for updates once per day
	CheckForUpdate()

	if err != nil {
		return err
	}
	return nil
}

func runPlugins(cfg *config.Config, hook string) error {
	_, _, err := runPluginsUI(cfg, hook)
	return err
}

func runPluginsUI(cfg *config.Config, hook string) (int, int, error) {
	plugins, err := plugin.Discover(cfg)
	if err != nil {
		return 0, 0, fmt.Errorf("discovering plugins: %w", err)
	}
	if len(plugins) == 0 {
		return 0, 0, nil
	}

	if err := plugin.Validate(plugins); err != nil {
		return 0, 0, fmt.Errorf("validating plugins: %w", err)
	}

	filtered := plugin.FilterByHook(plugins, hook)
	if len(filtered) == 0 {
		return 0, 0, nil
	}

	var currentContext string
	if mgr, err := context.NewManager(); err == nil {
		currentContext, _ = mgr.Current()
	}

	enabled := plugin.EvaluateConditions(filtered, cfg, currentContext)
	if len(enabled) == 0 {
		return 0, 0, nil
	}

	ordered, err := plugin.ResolveOrder(enabled)
	if err != nil {
		return 0, 0, fmt.Errorf("resolving plugin order: %w", err)
	}

	ui.SectionWithCount("Plugins", len(ordered))
	ran := 0
	failed := 0

	for i, p := range ordered {
		ran++
		st := ui.StepStartWithCounter(p.Name, i+1, len(ordered))
		st.StartSpin()

		pipe := ui.NewPipeCmd(st)
		execErr := plugin.ExecuteWithWriter(p, hook, cfg, currentContext, pipe.Writer())
		pipe.Flush()

		if execErr != nil {
			if p.Options.ContinueOnError {
				st.Warn(execErr)
				continue
			}
			st.Fail(execErr)
			failed++
			return ran, failed, fmt.Errorf("plugin %s failed: %w", p.Name, execErr)
		}
		st.Success()
	}

	return ran, failed, nil
}
