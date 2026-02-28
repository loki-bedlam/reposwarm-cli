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
	cmd.AddCommand(newResultsShowCmd())
	cmd.AddCommand(newResultsReadCmd())
	cmd.AddCommand(newResultsMetaCmd())
	cmd.AddCommand(newResultsExportCmd())
	cmd.AddCommand(newResultsSearchCmd())
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

			fmt.Printf("\n  %s (%d repos with results)\n\n", output.Bold("Investigation Results"), len(result.Repos))
			headers := []string{"Repository", "Sections", "Last Updated"}
			var rows [][]string
			for _, r := range result.Repos {
				rows = append(rows, []string{
					r.Name,
					fmt.Sprint(r.SectionCount),
					r.LastUpdated,
				})
			}
			output.Table(headers, rows)
			fmt.Println()
			return nil
		},
	}
}

func newResultsShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <repo>",
		Short: "List investigation sections for a repo",
		Args:  cobra.ExactArgs(1),
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

			fmt.Printf("\n  %s — %s (%d sections)\n\n",
				output.Bold("Results"), output.Bold(args[0]), len(index.Sections))
			headers := []string{"Section", "Label", "Created"}
			var rows [][]string
			for _, s := range index.Sections {
				rows = append(rows, []string{
					sectionIcon(s.ID) + " " + s.ID,
					s.Label,
					s.CreatedAt,
				})
			}
			output.Table(headers, rows)
			fmt.Println()
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
				// Single section
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
				fmt.Printf("\n  %s — %s / %s\n", output.Bold("Results"), output.Bold(repo), output.Cyan(section))
				fmt.Printf("  %s\n\n", output.Dim(content.CreatedAt))
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
				if err := client.Get(ctx(), "/wiki/"+repo+"/"+s.ID, &content); err != nil {
					output.Errorf("Failed to read %s: %s", s.ID, err)
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

			fmt.Printf("\n  %s — %s (%d sections)\n\n",
				output.Bold("Full Investigation"), output.Bold(repo), len(allContent))
			for _, c := range allContent {
				fmt.Printf("  %s %s\n", output.Bold("══"), output.Bold(c.Section))
				fmt.Printf("  %s\n\n", output.Dim(c.CreatedAt))
				fmt.Println(c.Content)
				fmt.Println()
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
				// Single section metadata
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

				fmt.Printf("\n  %s\n\n", output.Bold("Section Metadata"))
				fmt.Printf("  %s  %s\n", output.Dim("Repository  "), repo)
				fmt.Printf("  %s  %s\n", output.Dim("Section     "), section)
				fmt.Printf("  %s  %s\n", output.Dim("Created     "), content.CreatedAt)
				fmt.Printf("  %s  %d\n", output.Dim("Timestamp   "), content.Timestamp)
				fmt.Printf("  %s  %s\n", output.Dim("Ref Key     "), content.ReferenceKey)
				fmt.Println()
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

			fmt.Printf("\n  %s\n\n", output.Bold("Repository Metadata"))
			fmt.Printf("  %s  %s\n", output.Dim("Repository  "), repo)
			fmt.Printf("  %s  %d\n", output.Dim("Sections    "), len(index.Sections))
			fmt.Printf("  %s  %v\n", output.Dim("Has Docs    "), index.HasDocs)
			if len(index.Sections) > 0 {
				fmt.Printf("  %s  %s\n", output.Dim("Last Update "), index.Sections[len(index.Sections)-1].CreatedAt)
			}
			fmt.Println()
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
				if err := client.Get(ctx(), "/wiki/"+repo+"/"+s.ID, &content); err != nil {
					continue
				}
				sb.WriteString(fmt.Sprintf("## %s\n\n%s\n\n---\n\n", s.Label, content.Content))
			}

			if outputFile != "" {
				if err := os.WriteFile(outputFile, []byte(sb.String()), 0644); err != nil {
					return fmt.Errorf("writing file: %w", err)
				}
				output.Successf("Exported %d sections to %s (%d bytes)",
					len(index.Sections), outputFile, sb.Len())
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
					if err := client.Get(ctx(), "/wiki/"+r.Name+"/"+s.ID, &content); err != nil {
						continue
					}
					for _, line := range strings.Split(content.Content, "\n") {
						if strings.Contains(strings.ToLower(line), query) {
							hits = append(hits, SearchHit{
								Repo:    r.Name,
								Section: s.ID,
								Line:    strings.TrimSpace(line),
							})
						}
					}
				}
			}

			if flagJSON {
				return output.JSON(hits)
			}

			fmt.Printf("\n  %s '%s' (%d hits)\n\n", output.Bold("Search Results"), args[0], len(hits))
			for _, h := range hits {
				fmt.Printf("  %s/%s\n", output.Cyan(h.Repo), output.Dim(h.Section))
				fmt.Printf("    %s\n\n", h.Line)
			}
			return nil
		},
	}
}

func sectionIcon(id string) string {
	icons := map[string]string{
		"hl_overview": "📋", "module_deep_dive": "🔍", "dependencies": "📦",
		"core_entities": "🏗", "DBs": "💾", "APIs": "🌐", "api_surface": "🔌",
		"data_mapping": "🗺", "events": "⚡", "service_dependencies": "🔗",
		"deployment": "🚀", "authentication": "🔑", "authorization": "🛡",
		"security_check": "🔒", "prompt_security_check": "🤖",
		"monitoring": "📊", "ml_services": "🧠", "feature_flags": "🚩",
		"internals": "⚙",
	}
	if icon, ok := icons[id]; ok {
		return icon
	}
	return "📄"
}
