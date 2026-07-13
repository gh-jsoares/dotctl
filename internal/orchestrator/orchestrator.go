package orchestrator

import (
	"bufio"
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gh-jsoares/dotctl/internal/config"
)

type Step struct {
	ID      string
	Name    string
	Run     func(cfg *config.Config) error
	Enabled func(cfg *config.Config) bool
}

func GitPull(cfg *config.Config) error {
	cmd := exec.Command("git", "-C", cfg.Dotfiles.Path, "pull", "--ff-only")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func SubmoduleUpdate(cfg *config.Config) error {
	sync := exec.Command("git", "-C", cfg.Dotfiles.Path, "submodule", "sync")
	sync.Stdout = os.Stdout
	sync.Stderr = os.Stderr
	if err := sync.Run(); err != nil {
		return err
	}
	cmd := exec.Command("git", "-C", cfg.Dotfiles.Path, "submodule", "update", "--init", "--depth", "1")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func HasSubmodules(cfg *config.Config) bool {
	modulesFile := filepath.Join(cfg.Dotfiles.Path, ".gitmodules")
	_, err := os.Stat(modulesFile)
	return err == nil
}

func HasGitRepo(cfg *config.Config) bool {
	gitDir := filepath.Join(cfg.Dotfiles.Path, ".git")
	_, err := os.Stat(gitDir)
	return err == nil
}

func CommitLockfile(cfg *config.Config) error {
	statusCmd := exec.Command("git", "-C", cfg.Dotfiles.Path, "status", "--porcelain", "flake.lock")
	out, err := statusCmd.Output()
	if err != nil || len(out) == 0 {
		return nil
	}

	addCmd := exec.Command("git", "-C", cfg.Dotfiles.Path, "add", "flake.lock")
	addCmd.Stdout = os.Stdout
	addCmd.Stderr = os.Stderr
	if err := addCmd.Run(); err != nil {
		return err
	}

	commitCmd := exec.Command("git", "-C", cfg.Dotfiles.Path, "commit", "-m", "chore: update flake.lock")
	commitCmd.Stdout = os.Stdout
	commitCmd.Stderr = os.Stderr
	if err := commitCmd.Run(); err != nil {
		return err
	}

	pushCmd := exec.Command("git", "-C", cfg.Dotfiles.Path, "push")
	pushCmd.Stdout = os.Stdout
	pushCmd.Stderr = os.Stderr
	return pushCmd.Run()
}

func HasDirtyLockfile(cfg *config.Config) bool {
	cmd := exec.Command("git", "-C", cfg.Dotfiles.Path, "status", "--porcelain", "flake.lock")
	out, err := cmd.Output()
	return err == nil && len(out) > 0
}

func NixDarwinSwitch(cfg *config.Config) error {
	flakePath := cfg.Dotfiles.Path
	flakeFile := filepath.Join(flakePath, "flake.nix")

	if _, err := os.Stat(flakeFile); err != nil {
		return fmt.Errorf("no flake.nix found at %s (skipping nix-darwin)", flakePath)
	}

	hostname := cfg.Machine
	if hostname == "" {
		hostname = "default"
	}
	ref := fmt.Sprintf("%s#%s", flakePath, hostname)

	cmd := exec.Command("sudo", "darwin-rebuild", "switch", "--flake", ref)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func StowAll(cfg *config.Config) error {
	stowDir := filepath.Join(cfg.Dotfiles.Path, "stow")
	if _, err := os.Stat(stowDir); err != nil {
		return fmt.Errorf("no stow/ directory found at %s (skipping)", stowDir)
	}

	entries, err := os.ReadDir(stowDir)
	if err != nil {
		return err
	}

	packages := []string{}
	for _, e := range entries {
		if e.IsDir() && !isHidden(e.Name()) {
			packages = append(packages, e.Name())
		}
	}

	if len(packages) == 0 {
		return nil
	}

	home, _ := os.UserHomeDir()

	// Dry-run first to detect conflicts
	args := append([]string{"-R", "-n", "-d", stowDir, "-t", home}, packages...)
	dryRun := exec.Command("stow", args...)
	var stderrBuf bytes.Buffer
	dryRun.Stderr = &stderrBuf
	dryRun.Run()

	conflicts := parseStowConflicts(stderrBuf.String())
	if len(conflicts) > 0 {
		resolved, err := resolveConflicts(conflicts, stowDir, home)
		if err != nil {
			return err
		}
		if !resolved {
			return fmt.Errorf("stow aborted due to unresolved conflicts")
		}
	}

	// Run stow for real
	realArgs := append([]string{"-R", "-d", stowDir, "-t", home}, packages...)
	cmd := exec.Command("stow", realArgs...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

type stowConflict struct {
	pkg    string
	source string
	target string
}

func parseStowConflicts(stderr string) []stowConflict {
	var conflicts []stowConflict
	for _, line := range strings.Split(stderr, "\n") {
		if !strings.Contains(line, "cannot stow") {
			continue
		}
		// Extract package name and target file
		// Format: "* cannot stow .dotfiles/stow/PKG/PATH over existing target PATH..."
		parts := strings.SplitN(line, "stow/", 2)
		if len(parts) < 2 {
			continue
		}
		rest := parts[1]
		// rest is like "lazygit/.config/lazygit/config.yml over existing target .config/lazygit/config.yml..."
		overIdx := strings.Index(rest, " over existing target ")
		if overIdx < 0 {
			continue
		}
		sourcePart := rest[:overIdx]
		targetPart := rest[overIdx+len(" over existing target "):]
		// Clean target (remove trailing "since neither...")
		if idx := strings.Index(targetPart, " since"); idx > 0 {
			targetPart = targetPart[:idx]
		}
		// Extract package name
		slashIdx := strings.Index(sourcePart, "/")
		pkg := sourcePart
		if slashIdx > 0 {
			pkg = sourcePart[:slashIdx]
		}
		conflicts = append(conflicts, stowConflict{
			pkg:    pkg,
			source: sourcePart,
			target: targetPart,
		})
	}
	return conflicts
}

func resolveConflicts(conflicts []stowConflict, stowDir, home string) (bool, error) {
	reader := bufio.NewReader(os.Stdin)
	fmt.Fprintf(os.Stdout, "\n    Found %d stow conflict(s):\n\n", len(conflicts))

	for _, c := range conflicts {
		targetPath := filepath.Join(home, c.target)
		fmt.Fprintf(os.Stdout, "    %s → %s\n", c.pkg, c.target)
		fmt.Fprintf(os.Stdout, "    [a]dopt (pull into stow) / [s]kip / [o]verwrite / [A]bort: ")

		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		switch strings.ToLower(input) {
		case "a", "adopt":
			// Move existing file into the stow package
			sourcePath := filepath.Join(stowDir, c.source)
			os.MkdirAll(filepath.Dir(sourcePath), 0o755)
			if err := os.Rename(targetPath, sourcePath); err != nil {
				return false, fmt.Errorf("adopting %s: %w", c.target, err)
			}
			fmt.Fprintf(os.Stdout, "    → adopted (review with git diff)\n\n")
		case "s", "skip":
			fmt.Fprintf(os.Stdout, "    → skipped\n\n")
			continue
		case "o", "overwrite":
			if err := os.Remove(targetPath); err != nil {
				return false, fmt.Errorf("removing %s: %w", c.target, err)
			}
			fmt.Fprintf(os.Stdout, "    → removed existing file\n\n")
		case "abort", "":
			return false, nil
		default:
			return false, nil
		}
	}

	return true, nil
}

func SheldonLock(cfg *config.Config) error {
	cmd := exec.Command("sheldon", "lock", "--update")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func MiseInstall(cfg *config.Config) error {
	cmd := exec.Command("mise", "install")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func HasFlake(cfg *config.Config) bool {
	flakeFile := filepath.Join(cfg.Dotfiles.Path, "flake.nix")
	_, err := os.Stat(flakeFile)
	return err == nil
}

func HasStow(cfg *config.Config) bool {
	stowDir := filepath.Join(cfg.Dotfiles.Path, "stow")
	_, err := os.Stat(stowDir)
	return err == nil
}

func HasSheldon(_ *config.Config) bool {
	_, err := exec.LookPath("sheldon")
	return err == nil
}

func HasMise(_ *config.Config) bool {
	_, err := exec.LookPath("mise")
	return err == nil
}

func isHidden(name string) bool {
	return len(name) > 0 && name[0] == '.'
}
