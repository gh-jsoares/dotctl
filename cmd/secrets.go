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
	Long:  "Unified interface for managing secrets in 1Password vaults.",
}

var secretsGetCmd = &cobra.Command{
	Use:   "get <reference>",
	Short: "Get a secret value",
	Long:  "Retrieve a secret from 1Password using an op:// reference.\nExample: dotctl secrets get op://Personal/GitHub/token",
	Args:  cobra.ExactArgs(1),
	RunE:  runSecretsGet,
}

var secretsSetCmd = &cobra.Command{
	Use:   "set <reference> <value>",
	Short: "Set a secret value",
	Long:  "Store a secret in 1Password at the given op:// reference.\nExample: dotctl secrets set op://Personal/GitHub/token ghp_abc123",
	Args:  cobra.ExactArgs(2),
	RunE:  runSecretsSet,
}

var secretsListCmd = &cobra.Command{
	Use:   "list [vault]",
	Short: "List items in a vault",
	Long:  "List all items in a 1Password vault.\nDefaults to the \"dotctl\" vault.\nExample: dotctl secrets list",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runSecretsList,
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
	secretsCmd.AddCommand(secretsSetCmd)
	secretsCmd.AddCommand(secretsListCmd)
	secretsCmd.AddCommand(secretsEnvCmd)

	secretsCmd.PersistentFlags().String("account", "", "1Password account to use (e.g., my.1password.com)")
	secretsSetCmd.Flags().String("category", "Secure Note", "1Password item category (e.g., \"SSH Key\", \"API Credentials\", \"Password\")")
}

func secretsAccount(cmd *cobra.Command) string {
	account, _ := cmd.Flags().GetString("account")
	if account == "" {
		return ""
	}

	// Try resolving as a context name
	mgr, err := context.NewManager()
	if err != nil {
		return account
	}
	ctx, err := mgr.Load(account)
	if err != nil {
		return account
	}
	if ctx.OPAccount != "" {
		return ctx.OPAccount
	}
	return account
}

func runSecretsGet(cmd *cobra.Command, args []string) error {
	ref := args[0]
	provider := secrets.ProviderWithAccount(secretsAccount(cmd))

	val, err := provider.Get(ref)
	if err != nil {
		return err
	}

	fmt.Fprint(os.Stdout, val)
	return nil
}

func runSecretsSet(cmd *cobra.Command, args []string) error {
	ref := args[0]
	value := args[1]
	category, _ := cmd.Flags().GetString("category")
	provider := secrets.ProviderWithAccount(secretsAccount(cmd))

	if err := provider.Set(ref, value, category); err != nil {
		return err
	}

	fmt.Fprintln(os.Stdout, "✓ Secret saved.")
	return nil
}

func runSecretsList(cmd *cobra.Command, args []string) error {
	vault := "dotctl"
	if len(args) > 0 {
		vault = args[0]
	}
	provider := secrets.ProviderWithAccount(secretsAccount(cmd))

	items, err := provider.List(vault)
	if err != nil {
		return err
	}

	for _, item := range items {
		fmt.Fprintln(os.Stdout, item)
	}
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

	account := secretsAccount(cmd)
	if account == "" {
		account = ctx.OPAccount
	}
	provider := secrets.ProviderWithAccount(account)
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
