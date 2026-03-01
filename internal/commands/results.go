package commands

import (
	"fmt"
	"os"
	"strings"

	"github.com/loki-bedlam/reposwarm-cli/internal/api"
	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newResultsCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:     "results",
		Aliases: []string{"res"},
		Short:   "Browse architecture investigation results",
	}
	cmd.AddCommand(newResultsListCmd())
	cmd.AddCommand(newResultsSectionsCmd())
	cmd.AddCommand(newResultsReadCmd())
	cmd.AddCommand(newResultsMetaCmd())
	cmd.AddCommand(newResultsExportCmd())
	cmd.AddCommand(newResultsSearchCmd())
	cmd.AddCommand(newDiffCmd())
	cmd.AddCommand(newReportCmd())
	return cmd
}

func newResultsListCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List repos with investigation results",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			var result api.WikiReposResponse
			if err := client.Get(ctx(), "/wiki", &result); err != nil {
				return err
			}

			if flagJSON {
				return output.JSON(result.Repos)
			}

			F := output.F
			F.Section(fmt.Sprintf("Investigation Results (%d repos with results)", len(result.Repos)))
			headers := []string{"Repository", "Sections", "Last Updated"}
			var rows [][]string
			for _, r := range result.Repos {
				rows = append(rows, []string{
					r.Name,
					fmt.Sprint(r.SectionCount),
					r.LastUpdated,
				})
			}
			F.Table(headers, rows)
			F.Println()
			return nil
		},
	}
}

func newResultsSectionsCmd() *cobra.Command {
	return &cobra.Command{
		Use:     "sections <repo>",
		Aliases: []string{"show"},
		Short:   "List investigation sections for a repo",
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			var index api.WikiIndex
			if err := client.Get(ctx(), "/wiki/"+args[0], &index); err != nil {
				return err
			}

			if flagJSON {
				return output.JSON(index)
			}

			F := output.F
			F.Section(fmt.Sprintf("Results — %s (%d sections)", args[0], len(index.Sections)))
			headers := []string{"Section", "Created"}
			var rows [][]string
			for _, s := range index.Sections {
				rows = append(rows, []string{
					F.SectionIcon(s.Name()) + s.Name(),
					s.CreatedAt,
				})
			}
			F.Table(headers, rows)
			F.Println()
			return nil
		},
	}
}

func newResultsReadCmd() *cobra.Command {
	var raw bool

	cmd := &cobra.Command{
		Use:   "read <repo> [section]",
		Short: "Read investigation results (one section or all)",
		Long: `Read investigation results for a repository.

With section name: returns just that section.
Without section name: returns ALL sections concatenated.

Examples:
  reposwarm results read is-odd                  # All sections
  reposwarm results read is-odd hl_overview      # Single section
  reposwarm results read is-odd --raw > out.md   # Raw markdown`,
		Args: cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			repo := args[0]

			if len(args) == 2 {
				section := args[1]
				var content api.WikiContent
				if err := client.Get(ctx(), "/wiki/"+repo+"/"+section, &content); err != nil {
					return err
				}

				if flagJSON {
					return output.JSON(content)
				}
				if raw {
					fmt.Print(content.Content)
					return nil
				}
				F := output.F
				F.Section(fmt.Sprintf("Results — %s / %s", repo, section))
				F.Info(content.CreatedAt)
				F.Println()
				fmt.Println(content.Content)
				return nil
			}

			// All sections
			var index api.WikiIndex
			if err := client.Get(ctx(), "/wiki/"+repo, &index); err != nil {
				return err
			}

			if len(index.Sections) == 0 {
				return fmt.Errorf("no investigation results for %s", repo)
			}

			var allContent []api.WikiContent
			for _, s := range index.Sections {
				var content api.WikiContent
				if err := client.Get(ctx(), "/wiki/"+repo+"/"+s.Name(), &content); err != nil {
					output.F.Error(fmt.Sprintf("Failed to read %s: %s", s.Name(), err))
					continue
				}
				allContent = append(allContent, content)
			}

			if flagJSON {
				return output.JSON(allContent)
			}

			if raw {
				for _, c := range allContent {
					fmt.Printf("## %s\n\n%s\n\n", c.Section, c.Content)
				}
				return nil
			}

			F := output.F
			F.Section(fmt.Sprintf("Full Investigation — %s (%d sections)", repo, len(allContent)))
			for _, c := range allContent {
				F.Printf("--- %s ---\n", c.Section)
				F.Info(c.CreatedAt)
				F.Println()
				fmt.Println(c.Content)
				F.Println()
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&raw, "raw", false, "Output raw markdown (no formatting)")
	return cmd
}

func newResultsMetaCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "meta <repo> [section]",
		Short: "Show metadata for investigation results (no content)",
		Args:  cobra.RangeArgs(1, 2),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			repo := args[0]

			if len(args) == 2 {
				section := args[1]
				var content api.WikiContent
				if err := client.Get(ctx(), "/wiki/"+repo+"/"+section, &content); err != nil {
					return err
				}

				meta := map[string]any{
					"repo":         content.Repo,
					"section":      content.Section,
					"createdAt":    content.CreatedAt,
					"timestamp":    content.Timestamp,
					"referenceKey": content.ReferenceKey,
				}

				if flagJSON {
					return output.JSON(meta)
				}

				F := output.F
				F.Section("Section Metadata")
				F.KeyValue("Repository", repo)
				F.KeyValue("Section", section)
				F.KeyValue("Created", content.CreatedAt)
				F.KeyValue("Timestamp", fmt.Sprint(content.Timestamp))
				F.KeyValue("Ref Key", content.ReferenceKey)
				F.Println()
				return nil
			}

			// Repo-level metadata
			var index api.WikiIndex
			if err := client.Get(ctx(), "/wiki/"+repo, &index); err != nil {
				return err
			}

			meta := map[string]any{
				"repo":     repo,
				"sections": len(index.Sections),
				"hasDocs":  index.HasDocs,
			}
			if len(index.Sections) > 0 {
				meta["lastSection"] = index.Sections[len(index.Sections)-1].CreatedAt
			}

			if flagJSON {
				return output.JSON(meta)
			}

			F := output.F
			F.Section("Repository Metadata")
			F.KeyValue("Repository", repo)
			F.KeyValue("Sections", fmt.Sprint(len(index.Sections)))
			F.KeyValue("Has Docs", fmt.Sprint(index.HasDocs))
			if len(index.Sections) > 0 {
				F.KeyValue("Last Update", index.Sections[len(index.Sections)-1].CreatedAt)
			}
			F.Println()
			return nil
		},
	}
}

func newResultsExportCmd() *cobra.Command {
	var outputFile string

	cmd := &cobra.Command{
		Use:   "export <repo>",
		Short: "Export full investigation as markdown",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			repo := args[0]
			var index api.WikiIndex
			if err := client.Get(ctx(), "/wiki/"+repo, &index); err != nil {
				return err
			}

			var sb strings.Builder
			sb.WriteString(fmt.Sprintf("# %s — Architecture Investigation\n\n", repo))

			for _, s := range index.Sections {
				var content api.WikiContent
				if err := client.Get(ctx(), "/wiki/"+repo+"/"+s.Name(), &content); err != nil {
					continue
				}
				label := s.Label
				if label == "" {
					label = s.StepName
					if label == "" {
						label = s.Name()
					}
				}
				sb.WriteString(fmt.Sprintf("## %s\n\n%s\n\n---\n\n", label, content.Content))
			}

			if outputFile != "" {
				if err := os.WriteFile(outputFile, []byte(sb.String()), 0644); err != nil {
					return fmt.Errorf("writing file: %w", err)
				}
				output.F.Success(fmt.Sprintf("Exported %d sections to %s (%d bytes)",
					len(index.Sections), outputFile, sb.Len()))
				return nil
			}

			fmt.Print(sb.String())
			return nil
		},
	}

	cmd.Flags().StringVarP(&outputFile, "output", "o", "", "Output file path")
	return cmd
}

func newResultsSearchCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "search <query>",
		Short: "Search across all investigation results",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			query := strings.ToLower(args[0])

			var repoList api.WikiReposResponse
			if err := client.Get(ctx(), "/wiki", &repoList); err != nil {
				return err
			}

			type SearchHit struct {
				Repo    string `json:"repo"`
				Section string `json:"section"`
				Line    string `json:"line"`
			}

			var hits []SearchHit
			for _, r := range repoList.Repos {
				var index api.WikiIndex
				if err := client.Get(ctx(), "/wiki/"+r.Name, &index); err != nil {
					continue
				}
				for _, s := range index.Sections {
					var content api.WikiContent
					if err := client.Get(ctx(), "/wiki/"+r.Name+"/"+s.Name(), &content); err != nil {
						continue
					}
					for _, line := range strings.Split(content.Content, "\n") {
						if strings.Contains(strings.ToLower(line), query) {
							hits = append(hits, SearchHit{
								Repo:    r.Name,
								Section: s.Name(),
								Line:    strings.TrimSpace(line),
							})
						}
					}
				}
			}

			if flagJSON {
				return output.JSON(hits)
			}

			F := output.F
			F.Section(fmt.Sprintf("Search Results '%s' (%d hits)", args[0], len(hits)))
			for _, h := range hits {
				F.Printf("  %s/%s\n", h.Repo, h.Section)
				F.Printf("    %s\n\n", h.Line)
			}
			return nil
		},
	}
}
