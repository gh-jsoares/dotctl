package orchestrator

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/gh-jsoares/dotctl/internal/config"
)

type Step struct {
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
	args := append([]string{"-R", "-d", stowDir, "-t", home}, packages...)
	cmd := exec.Command("stow", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
