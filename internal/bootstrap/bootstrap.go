package bootstrap

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/gh-jsoares/dotctl/internal/context"
)

type Options struct {
	DotfilesRemote string
	DotfilesPath   string
	DefaultContext string
	SSHHost        string // populated by pre-clone SSH step
}

// PreCloneInfo is gathered interactively before repos are cloned
type PreCloneInfo struct {
	KeyLabel string
	Host     string
}

type Step struct {
	Name string
	Fn   func(opts *Options, reader *bufio.Reader) error
	Skip func(opts *Options) bool
}

func Steps() []Step {
	return []Step{
		{"Xcode CLI tools", stepXcode, xcodeInstalled},
		{"Nix", stepNix, nixInstalled},
		{"Pre-clone SSH setup", stepPreCloneSSH, nil},
		{"Clone dotfiles repo", stepCloneDotfiles, dotfilesCloned},
		{"nix-darwin switch", stepNixDarwinSwitch, noFlake},
		{"Post-clone SSH setup (from contexts)", stepPostCloneSSH, nil},
		{"Create context directories", stepCreateContextDirs, nil},
		{"Stow dotfiles", stepStowDotfiles, noStowDir},
		{"mise install", stepMiseInstall, miseNotAvailable},
	}
}

// --- Step implementations ---

func stepXcode(opts *Options, _ *bufio.Reader) error {
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

	// xcode-select --install returns immediately but downloads in background.
	// Wait for it to finish before proceeding (avoids bandwidth contention).
	fmt.Println("  Waiting for Xcode CLI tools installation to complete...")
	for {
		if exec.Command("xcode-select", "-p").Run() == nil {
			break
		}
		time.Sleep(5 * time.Second)
	}
	return nil
}

func xcodeInstalled(_ *Options) bool {
	return exec.Command("xcode-select", "-p").Run() == nil
}

func stepNix(opts *Options, _ *bufio.Reader) error {
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

func stepPreCloneSSH(opts *Options, reader *bufio.Reader) error {
	// Check if repos are already cloned — if so, skip pre-clone SSH
	if dotfilesCloned(opts) {
		fmt.Println("  Repos already cloned, skipping pre-clone SSH.")
		return nil
	}

	// We need at least one SSH key to clone. Ask user for the minimal info.
	fmt.Println("  SSH key needed to clone repos from GitHub.")
	fmt.Println("")

	label, err := promptLine(reader, "  SSH key label (e.g., personal)")
	if err != nil {
		return err
	}

	host, err := promptLine(reader, "  GitHub host alias (e.g., personal.github.com)")
	if err != nil {
		return err
	}

	// Generate key
	home, _ := os.UserHomeDir()
	keyPath := filepath.Join(home, ".ssh", "id_ed25519_"+label)
	pubkey, err := GenerateSSHKey(keyPath, label)
	if err != nil {
		return err
	}
	if err := WriteSSHConfig([]SSHKeyInfo{{Label: label, Host: host, KeyFile: keyPath}}); err != nil {
		return err
	}

	// Verify or prompt user to add key
	if err := VerifySSHConnection(host); err != nil {
		if err := PromptAndWaitForSSHKey(reader, label, host, pubkey); err != nil {
			return err
		}
	} else {
		fmt.Printf("  ✓ SSH key already authorized on %s.\n", host)
	}

	opts.SSHHost = host
	return nil
}

func stepCloneDotfiles(opts *Options, reader *bufio.Reader) error {
	if opts.DotfilesRemote == "" {
		repo, err := promptLine(reader, "  GitHub repo (e.g., user/dotfiles)")
		if err != nil {
			return err
		}
		host := opts.SSHHost
		if host == "" {
			host = "github.com"
		}
		opts.DotfilesRemote = fmt.Sprintf("git@%s:%s.git", host, repo)
	}

	fmt.Printf("  Cloning %s\n", opts.DotfilesRemote)
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

func stepNixDarwinSwitch(opts *Options, _ *bufio.Reader) error {
	flakePath := opts.DotfilesPath
	hostname, _ := os.Hostname()
	ref := fmt.Sprintf("%s#%s", flakePath, hostname)

	// Check if darwin-rebuild exists (not first run)
	if _, err := exec.LookPath("darwin-rebuild"); err == nil {
		cmd := exec.Command("darwin-rebuild", "switch", "--flake", ref)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return cmd.Run()
	}

	// First run: use nix run to bootstrap nix-darwin
	fmt.Println("  First nix-darwin run (bootstrapping)...")
	cmd := exec.Command("nix", "run", "nix-darwin", "--", "switch", "--flake", ref)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = append(os.Environ(), "NIX_CONFIG=experimental-features = nix-command flakes")
	return cmd.Run()
}

func noFlake(opts *Options) bool {
	_, err := os.Stat(filepath.Join(opts.DotfilesPath, "flake.nix"))
	return err != nil
}

func stepPostCloneSSH(opts *Options, reader *bufio.Reader) error {
	// Read all context definitions and set up any remaining SSH keys
	contextsDir := filepath.Join(opts.DotfilesPath, "contexts")
	if _, err := os.Stat(contextsDir); err != nil {
		fmt.Println("  No contexts/ directory found, skipping.")
		return nil
	}

	entries, err := os.ReadDir(contextsDir)
	if err != nil {
		return err
	}

	contexts := make(map[string]*context.ContextDef)
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		name := strings.TrimSuffix(e.Name(), ".toml")
		var ctx context.ContextDef
		path := filepath.Join(contextsDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		if err := tomlUnmarshal(data, &ctx); err != nil {
			fmt.Printf("  ⚠ Could not parse %s: %v\n", e.Name(), err)
			continue
		}
		contexts[name] = &ctx
	}

	if len(contexts) == 0 {
		return nil
	}

	return SetupSSHFromContexts(reader, contexts)
}

func stepCreateContextDirs(opts *Options, _ *bufio.Reader) error {
	// Read context definitions to know what directories to create
	contextsDir := filepath.Join(opts.DotfilesPath, "contexts")
	if _, err := os.Stat(contextsDir); err != nil {
		// No contexts dir — create defaults
		return createDefaultContextDirs()
	}

	entries, err := os.ReadDir(contextsDir)
	if err != nil {
		return createDefaultContextDirs()
	}

	home, _ := os.UserHomeDir()
	created := map[string]bool{}

	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".toml") {
			continue
		}
		path := filepath.Join(contextsDir, e.Name())
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var ctx context.ContextDef
		if err := tomlUnmarshal(data, &ctx); err != nil {
			continue
		}

		// Create symlink targets
		for _, target := range ctx.Symlinks {
			target = expandHome(target, home)
			if !created[target] {
				os.MkdirAll(target, 0o755)
				created[target] = true
			}
		}

		// Create DOCKER_CONFIG directory if set
		if dockerCfg, ok := ctx.Env["DOCKER_CONFIG"]; ok {
			dockerCfg = expandHome(dockerCfg, home)
			if !created[dockerCfg] {
				os.MkdirAll(dockerCfg, 0o755)
				created[dockerCfg] = true
			}
		}
	}

	// Also create state dir
	os.MkdirAll(filepath.Join(home, ".local", "state", "dotctl"), 0o755)
	return nil
}

func createDefaultContextDirs() error {
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

func stepStowDotfiles(opts *Options, _ *bufio.Reader) error {
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

func stepMiseInstall(opts *Options, _ *bufio.Reader) error {
	cmd := exec.Command("mise", "install")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func miseNotAvailable(_ *Options) bool {
	_, err := exec.LookPath("mise")
	return err != nil
}

// --- Helpers ---

func promptLine(reader *bufio.Reader, label string) (string, error) {
	fmt.Printf("%s: ", label)
	line, err := reader.ReadString('\n')
	if err != nil {
		return "", err
	}
	val := strings.TrimSpace(line)
	if val == "" {
		return "", fmt.Errorf("%s is required", label)
	}
	return val, nil
}

func expandHome(path, home string) string {
	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:])
	}
	return path
}
