package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"

	"github.com/gh-jsoares/dotctl/internal/config"
	"github.com/spf13/cobra"
)

var updateFromSource bool

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update dotctl to the latest version",
	Long:  "Download the latest dotctl release from GitHub and replace the current binary, or rebuild from source.",
	RunE:  runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolVar(&updateFromSource, "from-source", false, "rebuild from source (git pull + go build) instead of downloading release")
}

func runUpdate(cmd *cobra.Command, args []string) error {
	if updateFromSource {
		return updateSource()
	}
	return updateRelease()
}

func updateRelease() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	owner := cfg.Dotctl.RepoOwner
	repo := cfg.Dotctl.RepoName
	if owner == "" || repo == "" {
		return fmt.Errorf("dotctl.repo_owner and dotctl.repo_name must be set in %s", config.DefaultConfigPath())
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("checking latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return fmt.Errorf("GitHub API returned %d (repo may be private or not yet published)", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return fmt.Errorf("parsing release: %w", err)
	}

	assetName := fmt.Sprintf("dotctl_%s_%s", runtime.GOOS, runtime.GOARCH)
	var downloadURL string
	for _, a := range release.Assets {
		if a.Name == assetName {
			downloadURL = a.BrowserDownloadURL
			break
		}
	}

	if downloadURL == "" {
		return fmt.Errorf("no release asset found for %s/%s in %s", runtime.GOOS, runtime.GOARCH, release.TagName)
	}

	currentBin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current binary: %w", err)
	}

	fmt.Printf("Updating to %s...\n", release.TagName)

	dlResp, err := http.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("downloading: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", dlResp.StatusCode)
	}

	tmpFile := currentBin + ".new"
	f, err := os.OpenFile(tmpFile, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o755)
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}

	if _, err := io.Copy(f, dlResp.Body); err != nil {
		f.Close()
		os.Remove(tmpFile)
		return fmt.Errorf("writing binary: %w", err)
	}
	f.Close()

	if err := os.Rename(tmpFile, currentBin); err != nil {
		os.Remove(tmpFile)
		return fmt.Errorf("replacing binary: %w", err)
	}

	fmt.Printf("Updated to %s successfully.\n", release.TagName)
	return nil
}

func updateSource() error {
	cfg, err := config.Load()
	if err != nil {
		return err
	}

	dotctlPath := cfg.Dotctl.Path
	if dotctlPath == "" {
		return fmt.Errorf("dotctl.path not configured — set it in ~/.config/dotctl/config.toml")
	}

	fmt.Println("Pulling latest source...")
	pull := exec.Command("git", "-C", dotctlPath, "pull", "--ff-only")
	pull.Stdout = os.Stdout
	pull.Stderr = os.Stderr
	if err := pull.Run(); err != nil {
		return fmt.Errorf("git pull: %w", err)
	}

	currentBin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current binary: %w", err)
	}

	fmt.Println("Building...")
	build := exec.Command("go", "build", "-o", currentBin, ".")
	build.Dir = dotctlPath
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("go build: %w", err)
	}

	fmt.Println("Updated from source successfully.")
	return nil
}
