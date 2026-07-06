package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gh-jsoares/dotctl/internal/bootstrap"
	"github.com/gh-jsoares/dotctl/internal/config"
	"github.com/spf13/cobra"
)

var (
	bootstrapDotfilesRemote string
	bootstrapDotfilesPath   string
	bootstrapDefaultCtx     string
)

var bootstrapCmd = &cobra.Command{
	Use:   "bootstrap",
	Short: "Bootstrap a fresh machine",
	Long:  "Install prerequisites, setup SSH, clone repos, run nix-darwin, stow dotfiles, and configure contexts. Safe to re-run (idempotent).",
	RunE:  runBootstrap,
}

func init() {
	rootCmd.AddCommand(bootstrapCmd)
	bootstrapCmd.Flags().StringVar(&bootstrapDotfilesRemote, "dotfiles-remote", "", "git remote for dotfiles repo")
	bootstrapCmd.Flags().StringVar(&bootstrapDotfilesPath, "dotfiles-path", "", "local path for dotfiles repo")
	bootstrapCmd.Flags().StringVar(&bootstrapDefaultCtx, "default-context", "personal", "default context to set after bootstrap")
}

func runBootstrap(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	opts := &bootstrap.Options{
		DotfilesRemote: coalesce(bootstrapDotfilesRemote, cfg.Dotfiles.Remote),
		DotfilesPath:   coalesce(bootstrapDotfilesPath, cfg.Dotfiles.Path),
		DefaultContext: bootstrapDefaultCtx,
		Machine:        cfg.Machine,
	}

	reader := bufio.NewReader(os.Stdin)

	// Cache sudo credentials upfront and keep them alive
	fmt.Fprintln(os.Stdout, "▸ Requesting sudo access (needed for nix-darwin and Homebrew)...")
	sudoCmd := exec.Command("sudo", "-v")
	sudoCmd.Stdin = os.Stdin
	sudoCmd.Stdout = os.Stdout
	sudoCmd.Stderr = os.Stderr
	if err := sudoCmd.Run(); err != nil {
		return fmt.Errorf("sudo: %w", err)
	}
	stopKeepAlive := make(chan struct{})
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				exec.Command("sudo", "-n", "-v").Run()
			case <-stopKeepAlive:
				return
			}
		}
	}()
	defer close(stopKeepAlive)

	steps := bootstrap.Steps()
	configWritten := false
	for _, step := range steps {
		if step.Skip != nil && step.Skip(opts) {
			fmt.Fprintf(os.Stdout, "⊘ %s (already done)\n", step.Name)
			continue
		}

		fmt.Fprintf(os.Stdout, "▸ %s\n", step.Name)
		if err := step.Fn(opts, reader); err != nil {
			return fmt.Errorf("%s: %w", step.Name, err)
		}
		fmt.Fprintf(os.Stdout, "✓ %s\n\n", step.Name)

		// Write config after dotfiles clone succeeds (not before)
		if !configWritten && step.Name == "Clone dotfiles repo" {
			if err := maybeWriteConfig(cfg, opts); err != nil {
				fmt.Fprintf(os.Stderr, "⚠ could not save config: %v\n", err)
			}
			configWritten = true
		}
	}

	// Write config even if clone was skipped (already cloned)
	if !configWritten {
		if err := maybeWriteConfig(cfg, opts); err != nil {
			fmt.Fprintf(os.Stderr, "⚠ could not save config: %v\n", err)
		}
	}

	// Persist machine name if it was set during nix-darwin step
	if opts.Machine != "" {
		updateConfigMachine(opts.Machine)
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



func coalesce(values ...string) string {
	for _, v := range values {
		if v != "" {
			return v
		}
	}
	return ""
}

func parseGitRemote(remote string) (owner, repo string) {
	// Handle git@host:owner/repo.git and https://host/owner/repo.git
	remote = strings.TrimSuffix(remote, ".git")
	if idx := strings.LastIndex(remote, ":"); idx != -1 && !strings.Contains(remote, "://") {
		remote = remote[idx+1:]
	} else if idx := strings.LastIndex(remote, "/"); idx != -1 {
		// Take last two path components
		parts := strings.Split(remote, "/")
		if len(parts) >= 2 {
			return parts[len(parts)-2], parts[len(parts)-1]
		}
	}
	parts := strings.Split(remote, "/")
	if len(parts) == 2 {
		return parts[0], parts[1]
	}
	return "", ""
}

func maybeWriteConfig(cfg *config.Config, opts *bootstrap.Options) error {
	configPath := config.DefaultConfigPath()
	if _, err := os.Stat(configPath); err == nil {
		return nil
	}

	if err := os.MkdirAll(filepath.Dir(configPath), 0o755); err != nil {
		return err
	}

	var content string
	if opts.Machine != "" {
		content = fmt.Sprintf("machine = %q\n\n", opts.Machine)
	}
	content += fmt.Sprintf(`[dotfiles]
path = %q
remote = %q
`, opts.DotfilesPath, opts.DotfilesRemote)

	// Derive dotctl remote from same SSH host as dotfiles
	dotctlRemote := deriveDotctlRemote(opts.DotfilesRemote)
	if dotctlRemote != "" {
		content += fmt.Sprintf("\n[dotctl]\nremote = %q\n", dotctlRemote)
	}

	return os.WriteFile(configPath, []byte(content), 0o644)
}

func updateConfigMachine(machine string) {
	configPath := config.DefaultConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		return
	}
	content := string(data)
	if strings.Contains(content, "machine") {
		return
	}
	content = fmt.Sprintf("machine = %q\n\n%s", machine, content)
	os.WriteFile(configPath, []byte(content), 0o644)
}

func deriveDotctlRemote(dotfilesRemote string) string {
	// git@host:owner/repo.git → git@host:owner/dotctl.git
	if !strings.Contains(dotfilesRemote, ":") {
		return ""
	}
	parts := strings.SplitN(dotfilesRemote, ":", 2)
	host := parts[0]
	ownerRepo := parts[1]
	owner, _, _ := strings.Cut(ownerRepo, "/")
	if owner == "" {
		return ""
	}
	return fmt.Sprintf("%s:%s/dotctl.git", host, owner)
}
