package commands

import (
	"fmt"
	"sort"
	"strings"

	"github.com/loki-bedlam/reposwarm-cli/internal/api"
	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newResultsAuditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "audit",
		Short: "Validate all repos have complete investigation sections",
		Long: `Check every repo with results and verify it has all expected sections.
The expected section list is derived from the majority of completed repos.

Reports:
  - Total repos and section coverage
  - Any repos with missing or extra sections
  - Summary pass/fail`,
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			var repoList api.WikiReposResponse
			if err := client.Get(ctx(), "/wiki", &repoList); err != nil {
				return err
			}

			if len(repoList.Repos) == 0 {
				output.F.Info("No repos with results")
				return nil
			}

			type repoResult struct {
				Name     string   `json:"name"`
				Sections []string `json:"sections"`
				Missing  []string `json:"missing,omitempty"`
				Extra    []string `json:"extra,omitempty"`
				OK       bool     `json:"ok"`
			}

			// Collect section names from all repos
			sectionFreq := map[string]int{}
			repoSections := map[string][]string{}
			var fetchFailed []repoResult

			for _, r := range repoList.Repos {
				var index api.WikiIndex
				if err := client.Get(ctx(), "/wiki/"+r.Name, &index); err != nil {
					fetchFailed = append(fetchFailed, repoResult{Name: r.Name, OK: false, Missing: []string{"(fetch failed)"}})
					continue
				}
				var names []string
				for _, s := range index.Sections {
					name := s.Name()
					names = append(names, name)
					sectionFreq[name]++
				}
				repoSections[r.Name] = names
			}

			// Expected = sections in majority of repos
			totalRepos := len(repoList.Repos)
			threshold := totalRepos / 2
			var expectedSections []string
			for name, count := range sectionFreq {
				if count > threshold {
					expectedSections = append(expectedSections, name)
				}
			}
			sort.Strings(expectedSections)

			expectedSet := map[string]bool{}
			for _, s := range expectedSections {
				expectedSet[s] = true
			}

			// Audit each repo
			var results []repoResult
			results = append(results, fetchFailed...)
			passCount := 0

			for _, r := range repoList.Repos {
				sections, ok := repoSections[r.Name]
				if !ok {
					continue
				}
				gotSet := map[string]bool{}
				for _, s := range sections {
					gotSet[s] = true
				}
				var missing, extra []string
				for _, exp := range expectedSections {
					if !gotSet[exp] {
						missing = append(missing, exp)
					}
				}
				for _, got := range sections {
					if !expectedSet[got] {
						extra = append(extra, got)
					}
				}
				isOK := len(missing) == 0
				if isOK {
					passCount++
				}
				results = append(results, repoResult{
					Name:     r.Name,
					Sections: sections,
					Missing:  missing,
					Extra:    extra,
					OK:       isOK,
				})
			}

			failCount := len(results) - passCount

			if flagJSON {
				return output.JSON(map[string]any{
					"totalRepos":       totalRepos,
					"expectedSections": expectedSections,
					"passed":           passCount,
					"failed":           failCount,
					"repos":            results,
				})
			}

			F := output.F
			F.Section(fmt.Sprintf("Results Audit (%d repos, %d expected sections)", totalRepos, len(expectedSections)))
			F.Printf("Expected: %s\n\n", strings.Join(expectedSections, ", "))

			// Only show repos with issues (or all if verbose)
			hasIssues := false
			for _, r := range results {
				if !r.OK {
					hasIssues = true
					issues := ""
					if len(r.Missing) > 0 {
						issues += fmt.Sprintf("missing: %s", strings.Join(r.Missing, ", "))
					}
					if len(r.Extra) > 0 {
						if issues != "" {
							issues += "; "
						}
						issues += fmt.Sprintf("extra: %s", strings.Join(r.Extra, ", "))
					}
					F.Printf("FAIL  %-30s %d/%d  %s\n", r.Name, len(r.Sections), len(expectedSections), issues)
				}
			}
			if !hasIssues {
				F.Println()
			}

			F.CheckSummary(passCount, 0, failCount)
			return nil
		},
	}
}
