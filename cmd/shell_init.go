package cmd

import (
	"fmt"
	"os"

	"github.com/gh-jsoares/dotctl/internal/shell"
	"github.com/spf13/cobra"
)

var shellInitCmd = &cobra.Command{
	Use:   "shell-init [shell]",
	Short: "Output shell integration code",
	Long:  "Outputs shell functions and hooks for context switching, project detection, and tool wrappers. Source this in your shell rc file.",
	Args:  cobra.ExactArgs(1),
	ValidArgs: []string{"zsh", "bash"},
	RunE:  runShellInit,
}

var shellInitInstallCmd = &cobra.Command{
	Use:   "install",
	Short: "Write shell integration to disk",
	Long:  "Writes shell integration files to ~/.local/share/dotctl/ for sourcing without subprocess cost.",
	RunE:  runShellInitInstall,
}

func init() {
	rootCmd.AddCommand(shellInitCmd)
	shellInitCmd.AddCommand(shellInitInstallCmd)
}

func runShellInit(cmd *cobra.Command, args []string) error {
	shellName := args[0]
	code, err := shell.Generate(shellName)
	if err != nil {
		return err
	}
	fmt.Fprint(os.Stdout, code)
	return nil
}

func runShellInitInstall(cmd *cobra.Command, args []string) error {
	if err := shell.Install(); err != nil {
		return err
	}
	fmt.Fprintln(os.Stdout, "Shell integration installed to ~/.local/share/dotctl/")
	fmt.Fprintln(os.Stdout, "Add to .zshrc: source ~/.local/share/dotctl/init.zsh")
	return nil
}
