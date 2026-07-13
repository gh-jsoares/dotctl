package cmd

import (
	"fmt"
	"os"

	"github.com/gh-jsoares/dotctl/internal/config"
	"github.com/gh-jsoares/dotctl/internal/context"
	"github.com/gh-jsoares/dotctl/internal/plugin"
	"github.com/spf13/cobra"
)

var pluginsCmd = &cobra.Command{
	Use:   "plugins",
	Short: "Manage dotctl plugins",
}

var pluginsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List discovered plugins and their status",
	RunE:  runPluginsList,
}

var pluginsValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate all plugin manifests",
	RunE:  runPluginsValidate,
}

var pluginsRunCmd = &cobra.Command{
	Use:   "run <name>",
	Short: "Run a plugin's sync hook manually",
	Args:  cobra.ExactArgs(1),
	RunE:  runPluginsRun,
}

func init() {
	pluginsCmd.AddCommand(pluginsListCmd)
	pluginsCmd.AddCommand(pluginsValidateCmd)
	pluginsCmd.AddCommand(pluginsRunCmd)
	rootCmd.AddCommand(pluginsCmd)
}

func runPluginsList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	plugins, err := plugin.Discover(cfg)
	if err != nil {
		return err
	}

	if len(plugins) == 0 {
		fmt.Fprintln(os.Stdout, "No plugins found.")
		return nil
	}

	var currentContext string
	if mgr, err := context.NewManager(); err == nil {
		currentContext, _ = mgr.Current()
	}

	for _, p := range plugins {
		enabled := plugin.EvaluateConditions([]*plugin.Plugin{p}, cfg, currentContext)
		status := "✓"
		if len(enabled) == 0 {
			status = "⊘"
		}

		hooks := ""
		if p.Hooks.Sync != "" {
			hooks += " sync"
		}
		if p.Hooks.Bootstrap != "" {
			hooks += " bootstrap"
		}
		if p.Hooks.Doctor != "" {
			hooks += " doctor"
		}

		fmt.Fprintf(os.Stdout, "%s %-20s %s [%s]\n", status, p.Name, p.Description, hooks)
	}

	return nil
}

func runPluginsValidate(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	plugins, err := plugin.Discover(cfg)
	if err != nil {
		return err
	}

	if len(plugins) == 0 {
		fmt.Fprintln(os.Stdout, "No plugins found.")
		return nil
	}

	if err := plugin.Validate(plugins); err != nil {
		return err
	}

	fmt.Fprintf(os.Stdout, "✓ All %d plugins valid.\n", len(plugins))
	return nil
}

func runPluginsRun(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	plugins, err := plugin.Discover(cfg)
	if err != nil {
		return err
	}

	name := args[0]
	var target *plugin.Plugin
	for _, p := range plugins {
		if p.Name == name {
			target = p
			break
		}
	}

	if target == nil {
		return fmt.Errorf("plugin %q not found", name)
	}

	if target.Hooks.Sync == "" {
		return fmt.Errorf("plugin %q has no sync hook", name)
	}

	var currentContext string
	if mgr, err := context.NewManager(); err == nil {
		currentContext, _ = mgr.Current()
	}

	fmt.Fprintf(os.Stdout, "▸ %s\n", target.Name)
	if err := plugin.Execute(target, "sync", cfg, currentContext); err != nil {
		return fmt.Errorf("plugin %s failed: %w", target.Name, err)
	}
	fmt.Fprintf(os.Stdout, "✓ %s\n", target.Name)
	return nil
}
