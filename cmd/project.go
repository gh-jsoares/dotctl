package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/gh-jsoares/dotctl/internal/context"
	"github.com/spf13/cobra"
)

var projectCmd = &cobra.Command{
	Use:   "project",
	Short: "Detect project context",
	Long:  "Check the current directory for a .dotctx file and show project context information.",
	RunE:  runProject,
}

func init() {
	rootCmd.AddCommand(projectCmd)
}

func runProject(cmd *cobra.Command, args []string) error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}

	dotctxPath := findDotctx(dir)
	if dotctxPath == "" {
		fmt.Println("No .dotctx found in this directory tree.")
		return nil
	}

	preferred, err := parseDotctx(dotctxPath)
	if err != nil {
		return fmt.Errorf("parsing %s: %w", dotctxPath, err)
	}

	mgr, err := context.NewManager()
	if err != nil {
		return err
	}

	current, _ := mgr.Current()

	fmt.Printf("Project dotctx: %s\n", dotctxPath)
	fmt.Printf("Preferred context: %s\n", preferred)
	fmt.Printf("Current context: %s\n", valueOrDefault(current, "(none)"))

	if preferred != "" && current != preferred {
		fmt.Printf("\n⚠ Context mismatch. Run: ctx %s\n", preferred)
	} else if preferred != "" && current == preferred {
		fmt.Println("\n✓ Context matches.")
	}

	return nil
}

func findDotctx(dir string) string {
	for i := 0; i < 10; i++ {
		path := filepath.Join(dir, ".dotctx")
		if _, err := os.Stat(path); err == nil {
			return path
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

func parseDotctx(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "context") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				val := strings.TrimSpace(parts[1])
				val = strings.Trim(val, `"'`)
				return val, nil
			}
		}
	}
	return "", nil
}

func valueOrDefault(val, def string) string {
	if val == "" {
		return def
	}
	return val
}
