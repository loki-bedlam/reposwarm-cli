package commands

import (
	"fmt"

	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newChangelogCmd(currentVersion string) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "changelog [version]",
		Short: "Show release notes for a version",
		Long: `Show what's changed in a specific version or the current one.

Examples:
  reposwarm changelog             # Current version's changes
  reposwarm changelog v1.3.50     # Specific version
  reposwarm changelog latest      # Latest release notes`,
		Args: cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
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
					// Strip leading 'v' for consistency
					if len(targetVersion) > 0 && targetVersion[0] == 'v' {
						targetVersion = targetVersion[1:]
					}
					label = "v" + targetVersion
				}
			}

			if targetVersion == "" {
				return fmt.Errorf("no version specified and current version unknown")
			}

			if flagJSON {
				changes, err := getChangelog("0.0.0", targetVersion)
				if err != nil {
					return output.JSON(map[string]any{
						"version": targetVersion,
						"changes": []string{},
						"error":   err.Error(),
					})
				}
				return output.JSON(map[string]any{
					"version": targetVersion,
					"changes": changes,
				})
			}

			output.F.Section(fmt.Sprintf("Changelog — %s (v%s)", label, targetVersion))
			fmt.Println()

			changes, err := getChangelog("0.0.0", targetVersion)
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

			// Show context
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

	return cmd
}
