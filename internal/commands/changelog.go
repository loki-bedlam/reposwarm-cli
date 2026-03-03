package commands

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newChangelogCmd(currentVersion string) *cobra.Command {
	var allFlag bool
	var sinceFlag string

	cmd := &cobra.Command{
		Use:   "changelog [version]",
		Short: "Show release notes for a version",
		Long: `Show what's changed in a specific version or the current one.

Examples:
  reposwarm changelog                    # Current version's changes
  reposwarm changelog v1.3.50            # Specific version
  reposwarm changelog latest             # Latest release notes
  reposwarm changelog --all              # All versions
  reposwarm changelog --since v1.3.45    # Changes since a version`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			// --all mode: list all releases
			if allFlag {
				return showAllReleases()
			}

			// --since mode: show changes between a version and latest
			if sinceFlag != "" {
				return showChangesSince(sinceFlag, currentVersion)
			}

			// Single version mode
			targetVersion := currentVersion
			label := "current"

			if len(args) > 0 {
				if args[0] == "latest" {
					latest, _, err := getLatestRelease()
					if err != nil {
						return fmt.Errorf("checking latest release: %w", err)
					}
					targetVersion = latest
					label = "latest"
				} else {
					targetVersion = args[0]
					if len(targetVersion) > 0 && targetVersion[0] == 'v' {
						targetVersion = targetVersion[1:]
					}
					label = "v" + targetVersion
				}
			}

			if targetVersion == "" {
				return fmt.Errorf("no version specified and current version unknown")
			}

			changes, err := getChangelog("0.0.0", targetVersion)

			if flagJSON {
				result := map[string]any{"version": targetVersion, "changes": changes}
				if err != nil {
					result["error"] = err.Error()
					result["changes"] = []string{}
				}
				return output.JSON(result)
			}

			// --for-agent: plain text, no decorations
			if flagAgent {
				if err != nil {
					fmt.Printf("changelog v%s: error fetching (%v)\n", targetVersion, err)
					return nil
				}
				for _, line := range changes {
					fmt.Println(line)
				}
				return nil
			}

			output.F.Section(fmt.Sprintf("Changelog — %s (v%s)", label, targetVersion))
			fmt.Println()

			if err != nil {
				output.F.Warning(fmt.Sprintf("Could not fetch changelog: %v", err))
				fmt.Println()
				fmt.Printf("  View online: https://github.com/loki-bedlam/reposwarm-cli/releases/tag/v%s\n\n", targetVersion)
				return nil
			}

			if len(changes) == 0 {
				output.F.Info("No changelog entries found for this version")
				fmt.Println()
				fmt.Printf("  View online: https://github.com/loki-bedlam/reposwarm-cli/releases/tag/v%s\n\n", targetVersion)
				return nil
			}

			for _, line := range changes {
				fmt.Printf("  %s\n", line)
			}
			fmt.Println()

			if label == "current" {
				latest, _, err := getLatestRelease()
				if err == nil && latest != currentVersion {
					output.F.Info(fmt.Sprintf("Update available: v%s → v%s (run: reposwarm upgrade)", currentVersion, latest))
				}
			}

			fmt.Printf("  Full release: https://github.com/loki-bedlam/reposwarm-cli/releases/tag/v%s\n\n", targetVersion)
			return nil
		},
	}

	cmd.Flags().BoolVar(&allFlag, "all", false, "Show all release versions")
	cmd.Flags().StringVar(&sinceFlag, "since", "", "Show changes since a version (e.g. v1.3.45)")
	return cmd
}

type releaseEntry struct {
	TagName string `json:"tag_name"`
	Name    string `json:"name"`
	Body    string `json:"body"`
	Date    string `json:"published_at"`
}

func showAllReleases() error {
	releases, err := fetchAllReleases(20)
	if err != nil {
		return fmt.Errorf("fetching releases: %w", err)
	}

	if flagJSON {
		return output.JSON(releases)
	}

	if flagAgent {
		for _, r := range releases {
			body := strings.TrimSpace(r.Body)
			lines := strings.Split(body, "\n")
			count := 0
			for _, l := range lines {
				if strings.HasPrefix(strings.TrimSpace(l), "•") || strings.HasPrefix(strings.TrimSpace(l), "-") || strings.HasPrefix(strings.TrimSpace(l), "*") {
					count++
				}
			}
			fmt.Printf("%s (%d changes)\n", r.TagName, count)
		}
		return nil
	}

	output.F.Section("All Releases")
	fmt.Println()

	for _, r := range releases {
		body := strings.TrimSpace(r.Body)
		lines := strings.Split(body, "\n")
		var changes []string
		for _, l := range lines {
			t := strings.TrimSpace(l)
			if strings.HasPrefix(t, "•") || strings.HasPrefix(t, "-") || strings.HasPrefix(t, "*") {
				changes = append(changes, t)
			}
		}

		date := ""
		if r.Date != "" {
			if t, err := time.Parse(time.RFC3339, r.Date); err == nil {
				date = t.Format("Jan 2, 2006")
			}
		}

		fmt.Printf("  %s  %s\n", output.Bold(r.TagName), output.Dim(date))
		if len(changes) > 0 {
			for _, c := range changes {
				fmt.Printf("    %s\n", c)
			}
		} else {
			fmt.Printf("    %s\n", output.Dim("(no changelog)"))
		}
		fmt.Println()
	}

	return nil
}

func showChangesSince(since string, currentVersion string) error {
	if len(since) > 0 && since[0] == 'v' {
		since = since[1:]
	}

	releases, err := fetchAllReleases(50)
	if err != nil {
		return fmt.Errorf("fetching releases: %w", err)
	}

	// Collect all changes from releases newer than 'since'
	var allChanges []string
	var versions []string

	for _, r := range releases {
		ver := strings.TrimPrefix(r.TagName, "v")
		if ver == since {
			break // Stop at the 'since' version
		}
		versions = append(versions, r.TagName)

		body := strings.TrimSpace(r.Body)
		for _, line := range strings.Split(body, "\n") {
			t := strings.TrimSpace(line)
			if strings.HasPrefix(t, "•") || strings.HasPrefix(t, "-") || strings.HasPrefix(t, "*") {
				allChanges = append(allChanges, t)
			}
		}
	}

	if flagJSON {
		return output.JSON(map[string]any{
			"since":    since,
			"versions": versions,
			"changes":  allChanges,
		})
	}

	if flagAgent {
		for _, c := range allChanges {
			fmt.Println(c)
		}
		return nil
	}

	output.F.Section(fmt.Sprintf("Changes since v%s (%d releases)", since, len(versions)))
	fmt.Println()

	if len(allChanges) == 0 {
		output.F.Info("No changes found")
		return nil
	}

	for _, c := range allChanges {
		fmt.Printf("  %s\n", c)
	}
	fmt.Printf("\n  Versions: %s\n\n", strings.Join(versions, ", "))

	return nil
}

func fetchAllReleases(limit int) ([]releaseEntry, error) {
	url := fmt.Sprintf("https://api.github.com/repos/loki-bedlam/reposwarm-cli/releases?per_page=%d", limit)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("GitHub API returned %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var releases []releaseEntry
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, err
	}

	return releases, nil
}
