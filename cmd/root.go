package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "dotctl",
	Short: "Developer environment orchestrator",
	Long:  "dotctl orchestrates your developer environment — context switching, bootstrapping, syncing, and health checks.",
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

