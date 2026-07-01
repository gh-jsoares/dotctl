package secrets

import (
	"fmt"
	"os/exec"
	"strings"
)

type Provider interface {
	Get(ref string) (string, error)
	Name() string
}

type OPProvider struct{}

func (o *OPProvider) Name() string { return "1password" }

func (o *OPProvider) Get(ref string) (string, error) {
	if !strings.HasPrefix(ref, "op://") {
		return "", fmt.Errorf("invalid 1Password reference: %q (must start with op://)", ref)
	}

	out, err := exec.Command("op", "read", ref).Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("op read failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("op read failed: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

func DefaultProvider() Provider {
	return &OPProvider{}
}
