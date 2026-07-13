package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var manDir string

var manCmd = &cobra.Command{
	Use:    "man",
	Short:  "Generate man pages",
	Long:   "Generate man pages for all dotctl commands.",
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

	header := &doc.GenManHeader{
		Title:   "DOTCTL",
		Section: "1",
	}

	if err := doc.GenManTree(rootCmd, header, manDir); err != nil {
		return err
	}

	fmt.Printf("Man pages generated in %s/\n", manDir)
	return nil
}
