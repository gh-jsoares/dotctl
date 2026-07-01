package cmd

import (
	"fmt"
	"os"

	"github.com/gh-jsoares/dotctl/internal/bootstrap"
	"github.com/gh-jsoares/dotctl/internal/config"
	"github.com/spf13/cobra"
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Bootstrap a fresh machine",
	Long:  "Install prerequisites, setup SSH, clone repos, run nix-darwin, stow dotfiles, and configure contexts. Safe to re-run (idempotent).",
	RunE:  runBootstrap,
}

func init() {
	rootCmd.AddCommand(bootstrapCmd)
}

func runBootstrap(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	opts := &bootstrap.Options{
		DotfilesRemote: cfg.Dotfiles.Remote,
		DotctlRemote:   cfg.Dotctl.Remote,
		DotfilesPath:   cfg.Dotfiles.Path,
		DotctlPath:     cfg.Dotctl.Path,
		SSHHosts:       cfg.SSH.Hosts,
		DefaultContext: "personal",
	}

	if opts.DotfilesRemote == "" {
		return fmt.Errorf("dotfiles.remote not configured in %s", config.DefaultConfigPath())
	}
	if opts.DotctlRemote == "" {
		return fmt.Errorf("dotctl.remote not configured in %s", config.DefaultConfigPath())
	}

	steps := bootstrap.Steps()
	for _, step := range steps {
		if step.Skip != nil && step.Skip(opts) {
			fmt.Fprintf(os.Stdout, "⊘ %s (already done)\n", step.Name)
			continue
		}

		fmt.Fprintf(os.Stdout, "▸ %s\n", step.Name)
		if err := step.Fn(opts); err != nil {
			return fmt.Errorf("%s: %w", step.Name, err)
		}
		fmt.Fprintf(os.Stdout, "✓ %s\n\n", step.Name)
	}

	// Set default context
	fmt.Fprintf(os.Stdout, "▸ Setting default context to %q\n", opts.DefaultContext)
	ctxDefaultCmd.SetArgs([]string{opts.DefaultContext})
	if err := ctxDefaultCmd.RunE(ctxDefaultCmd, []string{opts.DefaultContext}); err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠ could not set default context: %v\n", err)
	} else {
		fmt.Fprintf(os.Stdout, "✓ Default context: %s\n\n", opts.DefaultContext)
	}

	// Run doctor
	fmt.Fprintln(os.Stdout, "▸ Running doctor...")
	if err := runDoctor(cmd, nil); err != nil {
		fmt.Fprintf(os.Stderr, "\n⚠ Some doctor checks failed — review above.\n")
	}

	fmt.Fprintln(os.Stdout, "\nBootstrap complete.")
	fmt.Fprintln(os.Stdout, "Run 'eval \"$(dotctl shell-init zsh)\"' or restart your shell.")
	return nil
}
