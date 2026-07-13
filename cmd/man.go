package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var manDir string

var manCmd = &cobra.Command{
	Use:    "man",
	Short:  "Generate man page",
	Long:   "Generate a single consolidated man page for dotctl.",
	Hidden: true,
	RunE:   runMan,
}

func init() {
	manCmd.Flags().StringVarP(&manDir, "dir", "d", "man", "output directory")
	rootCmd.AddCommand(manCmd)
}

func runMan(cmd *cobra.Command, args []string) error {
	if err := os.MkdirAll(manDir, 0o755); err != nil {
		return err
	}

	content := generateManPage(rootCmd)
	path := filepath.Join(manDir, "dotctl.1")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}

	fmt.Printf("Man page generated: %s\n", path)
	return nil
}

func generateManPage(root *cobra.Command) string {
	var b strings.Builder
	date := time.Now().Format("2006-01-02")

	b.WriteString(fmt.Sprintf(`.TH DOTCTL 1 "%s" "dotctl %s" "User Commands"
.SH NAME
dotctl \- developer environment orchestrator
.SH SYNOPSIS
.B dotctl
[\fIcommand\fR] [\fIflags\fR]
.SH DESCRIPTION
dotctl orchestrates your developer environment \(em context switching, bootstrapping, syncing, and health checks.
.SH COMMANDS
`, date, strings.TrimPrefix(version, "v")))

	writeCommands(&b, root.Commands(), "")

	b.WriteString(`.SH GLOBAL FLAGS
.TP
.B \-h, \-\-help
Show help for any command.
.SH ENVIRONMENT
.TP
.B DOTCTL_CONTEXT
Currently active context name.
.TP
.B DOTCTL_CONTEXT_ICON
Icon for the active context (from context TOML).
.SH FILES
.TP
.B ~/.config/dotctl/config.toml
Main configuration file.
.TP
.B <dotfiles>/contexts/*.toml
Context definitions with identity and environment.
.TP
.B ~/.config/git/config
Generated git config with identity includes.
.TP
.B ~/.local/state/dotctl/
State directory (active context, update cache).
.SH EXIT STATUS
.TP
.B 0
Success.
.TP
.B 1
Error or failed health check.
`)

	return b.String()
}

func writeCommands(b *strings.Builder, cmds []*cobra.Command, prefix string) {
	for _, cmd := range cmds {
		if cmd.Hidden || cmd.Name() == "help" || cmd.Name() == "completion" {
			continue
		}

		name := prefix + cmd.Name()
		b.WriteString(fmt.Sprintf(".TP\n.B %s\n%s\n", name, cmd.Short))

		if cmd.HasSubCommands() {
			writeCommands(b, cmd.Commands(), name+" ")
		}
	}
}
