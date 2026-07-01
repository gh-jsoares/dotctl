package bootstrap

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

type Options struct {
	DotfilesRemote string
	DotctlRemote   string
	DotfilesPath   string
	DotctlPath     string
	SSHHosts       map[string]string
	DefaultContext string
}

type Step struct {
	Name string
	Fn   func(opts *Options) error
	Skip func(opts *Options) bool
}

func Steps() []Step {
	return []Step{
		{"Xcode CLI tools", installXcode, xcodeInstalled},
		{"Nix", installNix, nixInstalled},
		{"1Password CLI", installOPCLI, opInstalled},
		{"SSH keys from 1Password", setupSSHKeys, nil},
		{"Clone dotfiles repo", cloneDotfiles, dotfilesCloned},
		{"Clone dotctl repo", cloneDotctl, dotctlCloned},
		{"Create context directories", createContextDirs, nil},
		{"nix-darwin switch", nixDarwinSwitch, noFlake},
		{"Stow dotfiles", stowDotfiles, noStowDir},
		{"mise install", miseInstall, miseNotAvailable},
	}
}

func installXcode(_ *Options) error {
	cmd := exec.Command("xcode-select", "--install")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil
		}
		return err
	}
	return nil
}

func xcodeInstalled(_ *Options) bool {
	return exec.Command("xcode-select", "-p").Run() == nil
}

func installNix(_ *Options) error {
	cmd := exec.Command("bash", "-c", "curl --proto '=https' --tlsv1.2 -sSf -L https://install.determinate.systems/nix | sh -s -- install --no-confirm")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}

func nixInstalled(_ *Options) bool {
	_, err := exec.LookPath("nix")
	return err == nil
}

func installOPCLI(_ *Options) error {
	if _, err := exec.LookPath("brew"); err != nil {
		return fmt.Errorf("homebrew not available; install 1Password CLI manually")
	}
	cmd := exec.Command("brew", "install", "--cask", "1password-cli")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func opInstalled(_ *Options) bool {
	_, err := exec.LookPath("op")
	return err == nil
}

func setupSSHKeys(opts *Options) error {
	home, _ := os.UserHomeDir()
	sshDir := filepath.Join(home, ".ssh")
	if err := os.MkdirAll(sshDir, 0o700); err != nil {
		return err
	}

	fmt.Println("  Please ensure SSH keys are available (via 1Password SSH Agent or manual export).")

	configPath := filepath.Join(sshDir, "config")
	if _, err := os.Stat(configPath); err == nil {
		fmt.Println("  ~/.ssh/config already exists — skipping write.")
		return nil
	}

	if len(opts.SSHHosts) == 0 {
		fmt.Println("  No ssh.hosts configured — skipping SSH config generation.")
		return nil
	}

	var config string
	for name, host := range opts.SSHHosts {
		config += fmt.Sprintf("Host %s\n  HostName github.com\n  User git\n  IdentityFile ~/.ssh/id_ed25519_%s\n  IdentitiesOnly yes\n\n", host, name)
	}
	config += "Host *\n  AddKeysToAgent yes\n  UseKeychain yes\n"

	if err := os.WriteFile(configPath, []byte(config), 0o600); err != nil {
		return err
	}
	fmt.Println("  Wrote ~/.ssh/config with host aliases.")
	return nil
}

func cloneDotfiles(opts *Options) error {
	if err := os.MkdirAll(filepath.Dir(opts.DotfilesPath), 0o755); err != nil {
		return err
	}
	cmd := exec.Command("git", "clone", opts.DotfilesRemote, opts.DotfilesPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func dotfilesCloned(opts *Options) bool {
	_, err := os.Stat(filepath.Join(opts.DotfilesPath, ".git"))
	return err == nil
}

func cloneDotctl(opts *Options) error {
	if err := os.MkdirAll(filepath.Dir(opts.DotctlPath), 0o755); err != nil {
		return err
	}
	cmd := exec.Command("git", "clone", opts.DotctlRemote, opts.DotctlPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func dotctlCloned(opts *Options) bool {
	_, err := os.Stat(filepath.Join(opts.DotctlPath, ".git"))
	return err == nil
}

func createContextDirs(_ *Options) error {
	home, _ := os.UserHomeDir()
	dirs := []string{
		filepath.Join(home, ".aws-work"),
		filepath.Join(home, ".aws-personal"),
		filepath.Join(home, ".kube-work"),
		filepath.Join(home, ".kube-personal"),
		filepath.Join(home, ".docker-work"),
		filepath.Join(home, ".docker-personal"),
		filepath.Join(home, ".local", "state", "dotctl"),
	}
	for _, d := range dirs {
		if err := os.MkdirAll(d, 0o755); err != nil {
			return fmt.Errorf("creating %s: %w", d, err)
		}
	}
	return nil
}

func nixDarwinSwitch(opts *Options) error {
	flakePath := opts.DotfilesPath
	hostname, _ := os.Hostname()
	ref := fmt.Sprintf("%s#%s", flakePath, hostname)
	cmd := exec.Command("darwin-rebuild", "switch", "--flake", ref)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func noFlake(opts *Options) bool {
	_, err := os.Stat(filepath.Join(opts.DotfilesPath, "flake.nix"))
	return err != nil
}

func stowDotfiles(opts *Options) error {
	stowDir := filepath.Join(opts.DotfilesPath, "stow")
	entries, err := os.ReadDir(stowDir)
	if err != nil {
		return err
	}

	packages := []string{}
	for _, e := range entries {
		if e.IsDir() && e.Name()[0] != '.' {
			packages = append(packages, e.Name())
		}
	}
	if len(packages) == 0 {
		return nil
	}

	home, _ := os.UserHomeDir()
	args := append([]string{"-S", "-d", stowDir, "-t", home}, packages...)
	cmd := exec.Command("stow", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func noStowDir(opts *Options) bool {
	_, err := os.Stat(filepath.Join(opts.DotfilesPath, "stow"))
	return err != nil
}

func miseInstall(_ *Options) error {
	cmd := exec.Command("mise", "install")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func miseNotAvailable(_ *Options) bool {
	_, err := exec.LookPath("mise")
	return err != nil
}
