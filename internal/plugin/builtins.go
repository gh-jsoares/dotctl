package plugin

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/gh-jsoares/dotctl/internal/config"
)

//go:embed builtins/*
var builtinFS embed.FS

func extractBuiltins() (string, error) {
	cacheDir := filepath.Join(config.StateDir(), "builtins")

	if err := os.RemoveAll(cacheDir); err != nil {
		return "", fmt.Errorf("cleaning builtins cache: %w", err)
	}

	err := fs.WalkDir(builtinFS, "builtins", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Strip "builtins/" prefix for the output path
		rel, _ := filepath.Rel("builtins", path)
		out := filepath.Join(cacheDir, rel)

		if d.IsDir() {
			return os.MkdirAll(out, 0o755)
		}

		data, err := builtinFS.ReadFile(path)
		if err != nil {
			return err
		}

		perm := os.FileMode(0o644)
		if filepath.Ext(path) == ".sh" {
			perm = 0o755
		}

		return os.WriteFile(out, data, perm)
	})
	if err != nil {
		return "", fmt.Errorf("extracting builtins: %w", err)
	}

	return cacheDir, nil
}
