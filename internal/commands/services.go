package commands

import (
	"fmt"
	"strings"
	"time"

	"github.com/reposwarm/reposwarm-cli/internal/api"
	"github.com/reposwarm/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

var knownServices = []string{"api", "worker", "temporal", "ui"}

func newServicesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "services",
		Short: "Show all running RepoSwarm services",
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			var services []api.ServiceInfo
			if err := client.Get(ctx(), "/services", &services); err != nil {
				return fmt.Errorf("failed to list services: %w", err)
			}

			if flagJSON {
				return output.JSON(services)
			}

			F := output.F
			running := 0
			for _, s := range services {
				if s.Status == "running" {
					running++
				}
			}
			F.Section(fmt.Sprintf("Services (%d/%d running)", running, len(services)))

			headers := []string{"Service", "PID", "Status", "Port", "Manager"}
			var rows [][]string
			for _, s := range services {
				pid := "—"
				if s.PID > 0 {
					pid = fmt.Sprint(s.PID)
				}
				statusStr := s.Status
				switch s.Status {
				case "running":
					statusStr = output.Green("running")
				case "stopped":
					statusStr = output.Dim("stopped")
				}
				port := "—"
				if s.Port > 0 {
					port = fmt.Sprint(s.Port)
				}
				manager := s.Manager
				if manager == "" {
					manager = "—"
				}
				rows = append(rows, []string{s.Name, pid, statusStr, port, manager})
			}

			output.Table(headers, rows)
			F.Println()
			return nil
		},
	}
	return cmd
}

func newRestartCmd() *cobra.Command {
	var wait bool
	var timeout int

	cmd := &cobra.Command{
		Use:   "restart [service]",
		Short: "Restart one or all RepoSwarm services",
		Long: `Restart a RepoSwarm service or all services.

Examples:
  reposwarm restart           # Restart all services
  reposwarm restart worker    # Restart the worker
  reposwarm restart api       # Restart the API server`,
		Args: friendlyMaxArgs(1, "reposwarm restart [service]\n\nServices: api, worker, temporal, ui\n\nExample:\n  reposwarm restart worker"),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := getClient()
			if err != nil {
				return err
			}

			services := knownServices
			if len(args) > 0 {
				svc := args[0]
				if !isKnownService(svc) {
					return fmt.Errorf("unknown service: %s (must be one of: %s)",
						svc, strings.Join(knownServices, ", "))
				}
				services = []string{svc}
			}

			var results []map[string]any
			for _, svc := range services {
				var resp map[string]any
				err := client.Post(ctx(), "/services/"+svc+"/restart", nil, &resp)

				result := map[string]any{"service": svc}
				if err != nil {
					result["status"] = "error"
					result["error"] = err.Error()
				} else {
					result["status"] = resp["status"]
					result["pid"] = resp["pid"]
				}
				results = append(results, result)

				if !flagJSON {
					if err != nil {
						output.F.Printf("  %s %s: %v\n", output.Red("✗"), svc, err)
					} else {
						output.F.Printf("  %s %s restarted", output.Green("✓"), svc)
						if pid, ok := resp["pid"]; ok && pid != nil && pid != float64(0) {
							output.F.Printf(" (PID %.0f)", pid)
						}
						fmt.Println()
					}
				}
			}

			if wait && !flagJSON {
				// Wait for health
				output.F.Info("Waiting for healthy...")
				deadline := time.Now().Add(time.Duration(timeout) * time.Second)
				for time.Now().Before(deadline) {
					_, err := client.Health(ctx())
					if err == nil {
						output.Successf("Services healthy")
						break
					}
					time.Sleep(2 * time.Second)
				}
			}

			if flagJSON {
				return output.JSON(results)
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&wait, "wait", true, "Wait for service to be healthy after restart")
	cmd.Flags().IntVar(&timeout, "timeout", 30, "Max seconds to wait for healthy")
	return cmd
}

func newStopCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "stop [service]",
		Short: "Stop a RepoSwarm service",
		Args:  friendlyExactArgs(1, "reposwarm stop <service>\n\nServices: api, worker, temporal, ui\n\nExample:\n  reposwarm stop worker"),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc := args[0]
			if !isKnownService(svc) {
				return fmt.Errorf("unknown service: %s", svc)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			var resp map[string]any
			if err := client.Post(ctx(), "/services/"+svc+"/stop", nil, &resp); err != nil {
				return fmt.Errorf("failed to stop %s: %w", svc, err)
			}

			if flagJSON {
				return output.JSON(resp)
			}

			status, _ := resp["status"].(string)
			if status == "stopped" {
				output.Successf("%s stopped", svc)
			} else if status == "not_found" {
				output.F.Info(fmt.Sprintf("%s is not running", svc))
			} else {
				output.F.Error(fmt.Sprintf("Unexpected status: %s", status))
			}
			return nil
		},
	}
	return cmd
}

func newStartCmd() *cobra.Command {
	var wait bool

	cmd := &cobra.Command{
		Use:   "start [service]",
		Short: "Start a RepoSwarm service",
		Args:  friendlyExactArgs(1, "reposwarm start <service>\n\nServices: api, worker, temporal, ui\n\nExample:\n  reposwarm start worker"),
		RunE: func(cmd *cobra.Command, args []string) error {
			svc := args[0]
			if !isKnownService(svc) {
				return fmt.Errorf("unknown service: %s", svc)
			}

			client, err := getClient()
			if err != nil {
				return err
			}

			var resp map[string]any
			if err := client.Post(ctx(), "/services/"+svc+"/start", nil, &resp); err != nil {
				return fmt.Errorf("failed to start %s: %w", svc, err)
			}

			if flagJSON {
				return output.JSON(resp)
			}

			output.Successf("%s started", svc)
			if wait {
				output.F.Info("Waiting for healthy...")
				deadline := time.Now().Add(30 * time.Second)
				for time.Now().Before(deadline) {
					health, err := client.Health(ctx())
					if err == nil && health.Worker.Connected {
						output.Successf("%s healthy", svc)
						return nil
					}
					time.Sleep(2 * time.Second)
				}
				output.F.Warning("Health check timed out")
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&wait, "wait", true, "Wait for service to be healthy")
	return cmd
}

func isKnownService(name string) bool {
	for _, s := range knownServices {
		if s == name {
			return true
		}
	}
	if strings.HasPrefix(name, "worker-") {
		return true
	}
	return false
}
