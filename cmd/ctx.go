package cmd

import (
	"fmt"
	"os"

	"github.com/gh-jsoares/dotctl/internal/context"
	"github.com/spf13/cobra"
)

var ctxCmd = &cobra.Command{
	Use:   "ctx [context-name]",
	Short: "Switch developer context",
	Long:  "Switch between work and personal contexts. Updates symlinks, generates env file, and updates tmux environment.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runCtx,
}

var ctxDefaultCmd = &cobra.Command{
	Use:   "default [context-name]",
	Short: "Set the default context for new shells",
	Args:  cobra.ExactArgs(1),
	RunE:  runCtxDefault,
}

var ctxCurrentCmd = &cobra.Command{
	Use:   "current",
	Short: "Show the current active context",
	RunE:  runCtxCurrent,
}

var ctxListCmd = &cobra.Command{
	Use:   "list",
	Short: "List available contexts",
	RunE:  runCtxList,
}

func init() {
	rootCmd.AddCommand(ctxCmd)
	ctxCmd.AddCommand(ctxDefaultCmd)
	ctxCmd.AddCommand(ctxCurrentCmd)
	ctxCmd.AddCommand(ctxListCmd)
}

func runCtx(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return runCtxCurrent(cmd, args)
	}

	name := args[0]
	mgr, err := context.NewManager()
	if err != nil {
		return err
	}

	if err := mgr.Switch(name); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Switched to context: %s\n", name)
	fmt.Fprintf(os.Stdout, "Source the env file to apply: source %s\n", mgr.EnvFilePath())
	return nil
}

func runCtxDefault(cmd *cobra.Command, args []string) error {
	name := args[0]
	mgr, err := context.NewManager()
	if err != nil {
		return err
	}

	if err := mgr.SetDefault(name); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "Default context set to: %s\n", name)
	return nil
}

func runCtxList(cmd *cobra.Command, args []string) error {
	mgr, err := context.NewManager()
	if err != nil {
		return err
	}

	current, _ := mgr.Current()
	names, err := mgr.List()
	if err != nil {
		return err
	}

	if len(names) == 0 {
		fmt.Fprintln(os.Stdout, "No contexts found.")
		return nil
	}

	for _, name := range names {
		if name == current {
			fmt.Fprintf(os.Stdout, "* %s (active)\n", name)
		} else {
			fmt.Fprintf(os.Stdout, "  %s\n", name)
		}
	}
	return nil
}

func runCtxCurrent(cmd *cobra.Command, args []string) error {
	mgr, err := context.NewManager()
	if err != nil {
		return err
	}

	current, err := mgr.Current()
	if err != nil {
		return err
	}

	if current == "" {
		fmt.Fprintln(os.Stdout, "No context set")
	} else {
		fmt.Fprintln(os.Stdout, current)
	}
	return nil
}
