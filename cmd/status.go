package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/gh-jsoares/dotctl/internal/config"
	"github.com/gh-jsoares/dotctl/internal/context"
	"github.com/gh-jsoares/dotctl/internal/plugin"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show current environment state",
	Long:  "Display current context, git state, and what sync would do.",
	RunE:  runStatus,
}

func init() {
	rootCmd.AddCommand(statusCmd)
}

var (
	statusGreen  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	statusYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	statusDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	statusBold   = lipgloss.NewStyle().Bold(true)
	statusHeader = lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("4"))
)

func runStatus(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	// Context
	fmt.Fprintf(os.Stdout, "\n %s\n", statusHeader.Render("Context"))
	mgr, err := context.NewManager()
	if err != nil {
		fmt.Fprintf(os.Stdout, "  %s\n", statusDim.Render("unavailable"))
	} else {
		current, _ := mgr.Current()
		if current == "" {
			fmt.Fprintf(os.Stdout, "  %s\n", statusYellow.Render("none set"))
		} else {
			fmt.Fprintf(os.Stdout, "  %s %s\n", statusBold.Render(current), statusDim.Render("(active)"))
		}

		if expected, actual := mgr.CheckMismatch(); expected != "" {
			statusRed := lipgloss.NewStyle().Foreground(lipgloss.Color("1"))
			fmt.Fprintf(os.Stdout, "  %s\n", statusRed.Render(fmt.Sprintf("⚠ CWD belongs to %q but active context is %q", expected, actual)))
		}
	}

	// Git state
	fmt.Fprintf(os.Stdout, "\n %s\n", statusHeader.Render("Dotfiles"))
	printGitStatus(cfg)

	// Sync preview
	fmt.Fprintf(os.Stdout, "\n %s\n", statusHeader.Render("Sync would run"))
	printSyncPreview(cfg)

	// Plugins
	fmt.Fprintf(os.Stdout, "\n %s\n", statusHeader.Render("Plugins"))
	printPluginStatus(cfg)

	fmt.Fprintln(os.Stdout)
	return nil
}

func printGitStatus(cfg *config.Config) {
	dotfiles := cfg.Dotfiles.Path

	// Branch
	branch, err := gitOutput(dotfiles, "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		fmt.Fprintf(os.Stdout, "  %s\n", statusDim.Render("not a git repo"))
		return
	}
	fmt.Fprintf(os.Stdout, "  branch: %s\n", statusBold.Render(branch))

	// Ahead/behind
	revs, err := gitOutput(dotfiles, "rev-list", "--left-right", "--count", "HEAD...@{upstream}")
	if err == nil {
		parts := strings.Fields(revs)
		if len(parts) == 2 {
			ahead, behind := parts[0], parts[1]
			if ahead != "0" || behind != "0" {
				fmt.Fprintf(os.Stdout, "  %s ahead, %s behind remote\n", statusYellow.Render(ahead), statusYellow.Render(behind))
			} else {
				fmt.Fprintf(os.Stdout, "  %s\n", statusGreen.Render("up to date with remote"))
			}
		}
	}

	// Dirty files
	status, _ := gitOutput(dotfiles, "status", "--porcelain")
	if status == "" {
		fmt.Fprintf(os.Stdout, "  %s\n", statusGreen.Render("clean working tree"))
	} else {
		lines := strings.Split(strings.TrimSpace(status), "\n")
		fmt.Fprintf(os.Stdout, "  %s\n", statusYellow.Render(fmt.Sprintf("%d uncommitted change(s)", len(lines))))
	}
}

func printSyncPreview(cfg *config.Config) {
	type previewStep struct {
		name    string
		enabled bool
	}

	preview := []previewStep{
		{"git pull", hasGitRepo(cfg)},
		{"submodule update", hasSubmodules(cfg)},
		{"nix-darwin switch", hasFlake(cfg)},
		{"commit flake.lock", hasDirtyLockfile(cfg)},
		{"stow dotfiles", hasStow(cfg)},
		{"sheldon lock", hasSheldon(cfg)},
		{"mise install", hasMise(cfg)},
	}

	for _, s := range preview {
		if s.enabled {
			fmt.Fprintf(os.Stdout, "  %s %s\n", statusGreen.Render("▸"), s.name)
		} else {
			fmt.Fprintf(os.Stdout, "  %s %s\n", statusDim.Render("·"), statusDim.Render(s.name))
		}
	}
}

func printPluginStatus(cfg *config.Config) {
	plugins, err := plugin.Discover(cfg)
	if err != nil || len(plugins) == 0 {
		fmt.Fprintf(os.Stdout, "  %s\n", statusDim.Render("none"))
		return
	}

	var currentContext string
	if mgr, err := context.NewManager(); err == nil {
		currentContext, _ = mgr.Current()
	}

	for _, p := range plugins {
		enabled := plugin.EvaluateConditions([]*plugin.Plugin{p}, cfg, currentContext)
		if len(enabled) > 0 {
			fmt.Fprintf(os.Stdout, "  %s %s\n", statusGreen.Render("▸"), p.Name)
		} else {
			fmt.Fprintf(os.Stdout, "  %s %s %s\n", statusDim.Render("·"), statusDim.Render(p.Name), statusDim.Render("(skipped)"))
		}
	}
}

func gitOutput(dir string, args ...string) (string, error) {
	fullArgs := append([]string{"-C", dir}, args...)
	out, err := exec.Command("git", fullArgs...).Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

func hasGitRepo(cfg *config.Config) bool {
	_, err := os.Stat(filepath.Join(cfg.Dotfiles.Path, ".git"))
	return err == nil
}

func hasSubmodules(cfg *config.Config) bool {
	_, err := os.Stat(filepath.Join(cfg.Dotfiles.Path, ".gitmodules"))
	return err == nil
}

func hasFlake(cfg *config.Config) bool {
	_, err := os.Stat(filepath.Join(cfg.Dotfiles.Path, "flake.nix"))
	return err == nil
}

func hasDirtyLockfile(cfg *config.Config) bool {
	cmd := exec.Command("git", "-C", cfg.Dotfiles.Path, "status", "--porcelain", "flake.lock")
	out, err := cmd.Output()
	return err == nil && len(out) > 0
}

func hasStow(cfg *config.Config) bool {
	_, err := os.Stat(filepath.Join(cfg.Dotfiles.Path, "stow"))
	return err == nil
}

func hasSheldon(_ *config.Config) bool {
	_, err := exec.LookPath("sheldon")
	return err == nil
}

func hasMise(_ *config.Config) bool {
	_, err := exec.LookPath("mise")
	return err == nil
}
