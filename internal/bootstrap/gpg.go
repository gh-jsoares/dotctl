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

func SetupGPGFromContexts(reader *bufio.Reader, contexts map[string]*context.ContextDef) error {
	for name, ctx := range contexts {
		if ctx.Identity.GitConfig == "" {
			continue
		}

		email := gitConfigEmail(ctx.Identity.GitConfig)
		if email == "" {
			fmt.Printf("  ⊘ %s: no email in git config, skipping GPG\n", name)
			continue
		}

		// Check if key already exists locally
		if keyID := findGPGKey(email); keyID != "" {
			fmt.Printf("  ✓ GPG key for %s already exists: %s\n", name, keyID)
			if err := writeGitSigningConfig(ctx.Identity.GitConfig, keyID); err != nil {
				return err
			}
			continue
		}

		// Try 1Password if gpg_key_source is set
		if ctx.Identity.GPGKeySource != "" {
			if _, err := exec.LookPath("op"); err == nil {
				fmt.Printf("  Retrieving GPG key for %q from 1Password...\n", name)
				if err := importGPGFromOP(ctx.Identity.GPGKeySource); err != nil {
					fmt.Printf("  ⚠ Could not retrieve from 1Password: %v\n", err)
					fmt.Printf("  Falling back to key generation.\n")
				} else {
					fmt.Printf("  ✓ GPG key imported from 1Password.\n")
					keyID := findGPGKey(email)
					if keyID != "" {
						if err := writeGitSigningConfig(ctx.Identity.GitConfig, keyID); err != nil {
							return err
						}
					}
					continue
				}
			}
		}

		// Generate new key
		fmt.Printf("  Generating GPG key for %s (%s)...\n", name, email)
		keyID, err := generateGPGKey(name, email)
		if err != nil {
			return fmt.Errorf("generating GPG key for %q: %w", name, err)
		}
		fmt.Printf("  ✓ Generated key: %s\n", keyID)

		if err := writeGitSigningConfig(ctx.Identity.GitConfig, keyID); err != nil {
			return err
		}

		// Export and prompt user to add to GitHub
		pubkey, err := exportGPGPublicKey(keyID)
		if err != nil {
			return err
		}

		fmt.Printf("\n  GPG public key for %q:\n\n", name)
		fmt.Printf("    %s\n\n", keyID)
		fmt.Printf("  Add this key to your GitHub account:\n")
		fmt.Printf("  https://github.com/settings/gpg/new\n\n")
		fmt.Printf("  Public key block (paste this):\n\n")
		for _, line := range strings.Split(pubkey, "\n") {
			fmt.Printf("    %s\n", line)
		}
		fmt.Printf("\n  Press Enter when done...")
		reader.ReadString('\n')
	}

	return nil
}

func findGPGKey(email string) string {
	cmd := exec.Command("gpg", "--list-keys", "--with-colons", email)
	out, err := cmd.Output()
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(out), "\n") {
		if strings.HasPrefix(line, "fpr:") {
			parts := strings.Split(line, ":")
			if len(parts) >= 10 {
				fpr := parts[9]
				if len(fpr) >= 16 {
					return fpr[len(fpr)-16:]
				}
			}
		}
	}
	return ""
}

func generateGPGKey(name, email string) (string, error) {
	batchInput := fmt.Sprintf(`%%no-protection
Key-Type: eddsa
Key-Curve: ed25519
Subkey-Type: ecdh
Subkey-Curve: cv25519
Name-Real: %s
Name-Email: %s
Expire-Date: 0
%%commit
`, nameFromContext(name), email)

	cmd := exec.Command("gpg", "--batch", "--gen-key")
	cmd.Stdin = strings.NewReader(batchInput)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return "", err
	}

	keyID := findGPGKey(email)
	if keyID == "" {
		return "", fmt.Errorf("key generated but could not find it for %s", email)
	}
	return keyID, nil
}

func exportGPGPublicKey(keyID string) (string, error) {
	out, err := exec.Command("gpg", "--armor", "--export", keyID).Output()
	if err != nil {
		return "", fmt.Errorf("exporting GPG key: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func importGPGFromOP(ref string) error {
	out, err := exec.Command("op", "read", ref).Output()
	if err != nil {
		return err
	}

	cmd := exec.Command("gpg", "--batch", "--import")
	cmd.Stdin = strings.NewReader(string(out))
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func writeGitSigningConfig(gitConfigName, keyID string) error {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "git", gitConfigName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("reading %s: %w", configPath, err)
	}

	content := string(data)

	// Check if signing is already configured
	if strings.Contains(content, "signingkey") {
		return nil
	}

	// Append signing config to [user] section
	if strings.Contains(content, "[user]") {
		content = strings.Replace(content, "[user]",
			fmt.Sprintf("[user]\n    signingkey = %s", keyID), 1)
	}

	// Add commit and tag signing sections
	if !strings.Contains(content, "[commit]") {
		content += "\n[commit]\n    gpgsign = true\n"
	}
	if !strings.Contains(content, "[tag]") {
		content += "\n[tag]\n    gpgsign = true\n"
	}

	return os.WriteFile(configPath, []byte(content), 0o644)
}

func gitConfigEmail(gitConfigName string) string {
	home, _ := os.UserHomeDir()
	configPath := filepath.Join(home, ".config", "git", gitConfigName)

	data, err := os.ReadFile(configPath)
	if err != nil {
		return ""
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "email") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				return strings.TrimSpace(parts[1])
			}
		}
	}
	return ""
}

func nameFromContext(ctx string) string {
	return "João Soares"
}
