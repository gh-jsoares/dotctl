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
var syncDotfilesOnly bool

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync environment to desired state",
	Long:  "Rebuild nix-darwin, re-stow dotfiles, and run mise install. Skips steps that aren't applicable.",
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
			Name:    "git pull",
			Run:     orchestrator.GitPull,
			Enabled: func(cfg *config.Config) bool {
				return !syncNoPull && orchestrator.HasGitRepo(cfg)
			},
		},
		{
			Name:    "submodule update",
			Run:     orchestrator.SubmoduleUpdate,
			Enabled: orchestrator.HasSubmodules,
		},
		{
			Name: "nix-darwin switch",
			Run:  orchestrator.NixDarwinSwitch,
			Enabled: func(cfg *config.Config) bool {
				return !syncDotfilesOnly && orchestrator.HasFlake(cfg)
			},
		},
		{
			Name: "commit flake.lock",
			Run:  orchestrator.CommitLockfile,
			Enabled: func(cfg *config.Config) bool {
				return !syncDotfilesOnly && orchestrator.HasDirtyLockfile(cfg)
			},
		},
		{
			Name:    "stow dotfiles",
			Run:     orchestrator.StowAll,
			Enabled: orchestrator.HasStow,
		},
		{
			Name: "sheldon lock",
			Run:  orchestrator.SheldonLock,
			Enabled: func(cfg *config.Config) bool {
				return !syncDotfilesOnly && orchestrator.HasSheldon(cfg)
			},
		},
		{
			Name: "mise install",
			Run:  orchestrator.MiseInstall,
			Enabled: func(cfg *config.Config) bool {
				return !syncDotfilesOnly && orchestrator.HasMise(cfg)
			},
		},
		{
			Name:    "simple-bar-server",
			Run:     orchestrator.SimpleBarServer,
			Enabled: orchestrator.HasSimpleBarServer,
		},
		{
			Name:    "tmux reload",
			Run:     orchestrator.TmuxReload,
			Enabled: orchestrator.HasTmux,
		},
		{
			Name:    "aerospace reload",
			Run:     orchestrator.AerospaceReload,
			Enabled: orchestrator.HasAerospace,
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

	fmt.Fprintln(os.Stdout, "\n✓ Sync complete.")
	fmt.Fprintln(os.Stdout, "\n  Übersicht: set widget folder to ~/dotfilesv2/dotfiles/ubersicht/widgets")
	fmt.Fprintln(os.Stdout, "             enable 'Launch at Login' and set shell to '/bin/bash -l'")
	fmt.Fprintln(os.Stdout, "\nReloading shell...")
	shellPath := os.Getenv("SHELL")
	if shellPath == "" {
		shellPath = "/bin/zsh"
	}
	return syscall.Exec(shellPath, []string{"-" + filepath.Base(shellPath)}, os.Environ())
}
