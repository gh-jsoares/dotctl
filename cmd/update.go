package cmd

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/gh-jsoares/dotctl/internal/config"
	"github.com/spf13/cobra"
)

var (
	updateFromSource bool
	updateSourcePath string
	updateCheck      bool
)

var updateCmd = &cobra.Command{
	Use:   "update",
	Short: "Update dotctl to the latest version",
	Long:  "Download the latest dotctl release from GitHub and replace the current binary, or rebuild from source.",
	RunE:  runUpdate,
}

func init() {
	rootCmd.AddCommand(updateCmd)
	updateCmd.Flags().BoolVar(&updateFromSource, "from-source", false, "rebuild from source (git pull + go build)")
	updateCmd.Flags().StringVar(&updateSourcePath, "source-path", ".", "path to dotctl source repo (with --from-source)")
	updateCmd.Flags().BoolVar(&updateCheck, "check", false, "only check if an update is available")
}

var (
	updateGreen  = lipgloss.NewStyle().Foreground(lipgloss.Color("2"))
	updateYellow = lipgloss.NewStyle().Foreground(lipgloss.Color("3"))
	updateDim    = lipgloss.NewStyle().Foreground(lipgloss.Color("8"))
	updateBold   = lipgloss.NewStyle().Bold(true)
)

func runUpdate(cmd *cobra.Command, args []string) error {
	if updateFromSource {
		return updateSource()
	}

	cfg, err := config.Load()
	if err != nil {
		return err
	}

	if cfg.Dotctl.Remote == "" {
		return fmt.Errorf("dotctl.remote must be set in %s", config.DefaultConfigPath())
	}

	latest, downloadURL, err := fetchLatestRelease(cfg)
	if err != nil {
		return err
	}

	current := strings.TrimPrefix(version, "v")
	latestClean := strings.TrimPrefix(latest, "v")
	if current == "dev" {
		fmt.Fprintf(os.Stdout, "  %s dotctl built from source %s\n", updateGreen.Render("✓"), updateDim.Render("(dev)"))
		return nil
	}
	if current == latestClean {
		fmt.Fprintf(os.Stdout, "  %s dotctl is up to date %s\n", updateGreen.Render("✓"), updateDim.Render("(v"+current+")"))
		return nil
	}

	fmt.Fprintf(os.Stdout, "  %s %s → %s\n",
		updateYellow.Render("⬆"),
		updateDim.Render(current),
		updateBold.Render(latest),
	)

	if updateCheck {
		return nil
	}

	if downloadURL == "" {
		return fmt.Errorf("no release asset for %s/%s in %s", runtime.GOOS, runtime.GOARCH, latest)
	}

	return installRelease(latest, downloadURL)
}

func fetchLatestRelease(cfg *config.Config) (tag string, assetURL string, err error) {
	owner, repo := parseGitRemote(cfg.Dotctl.Remote)
	if owner == "" || repo == "" {
		return "", "", fmt.Errorf("could not parse owner/repo from dotctl.remote %q", cfg.Dotctl.Remote)
	}

	url := fmt.Sprintf("https://api.github.com/repos/%s/%s/releases/latest", owner, repo)
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", "", fmt.Errorf("checking latest release: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("GitHub API returned %d (repo may be private or not yet published)", resp.StatusCode)
	}

	var release struct {
		TagName string `json:"tag_name"`
		Assets  []struct {
			Name               string `json:"name"`
			BrowserDownloadURL string `json:"browser_download_url"`
		} `json:"assets"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", fmt.Errorf("parsing release: %w", err)
	}

	assetName := fmt.Sprintf("dotctl_%s_%s.tar.gz", runtime.GOOS, runtime.GOARCH)
	for _, a := range release.Assets {
		if a.Name == assetName {
			return release.TagName, a.BrowserDownloadURL, nil
		}
	}

	return release.TagName, "", nil
}

func installRelease(tag, downloadURL string) error {
	currentBin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current binary: %w", err)
	}

	// Check write permission, elevate if needed
	if f, err := os.OpenFile(currentBin, os.O_WRONLY, 0o755); err != nil {
		if os.IsPermission(err) {
			fmt.Fprintf(os.Stdout, "  %s\n", updateDim.Render("elevating to install..."))
			sudoArgs := append([]string{currentBin}, os.Args[1:]...)
			sudoCmd := exec.Command("sudo", sudoArgs...)
			sudoCmd.Stdin = os.Stdin
			sudoCmd.Stdout = os.Stdout
			sudoCmd.Stderr = os.Stderr
			return sudoCmd.Run()
		}
		return err
	} else {
		f.Close()
	}

	fmt.Fprintf(os.Stdout, "  %s\n", updateDim.Render("downloading..."))

	client := &http.Client{Timeout: 60 * time.Second}
	dlResp, err := client.Get(downloadURL)
	if err != nil {
		return fmt.Errorf("downloading: %w", err)
	}
	defer dlResp.Body.Close()

	if dlResp.StatusCode != http.StatusOK {
		return fmt.Errorf("download failed: HTTP %d", dlResp.StatusCode)
	}

	tmpDir, err := os.MkdirTemp("", "dotctl-update-*")
	if err != nil {
		return fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := extractTarGz(dlResp.Body, tmpDir); err != nil {
		return fmt.Errorf("extracting archive: %w", err)
	}

	binaryName := fmt.Sprintf("dotctl_%s_%s", runtime.GOOS, runtime.GOARCH)
	tmpBin := filepath.Join(tmpDir, binaryName)
	if err := os.Chmod(tmpBin, 0o755); err != nil {
		return fmt.Errorf("setting permissions: %w", err)
	}

	if err := os.Rename(tmpBin, currentBin); err != nil {
		return fmt.Errorf("replacing binary: %w", err)
	}

	// Install man pages if present
	manSrc := filepath.Join(tmpDir, "man")
	if entries, err := os.ReadDir(manSrc); err == nil && len(entries) > 0 {
		manDest := "/usr/local/share/man/man1"
		os.MkdirAll(manDest, 0o755)
		for _, e := range entries {
			if strings.HasSuffix(e.Name(), ".1") {
				src := filepath.Join(manSrc, e.Name())
				dst := filepath.Join(manDest, e.Name())
				data, _ := os.ReadFile(src)
				if data != nil {
					os.WriteFile(dst, data, 0o644)
				}
			}
		}
	}

	fmt.Fprintf(os.Stdout, "  %s updated to %s\n", updateGreen.Render("✓"), updateBold.Render(tag))
	return nil
}

func extractTarGz(r io.Reader, dest string) error {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gz.Close()

	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(dest, filepath.Clean(hdr.Name))
		if !strings.HasPrefix(target, dest) {
			continue
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			os.MkdirAll(target, 0o755)
		case tar.TypeReg:
			os.MkdirAll(filepath.Dir(target), 0o755)
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return err
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return err
			}
			f.Close()
		}
	}
	return nil
}

func updateSource() error {
	fmt.Fprintf(os.Stdout, "  %s\n", updateDim.Render("pulling latest source..."))
	pull := exec.Command("git", "-C", updateSourcePath, "pull", "--ff-only")
	pull.Stdout = os.Stdout
	pull.Stderr = os.Stderr
	if err := pull.Run(); err != nil {
		return fmt.Errorf("git pull: %w", err)
	}

	currentBin, err := os.Executable()
	if err != nil {
		return fmt.Errorf("finding current binary: %w", err)
	}

	fmt.Fprintf(os.Stdout, "  %s\n", updateDim.Render("building..."))
	build := exec.Command("go", "build", "-o", currentBin, ".")
	build.Dir = updateSourcePath
	build.Stdout = os.Stdout
	build.Stderr = os.Stderr
	if err := build.Run(); err != nil {
		return fmt.Errorf("go build: %w", err)
	}

	fmt.Fprintf(os.Stdout, "  %s updated from source\n", updateGreen.Render("✓"))
	return nil
}

// CheckForUpdate prints a notice if a newer version is available.
// Called at the end of sync if the last check was >24h ago.
func CheckForUpdate() {
	cacheFile := config.StateDir() + "/last-update-check"

	// Only check once per day
	if info, err := os.Stat(cacheFile); err == nil {
		if time.Since(info.ModTime()) < 24*time.Hour {
			return
		}
	}

	// Touch the file regardless of outcome
	os.MkdirAll(config.StateDir(), 0o755)
	os.WriteFile(cacheFile, []byte(time.Now().Format(time.RFC3339)), 0o644)

	cfg, err := config.Load()
	if err != nil || cfg.Dotctl.Remote == "" {
		return
	}

	tag, _, err := fetchLatestRelease(cfg)
	if err != nil {
		return
	}

	current := strings.TrimPrefix(version, "v")
	latestClean := strings.TrimPrefix(tag, "v")
	if current == "dev" || current == latestClean {
		return
	}

	fmt.Fprintf(os.Stderr, "\n  %s %s available %s\n",
		updateYellow.Render("⬆"),
		updateBold.Render("dotctl "+tag),
		updateDim.Render("(run: dotctl update)"),
	)
}
