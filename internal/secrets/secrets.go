package secrets

import (
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

type Provider interface {
	Get(ref string) (string, error)
	Set(ref, value, category string) error
	List(vault string) ([]string, error)
	Name() string
}

type OPProvider struct {
	Account string
}

func (o *OPProvider) Name() string { return "1password" }

func (o *OPProvider) opArgs(args ...string) []string {
	if o.Account != "" {
		args = append(args, "--account", o.Account)
	}
	return args
}

func (o *OPProvider) Get(ref string) (string, error) {
	if !strings.HasPrefix(ref, "op://") {
		return "", fmt.Errorf("invalid 1Password reference: %q (must start with op://)", ref)
	}

	out, err := exec.Command("op", o.opArgs("read", ref)...).Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return "", fmt.Errorf("op read failed: %s", string(exitErr.Stderr))
		}
		return "", fmt.Errorf("op read failed: %w", err)
	}

	return strings.TrimSpace(string(out)), nil
}

func (o *OPProvider) Set(ref, value, category string) error {
	if !strings.HasPrefix(ref, "op://") {
		return fmt.Errorf("invalid 1Password reference: %q (must start with op://)", ref)
	}

	parts := ParseOPRef(ref)
	if parts == nil {
		return fmt.Errorf("cannot parse op:// reference: %q (expected op://vault/item/field)", ref)
	}

	// Try editing the existing item first
	editArgs := o.opArgs("item", "edit", parts.Item, "--vault", parts.Vault,
		fmt.Sprintf("%s=%s", parts.Field, value))
	if out, err := exec.Command("op", editArgs...).CombinedOutput(); err != nil {
		// Item doesn't exist — create it
		createArgs := o.opArgs("item", "create", "--category", category,
			"--title", parts.Item, "--vault", parts.Vault,
			fmt.Sprintf("%s=%s", parts.Field, value))
		if createOut, createErr := exec.Command("op", createArgs...).CombinedOutput(); createErr != nil {
			return fmt.Errorf("edit failed: %s; create failed: %s", string(out), string(createOut))
		}
	}

	return nil
}

func (o *OPProvider) List(vault string) ([]string, error) {
	out, err := exec.Command("op", o.opArgs("item", "list", "--vault", vault, "--format", "json")...).Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok {
			return nil, fmt.Errorf("op item list failed: %s", string(exitErr.Stderr))
		}
		return nil, fmt.Errorf("op item list failed: %w", err)
	}

	var items []struct {
		Title string `json:"title"`
	}
	if err := json.Unmarshal(out, &items); err != nil {
		return nil, fmt.Errorf("parsing op output: %w", err)
	}

	result := make([]string, len(items))
	for i, item := range items {
		result[i] = item.Title
	}
	return result, nil
}

type OPRef struct {
	Vault string
	Item  string
	Field string
}

func ParseOPRef(ref string) *OPRef {
	ref = strings.TrimPrefix(ref, "op://")
	parts := strings.SplitN(ref, "/", 3)
	if len(parts) != 3 {
		return nil
	}
	return &OPRef{Vault: parts[0], Item: parts[1], Field: parts[2]}
}

func DefaultProvider() *OPProvider {
	return &OPProvider{}
}

func ProviderWithAccount(account string) *OPProvider {
	return &OPProvider{Account: account}
}
