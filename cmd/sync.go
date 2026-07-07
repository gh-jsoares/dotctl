package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"syscall"

	"github.com/gh-jsoares/dotctl/internal/config"
	"github.com/gh-jsoares/dotctl/internal/orchestrator"
	"github.com/spf13/cobra"
)

var syncNoPull bool

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync environment to desired state",
	Long:  "Rebuild nix-darwin, re-stow dotfiles, and run mise install. Skips steps that aren't applicable.",
	RunE:  runSync,
}

func init() {
	syncCmd.Flags().BoolVar(&syncNoPull, "no-pull", false, "skip git pull before syncing")
	rootCmd.AddCommand(syncCmd)
}

func runSync(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	steps := []orchestrator.Step{
		{
			Name:    "git pull",
			Run:     orchestrator.GitPull,
			Enabled: func(cfg *config.Config) bool {
				return !syncNoPull && orchestrator.HasGitRepo(cfg)
			},
		},
		{
			Name:    "nix-darwin switch",
			Run:     orchestrator.NixDarwinSwitch,
			Enabled: orchestrator.HasFlake,
		},
		{
			Name:    "stow dotfiles",
			Run:     orchestrator.StowAll,
			Enabled: orchestrator.HasStow,
		},
		{
			Name:    "sheldon lock",
			Run:     orchestrator.SheldonLock,
			Enabled: orchestrator.HasSheldon,
		},
		{
			Name:    "mise install",
			Run:     orchestrator.MiseInstall,
			Enabled: orchestrator.HasMise,
		},
	}

	for _, step := range steps {
		if !step.Enabled(cfg) {
			fmt.Fprintf(os.Stdout, "⊘ %s (skipped — not available)\n", step.Name)
			continue
		}

		fmt.Fprintf(os.Stdout, "▸ %s\n", step.Name)
		if err := step.Run(cfg); err != nil {
			return fmt.Errorf("%s failed: %w", step.Name, err)
		}
		fmt.Fprintf(os.Stdout, "✓ %s\n", step.Name)
	}

	fmt.Fprintln(os.Stdout, "\nSync complete. Reloading shell...")
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		shellPath = "/bin/zsh"
	}
	return syscall.Exec(shellPath, []string{"-" + filepath.Base(shellPath)}, os.Environ())
}
