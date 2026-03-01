package commands

import (
	"fmt"
	"strings"

	"github.com/loki-bedlam/reposwarm-cli/internal/api"
	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func init() {
	// Registered via addSyncCmd() in repos.go
}

func newReposSyncCmd() *cobra.Command {
	var removeExternal, dryRun bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Sync tracked repos with CodeCommit (discover + optionally remove external)",
		Long: `Discovers all CodeCommit repositories and adds them to tracking.
With --remove-external, also removes any non-CodeCommit repos (GitHub, etc).

Examples:
  reposwarm repos sync                     # Add new CodeCommit repos only
  reposwarm repos sync --remove-external   # Add CodeCommit + remove GitHub repos
  reposwarm repos sync --dry-run           # Preview what would change`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			// Step 1: Discover CodeCommit repos
			var discoverResult api.DiscoverResult
			if !dryRun {
				if err := client.Post(ctx(), "/repos/discover", nil, &discoverResult); err != nil {
					return fmt.Errorf("discover failed: %w", err)
				}
			}

			// Step 2: List all repos
			var repos []api.Repository
			if err := client.Get(ctx(), "/repos", &repos); err != nil {
				return fmt.Errorf("list repos failed: %w", err)
			}

			// Find external repos (non-CodeCommit)
			var external []api.Repository
			var codecommitCount int
			for _, r := range repos {
				if isCodeCommitURL(r.URL) || strings.EqualFold(r.Source, "CodeCommit") {
					codecommitCount++
				} else {
					external = append(external, r)
				}
			}

			if flagJSON {
				result := map[string]any{
					"discovered":      discoverResult.Discovered,
					"added":           discoverResult.Added,
					"codecommitRepos": codecommitCount,
					"externalRepos":   len(external),
					"removedExternal": 0,
					"dryRun":          dryRun,
				}
				if removeExternal && !dryRun {
					result["removedExternal"] = len(external)
				}
				return output.JSON(result)
			}

			fmt.Println()
			if !dryRun {
				output.Successf("Discovered %s CodeCommit repos, added %s new",
					output.Bold(fmt.Sprint(discoverResult.Discovered)),
					output.Bold(fmt.Sprint(discoverResult.Added)))
			}
			output.Infof("CodeCommit repos: %d, External repos: %d", codecommitCount, len(external))

			// Step 3: Remove external repos if requested
			if removeExternal && len(external) > 0 {
				removed := 0
				for _, r := range external {
					if dryRun {
						output.Infof("[dry-run] Would remove %s (%s)", output.Bold(r.Name), r.URL)
					} else {
						var result any
						if err := client.Delete(ctx(), "/repos/"+r.Name, &result); err != nil {
							output.Errorf("Failed to remove %s: %v", r.Name, err)
							continue
						}
						output.Successf("Removed %s (%s)", r.Name, r.Source)
						removed++
					}
				}
				if !dryRun {
					output.Successf("Removed %d external repos", removed)
				}
			} else if len(external) > 0 && !removeExternal {
				output.Infof("Use --remove-external to remove %d non-CodeCommit repos", len(external))
			}

			fmt.Println()
			return nil
		},
	}

	cmd.Flags().BoolVar(&removeExternal, "remove-external", false, "Remove non-CodeCommit repositories")
	cmd.Flags().BoolVar(&dryRun, "dry-run", false, "Preview changes without applying")
	return cmd
}

func isCodeCommitURL(url string) bool {
	return strings.Contains(url, "codecommit") || strings.Contains(url, "git-codecommit")
}
