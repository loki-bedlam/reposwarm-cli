package commands

import (
	"fmt"
	"time"

	"github.com/reposwarm/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

func newLogsCmd() *cobra.Command {
	var tail bool
	var lines int

	cmd := &cobra.Command{
		Use:   "logs [service]",
		Short: "View service logs via API",
		Long: `View logs for a RepoSwarm service.

Available services: api, worker, temporal, ui

If no service is specified, shows logs from all services.`,
		Args: friendlyMaxArgs(1, "reposwarm logs [service]\n\nServices: api, worker, temporal, ui\n\nExample:\n  reposwarm logs worker -n 100"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			services := []string{"api", "worker", "temporal", "ui"}
			if len(args) > 0 {
				svc := args[0]
				valid := false
				for _, s := range services {
					if s == svc {
						valid = true
						break
					}
				}
				if !valid {
					return fmt.Errorf("invalid service: %s (must be one of: api, worker, temporal, ui)", svc)
				}
				services = []string{svc}
			}

			if tail {
				// Follow mode: poll every 2s
				for {
					for _, svc := range services {
						var resp struct {
							Service string   `json:"service"`
							Lines   []string `json:"lines"`
							Total   int      `json:"total"`
						}
						path := fmt.Sprintf("/services/%s/logs?lines=%d", svc, lines)
						if err := client.Get(ctx(), path, &resp); err != nil {
							continue
						}
						for _, l := range resp.Lines {
							if len(services) > 1 {
								fmt.Printf("[%s] %s\n", output.Cyan(svc), l)
							} else {
								fmt.Println(l)
							}
						}
					}
					time.Sleep(2 * time.Second)
				}
			}

			// Non-follow: single fetch
			for _, svc := range services {
				var resp struct {
					Service string   `json:"service"`
					LogFile *string  `json:"logFile"`
					Lines   []string `json:"lines"`
					Total   int      `json:"total"`
				}
				path := fmt.Sprintf("/services/%s/logs?lines=%d", svc, lines)
				if err := client.Get(ctx(), path, &resp); err != nil {
					if !flagJSON {
						output.F.Warning(fmt.Sprintf("Error reading %s logs: %v", svc, err))
					}
					continue
				}

				if flagJSON {
					output.JSON(resp)
					continue
				}

				if len(resp.Lines) == 0 {
					continue
				}

				output.F.Section(fmt.Sprintf("%s logs (%d lines)", svc, len(resp.Lines)))
				for _, l := range resp.Lines {
					fmt.Printf("  %s\n", l)
				}
				output.F.Println()
			}

			return nil
		},
	}

	cmd.Flags().BoolVarP(&tail, "tail", "f", false, "Follow/stream logs")
	cmd.Flags().IntVarP(&lines, "lines", "n", 50, "Number of lines to show")
	return cmd
}
