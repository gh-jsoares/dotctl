package cmd

import (
	"fmt"
	"os"

	"github.com/gh-jsoares/dotctl/internal/context"
	"github.com/gh-jsoares/dotctl/internal/secrets"
	"github.com/spf13/cobra"
)

var secretsCmd = &cobra.Command{
	Use:   "secrets",
	Short: "Manage secrets via 1Password",
	Long:  "Unified interface for retrieving secrets from 1Password vaults.",
}

var secretsGetCmd = &cobra.Command{
	Use:   "get <reference>",
	Short: "Get a secret value",
	Long:  "Retrieve a secret from 1Password using an op:// reference.\nExample: dotctl secrets get op://Personal/GitHub/token",
	Args:  cobra.ExactArgs(1),
	RunE:  runSecretsGet,
}

var secretsEnvCmd = &cobra.Command{
	Use:   "env",
	Short: "Resolve lazy secrets for current context",
	Long:  "Resolve all [env.lazy] entries in the current context definition and print them as export statements.",
	RunE:  runSecretsEnv,
}

func init() {
	rootCmd.AddCommand(secretsCmd)
	secretsCmd.AddCommand(secretsGetCmd)
	secretsCmd.AddCommand(secretsEnvCmd)
}

func runSecretsGet(cmd *cobra.Command, args []string) error {
	ref := args[0]
	provider := secrets.DefaultProvider()

	val, err := provider.Get(ref)
	if err != nil {
		return err
	}

	fmt.Fprint(os.Stdout, val)
	return nil
}

func runSecretsEnv(cmd *cobra.Command, args []string) error {
	mgr, err := context.NewManager()
	if err != nil {
		return err
	}

	current, err := mgr.Current()
	if err != nil || current == "" {
		return fmt.Errorf("no active context — run 'dotctl ctx <name>' first")
	}

	ctx, err := mgr.Load(current)
	if err != nil {
		return err
	}

	if len(ctx.Lazy) == 0 {
		fmt.Fprintln(os.Stderr, "No lazy secrets defined in current context.")
		return nil
	}

	provider := secrets.DefaultProvider()
	for key, ref := range ctx.Lazy {
		val, err := provider.Get(ref)
		if err != nil {
			fmt.Fprintf(os.Stderr, "⚠ %s: %v\n", key, err)
			continue
		}
		fmt.Fprintf(os.Stdout, "export %s=%q\n", key, val)
	}
	return nil
}
