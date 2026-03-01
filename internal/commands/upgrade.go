package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"

	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newUpgradeCmd(currentVersion string) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "upgrade",
		Short: "Upgrade reposwarm CLI to the latest version",
		Long: `Downloads and installs the latest version from GitHub releases.

Examples:
  reposwarm upgrade           # Upgrade if newer version available
  reposwarm upgrade --force   # Reinstall even if same version`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !flagJSON {
				output.F.Section("RepoSwarm CLI Upgrade")
				fmt.Printf("  Current version: %s\n", output.Cyan("v"+currentVersion))
			}

			// Check latest release on GitHub
			latestVer, downloadURL, err := getLatestRelease()
			if err != nil {
				return fmt.Errorf("checking for updates: %w", err)
			}

			if flagJSON {
				return output.JSON(map[string]any{
					"current":      currentVersion,
					"latest":       latestVer,
					"updateAvail":  latestVer != currentVersion,
					"downloadUrl":  downloadURL,
				})
			}

			fmt.Printf("  Latest version:  %s\n", output.Cyan("v"+latestVer))

			if latestVer == currentVersion && !force {
				fmt.Printf("\n  %s\n\n", output.Green("Already up to date!"))
				return nil
			}

			if latestVer == currentVersion && force {
				output.Infof("Reinstalling v%s (--force)", currentVersion)
			} else {
				output.Infof("Upgrading v%s → v%s", currentVersion, latestVer)
			}

			// Download
			fmt.Printf("  Downloading...")
			tmpFile, err := downloadBinary(downloadURL)
			if err != nil {
				return fmt.Errorf("download failed: %w", err)
			}
			defer os.Remove(tmpFile)
			fmt.Printf(" done\n")

			// Find current binary path
			binPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("finding current binary: %w", err)
			}

			// Replace binary
			fmt.Printf("  Installing to %s...", binPath)
			if err := replaceBinary(tmpFile, binPath); err != nil {
				return fmt.Errorf("install failed: %w", err)
			}
			fmt.Printf(" done\n")

			// Verify
			out, err := exec.Command(binPath, "--version").Output()
			if err != nil {
				output.Errorf("Verification failed: %s", err)
			} else {
				output.F.Success(strings.TrimSpace(string(out)))
			}

			return nil
		},
	}

	cmd.Flags().BoolVar(&force, "force", false, "Reinstall even if same version")
	return cmd
}

type ghRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

func getLatestRelease() (version, downloadURL string, err error) {
	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get("https://api.github.com/repos/loki-bedlam/reposwarm-cli/releases/latest")
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", "", fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	var release ghRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return "", "", err
	}

	version = release.TagName
	if len(version) > 0 && version[0] == 'v' {
		version = version[1:]
	}

	// Find binary for this platform
	binaryName := fmt.Sprintf("reposwarm-%s-%s", runtime.GOOS, runtime.GOARCH)
	for _, asset := range release.Assets {
		if asset.Name == binaryName {
			return version, asset.BrowserDownloadURL, nil
		}
	}

	// Fallback to CDN
	cdnURL := fmt.Sprintf("https://db22kd0yixg8j.cloudfront.net/assets/reposwarm-cli/latest/%s", binaryName)
	return version, cdnURL, nil
}

func downloadBinary(url string) (string, error) {
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return "", fmt.Errorf("HTTP %d from %s", resp.StatusCode, url)
	}

	tmp, err := os.CreateTemp("", "reposwarm-upgrade-*")
	if err != nil {
		return "", err
	}

	if _, err := io.Copy(tmp, resp.Body); err != nil {
		tmp.Close()
		os.Remove(tmp.Name())
		return "", err
	}
	tmp.Close()
	os.Chmod(tmp.Name(), 0755)

	return tmp.Name(), nil
}

func replaceBinary(src, dst string) error {
	// Read new binary
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	// Try direct write
	if err := os.WriteFile(dst, data, 0755); err != nil {
		// May need sudo — try rename approach
		backup := dst + ".old"
		os.Rename(dst, backup)
		if err := os.WriteFile(dst, data, 0755); err != nil {
			os.Rename(backup, dst) // rollback
			return fmt.Errorf("cannot write to %s (try: sudo reposwarm upgrade)", dst)
		}
		os.Remove(backup)
	}

	return nil
}
