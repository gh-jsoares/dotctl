package bootstrap

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/gh-jsoares/dotctl/internal/context"
)

type SSHKeyInfo struct {
	Label   string
	Host    string
	KeyFile string
}

func GenerateSSHKey(keyPath, comment string) (pubkey string, err error) {
	if err := os.MkdirAll(filepath.Dir(keyPath), 0o700); err != nil {
		return "", err
	}

	if _, err := os.Stat(keyPath); err == nil {
		pub, err := os.ReadFile(keyPath + ".pub")
		if err != nil {
			return "", err
		}
		return strings.TrimSpace(string(pub)), nil
	}

	cmd := exec.Command("ssh-keygen", "-t", "ed25519", "-C", comment, "-f", keyPath, "-N", "")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("ssh-keygen: %w", err)
	}

	pub, err := os.ReadFile(keyPath + ".pub")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(pub)), nil
}

func WriteSSHConfig(keys []SSHKeyInfo) error {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".ssh", "config")

	existing := ""
	if data, err := os.ReadFile(configPath); err == nil {
		existing = string(data)
	}

	var additions strings.Builder
	for _, k := range keys {
		hostBlock := fmt.Sprintf("Host %s\n", k.Host)
		if strings.Contains(existing, hostBlock) {
			continue
		}
		additions.WriteString(fmt.Sprintf("Host %s\n  HostName github.com\n  User git\n  IdentityFile %s\n  IdentitiesOnly yes\n\n", k.Host, k.KeyFile))
	}

	if additions.Len() == 0 {
		return nil
	}

	// Append to existing or create new
	if existing != "" && !strings.HasSuffix(existing, "\n") {
		existing += "\n"
	}

	content := existing + additions.String()
	return os.WriteFile(configPath, []byte(content), 0o600)
}

func VerifySSHConnection(host string) error {
	cmd := exec.Command("ssh", "-T", "-o", "StrictHostKeyChecking=accept-new", "git@"+host)
	output, _ := cmd.CombinedOutput()
	// ssh -T to GitHub returns exit code 1 with "Hi username!" on success
	if strings.Contains(string(output), "Hi ") {
		return nil
	}
	return fmt.Errorf("SSH verification failed for %s: %s", host, strings.TrimSpace(string(output)))
}

func PromptAndWaitForSSHKey(reader *bufio.Reader, label, host, pubkey string) error {
	fmt.Printf("\n  Public key for %q:\n\n", label)
	if err := copyToClipboard(pubkey); err == nil {
		fmt.Printf("  ✓ Public key copied to clipboard\n\n")
	} else {
		fmt.Printf("    %s\n\n", pubkey)
	}
	fmt.Printf("  Add this key to your GitHub account for host %q\n", host)
	fmt.Printf("  https://github.com/settings/ssh/new\n\n")
	fmt.Printf("  Press Enter when done...")
	reader.ReadString('\n')

	fmt.Printf("  Verifying SSH connection to %s...\n", host)
	if err := VerifySSHConnection(host); err != nil {
		return err
	}
	fmt.Printf("  ✓ Connection verified.\n")
	return nil
}

func SetupSSHFromContexts(reader *bufio.Reader, contexts map[string]*context.ContextDef) error {
	// Deduplicate by host — allow shared hosts only if they reference the same key
	type hostClaim struct {
		context string
		keyName string
	}
	hostOwners := make(map[string]hostClaim)
	for name, ctx := range contexts {
		if ctx.SSH.Host == "" {
			continue
		}
		keyName := ctx.Identity.SSHKey
		if keyName == "" {
			keyName = "id_ed25519_" + name
		}
		if existing, ok := hostOwners[ctx.SSH.Host]; ok {
			if existing.keyName != keyName {
				return fmt.Errorf("contexts %q and %q both claim SSH host %q but reference different keys (%s vs %s)",
					existing.context, name, ctx.SSH.Host, existing.keyName, keyName)
			}
			continue
		}
		hostOwners[ctx.SSH.Host] = hostClaim{context: name, keyName: keyName}
	}

	var keys []SSHKeyInfo

	for name, ctx := range contexts {
		if ctx.SSH.Host == "" {
			continue
		}

		keyName := ctx.Identity.SSHKey
		if keyName == "" {
			keyName = "id_ed25519_" + name
		}

		// Skip if another context already handled this host
		owner := hostOwners[ctx.SSH.Host]
		if owner.context != name {
			fmt.Printf("  ⊘ %s: SSH host %s already set up by context %q\n", name, ctx.SSH.Host, owner.context)
			continue
		}

		home, _ := os.UserHomeDir()
		keyPath := filepath.Join(home, ".ssh", keyName)

		// Try 1Password if key_source is set and op is available
		if ctx.SSH.KeySource != "" {
			if _, err := exec.LookPath("op"); err == nil {
				if _, err := os.Stat(keyPath); err != nil {
					fmt.Printf("  Retrieving SSH key for %q from 1Password...\n", name)
					if err := retrieveKeyFromOP(ctx.SSH.KeySource, keyPath); err != nil {
						fmt.Printf("  ⚠ Could not retrieve from 1Password: %v\n", err)
						fmt.Printf("  Falling back to key generation.\n")
					} else {
						fmt.Printf("  ✓ Key retrieved from 1Password.\n")
						keyInfo := SSHKeyInfo{Label: name, Host: ctx.SSH.Host, KeyFile: keyPath}
						keys = append(keys, keyInfo)
						WriteSSHConfig([]SSHKeyInfo{keyInfo})
						continue
					}
				} else {
					keyInfo := SSHKeyInfo{Label: name, Host: ctx.SSH.Host, KeyFile: keyPath}
					keys = append(keys, keyInfo)
					WriteSSHConfig([]SSHKeyInfo{keyInfo})
					continue
				}
			}
		}

		// Generate if not exists
		pubkey, err := GenerateSSHKey(keyPath, name)
		if err != nil {
			return fmt.Errorf("generating SSH key for %q: %w", name, err)
		}

		keyInfo := SSHKeyInfo{Label: name, Host: ctx.SSH.Host, KeyFile: keyPath}
		keys = append(keys, keyInfo)

		// Write SSH config immediately so the host alias resolves during verification
		if err := WriteSSHConfig([]SSHKeyInfo{keyInfo}); err != nil {
			return fmt.Errorf("writing SSH config: %w", err)
		}

		// Check if we need to prompt user to add the key
		if err := VerifySSHConnection(ctx.SSH.Host); err != nil {
			if err := PromptAndWaitForSSHKey(reader, name, ctx.SSH.Host, pubkey); err != nil {
				return err
			}
		} else {
			fmt.Printf("  ✓ SSH key for %q already authorized on %s.\n", name, ctx.SSH.Host)
		}
	}

	return nil
}

func retrieveKeyFromOP(ref, destPath string) error {
	out, err := exec.Command("op", "read", ref).Output()
	if err != nil {
		return err
	}

	if err := os.WriteFile(destPath, out, 0o600); err != nil {
		return err
	}

	// Generate pubkey from private key
	cmd := exec.Command("ssh-keygen", "-y", "-f", destPath)
	pub, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("deriving pubkey: %w", err)
	}
	return os.WriteFile(destPath+".pub", pub, 0o644)
}
