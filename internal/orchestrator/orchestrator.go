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

func GrimoireInstall(cfg *config.Config) error {
	home, _ := os.UserHomeDir()
	installDir := filepath.Join(home, ".local", "bin")
	os.MkdirAll(installDir, 0755)
	cmd := exec.Command("bash", "-c", "curl -sL https://raw.githubusercontent.com/gh-jsoares/grimoire/main/install.sh | GRIMOIRE_INSTALL_DIR="+installDir+" sh")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return err
	}
	// Remove quarantine attribute so macOS doesn't show a permission popup
	exec.Command("xattr", "-d", "com.apple.quarantine", filepath.Join(installDir, "grimoire")).Run()
	return nil
}

func HasGrimoireConfig(cfg *config.Config) bool {
	grimoireDir := filepath.Join(cfg.Dotfiles.Path, "stow", "grimoire")
	_, err := os.Stat(grimoireDir)
	return err == nil
}

func AerospaceReload(cfg *config.Config) error {
	cmd := exec.Command("aerospace", "reload-config")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠ aerospace reload failed (not running?): %v\n", err)
	}
	return nil
}

func TmuxReload(cfg *config.Config) error {
	if os.Getenv("TMUX") == "" {
		return nil
	}
	cmd := exec.Command("tmux", "source-file", filepath.Join(os.Getenv("HOME"), ".config", "tmux", "tmux.conf"))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "  ⚠ tmux reload failed: %v\n", err)
	}
	return nil
}

func HasTmux(_ *config.Config) bool {
	return os.Getenv("TMUX") != ""
}

func SimpleBarServer(cfg *config.Config) error {
	serverDir := filepath.Join(cfg.Dotfiles.Path, "ubersicht", "simple-bar-server")

	// Install pm2 globally if not found
	if _, err := exec.LookPath("pm2"); err != nil {
		fmt.Fprintln(os.Stdout, "  installing pm2 globally...")
		npmGlobal := exec.Command("npm", "install", "-g", "pm2")
		npmGlobal.Stdout = os.Stdout
		npmGlobal.Stderr = os.Stderr
		if err := npmGlobal.Run(); err != nil {
			return fmt.Errorf("npm install -g pm2 failed: %w", err)
		}
	}

	// Install npm deps if needed
	nodeModules := filepath.Join(serverDir, "node_modules")
	if _, err := os.Stat(nodeModules); err != nil {
		fmt.Fprintln(os.Stdout, "  installing dependencies...")
		install := exec.Command("npm", "install")
		install.Dir = serverDir
		install.Stdout = os.Stdout
		install.Stderr = os.Stderr
		if err := install.Run(); err != nil {
			return fmt.Errorf("npm install failed: %w", err)
		}
	}

	// Start or restart via ecosystem config
	fmt.Fprintln(os.Stdout, "  starting simple-bar-server...")
	start := exec.Command("pm2", "startOrRestart", filepath.Join(serverDir, "ecosystem.config.cjs"))
	start.Dir = serverDir
	start.Stdout = os.Stdout
	start.Stderr = os.Stderr
	if err := start.Run(); err != nil {
		return fmt.Errorf("pm2 start failed: %w", err)
	}

	// Check if pm2 startup is configured (user-level LaunchAgent)
	home, _ := os.UserHomeDir()
	launchAgent := filepath.Join(home, "Library", "LaunchAgents", fmt.Sprintf("pm2.%s.plist", os.Getenv("USER")))
	if _, err := os.Stat(launchAgent); err != nil {
		fmt.Fprintln(os.Stdout, "  configuring pm2 startup (requires sudo)...")
		startup := exec.Command("pm2", "startup")
		startupOut, _ := startup.Output()
		// pm2 startup outputs a sudo command to run
		lines := string(startupOut)
		if idx := len(lines); idx > 0 {
			sudo := exec.Command("bash", "-c", "pm2 startup launchd -u $USER --hp $HOME | tail -1 | bash")
			sudo.Stdout = os.Stdout
			sudo.Stderr = os.Stderr
			sudo.Stdin = os.Stdin
			_ = sudo.Run()
		}
		save := exec.Command("pm2", "save")
		save.Stdout = os.Stdout
		save.Stderr = os.Stderr
		_ = save.Run()
	}

	return nil
}

func HasSimpleBarServer(cfg *config.Config) bool {
	serverDir := filepath.Join(cfg.Dotfiles.Path, "ubersicht", "simple-bar-server")
	_, dirErr := os.Stat(serverDir)
	_, npmErr := exec.LookPath("npm")
	return dirErr == nil && npmErr == nil
}

func HasAerospace(_ *config.Config) bool {
	_, err := exec.LookPath("aerospace")
	return err == nil
}

func isHidden(name string) bool {
	return len(name) > 0 && name[0] == '.'
}
