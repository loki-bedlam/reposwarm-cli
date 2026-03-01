package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"time"

	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newUpgradeCmd(currentVersion string) *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:     "upgrade",
		Aliases: []string{"update"},
		Short:   "Upgrade reposwarm CLI to the latest version",
		Long: `Downloads and installs the latest version from GitHub releases.

Examples:
  reposwarm upgrade           # Upgrade if newer version available
  reposwarm upgrade --force   # Reinstall even if same version`,
		RunE: func(cmd *cobra.Command, args []string) error {
			if !flagJSON {
				output.F.Section("RepoSwarm CLI Upgrade")
				fmt.Printf("  Current version: %s\n", output.Cyan("v"+currentVersion))
			}

			latestVer, downloadURL, err := getLatestRelease()
			if err != nil {
				return fmt.Errorf("checking for updates: %w", err)
			}

			if flagJSON {
				return output.JSON(map[string]any{
					"current":     currentVersion,
					"latest":      latestVer,
					"updateAvail": latestVer != currentVersion,
					"downloadUrl": downloadURL,
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

			fmt.Printf("  Downloading...")
			tmpFile, err := downloadBinary(downloadURL)
			if err != nil {
				return fmt.Errorf("download failed: %w", err)
			}
			defer os.Remove(tmpFile)
			fmt.Printf(" done\n")

			binPath, err := os.Executable()
			if err != nil {
				return fmt.Errorf("finding current binary: %w", err)
			}
			// Resolve symlinks
			binPath, err = filepath.EvalSymlinks(binPath)
			if err != nil {
				return fmt.Errorf("resolving binary path: %w", err)
			}

			fmt.Printf("  Installing to %s...", binPath)
			if err := safeReplaceBinary(tmpFile, binPath); err != nil {
				return fmt.Errorf("install failed: %w", err)
			}
			fmt.Printf(" done\n\n")

			output.F.Success(fmt.Sprintf("reposwarm v%s installed — restart your shell or run 'reposwarm version' to verify", latestVer))
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

	binaryName := fmt.Sprintf("reposwarm-%s-%s", runtime.GOOS, runtime.GOARCH)
	for _, asset := range release.Assets {
		if asset.Name == binaryName {
			return version, asset.BrowserDownloadURL, nil
		}
	}

	return version, "", fmt.Errorf("no binary found for %s in release assets", binaryName)
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

// safeReplaceBinary replaces the binary without corrupting the running process.
// On macOS/Linux, a running binary can be renamed but not overwritten safely.
// Strategy: rename old → write new → delete old.
func safeReplaceBinary(src, dst string) error {
	newData, err := os.ReadFile(src)
	if err != nil {
		return err
	}

	dir := filepath.Dir(dst)
	base := filepath.Base(dst)
	oldPath := filepath.Join(dir, "."+base+".old")

	// Remove any leftover from previous upgrade
	os.Remove(oldPath)

	// Rename running binary out of the way (safe on macOS/Linux)
	if err := os.Rename(dst, oldPath); err != nil {
		// Can't rename — try direct write as last resort
		if err := os.WriteFile(dst, newData, 0755); err != nil {
			return fmt.Errorf("cannot replace %s (try: sudo reposwarm upgrade): %w", dst, err)
		}
		return nil
	}

	// Write new binary to the original path
	if err := os.WriteFile(dst, newData, 0755); err != nil {
		// Rollback
		os.Rename(oldPath, dst)
		return fmt.Errorf("failed to write new binary: %w", err)
	}

	// Clean up old binary (best effort — may fail if still running, that's fine)
	go func() {
		time.Sleep(2 * time.Second)
		os.Remove(oldPath)
	}()

	return nil
}
