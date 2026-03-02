package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/loki-bedlam/reposwarm-cli/internal/api"
	"github.com/loki-bedlam/reposwarm-cli/internal/config"
	"github.com/loki-bedlam/reposwarm-cli/internal/output"
	"github.com/spf13/cobra"
)

var knownServices = []string{"api", "worker", "temporal", "ui"}

func newServicesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "services",
		Short: "Show all running RepoSwarm services",
		RunE: func(cmd *cobra.Command, args []string) error {
			services := detectServices()

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
	var allWorkers bool

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
			services := knownServices
			if len(args) > 0 {
				svc := args[0]
				if !isKnownService(svc) {
					return fmt.Errorf("unknown service: %s (must be one of: %s)",
						svc, strings.Join(knownServices, ", "))
				}
				services = []string{svc}
			}

			if allWorkers {
				// Future: expand to all worker instances
				services = []string{"worker"}
			}

			var results []map[string]any
			for _, svc := range services {
				result := restartServiceWithStatus(svc, wait, timeout)
				results = append(results, result)

				if !flagJSON {
					status := result["status"].(string)
					switch status {
					case "restarted":
						output.F.Printf("  %s %s restarted", output.Green("✓"), svc)
						if pid, ok := result["pid"]; ok {
							output.F.Printf(" (PID %v)", pid)
						}
						fmt.Println()
					case "healthy":
						output.F.Printf("  %s %s healthy\n", output.Green("✓"), svc)
					case "started":
						output.F.Printf("  %s %s started\n", output.Green("✓"), svc)
					case "not_found":
						output.F.Printf("  %s %s not found (not running)\n", output.Yellow("⚠"), svc)
					case "error":
						output.F.Printf("  %s %s: %v\n", output.Red("✗"), svc, result["error"])
					}
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
	cmd.Flags().BoolVar(&allWorkers, "all-workers", false, "Restart all worker instances")
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

			result := stopService(svc)

			if flagJSON {
				return output.JSON(result)
			}

			if result["status"] == "stopped" {
				output.Successf("%s stopped (was PID %v)", svc, result["pid"])
			} else if result["status"] == "not_found" {
				output.F.Info(fmt.Sprintf("%s is not running", svc))
			} else {
				output.F.Error(fmt.Sprintf("Failed to stop %s: %v", svc, result["error"]))
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

			result := startService(svc)

			if flagJSON {
				return output.JSON(result)
			}

			if result["status"] == "started" {
				output.Successf("%s started (PID %v)", svc, result["pid"])
				if wait {
					output.F.Info("Waiting for healthy...")
					if waitForHealthy(svc, 30) {
						output.Successf("%s healthy", svc)
					} else {
						output.F.Warning("Health check timed out")
					}
				}
			} else {
				output.F.Error(fmt.Sprintf("Failed to start %s: %v", svc, result["error"]))
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&wait, "wait", true, "Wait for service to be healthy")
	return cmd
}

// Service detection and management

func detectServices() []api.ServiceInfo {
	cfg, _ := config.Load()
	installDir := ""
	if cfg != nil {
		installDir = cfg.EffectiveInstallDir()
	}

	var services []api.ServiceInfo

	// Check each known service
	portMap := map[string]int{"api": 3000, "temporal": 7233, "ui": 3001}
	if cfg != nil {
		if p, err := strconv.Atoi(cfg.EffectiveAPIPort()); err == nil {
			portMap["api"] = p
		}
		if p, err := strconv.Atoi(cfg.EffectiveTemporalPort()); err == nil {
			portMap["temporal"] = p
		}
		if p, err := strconv.Atoi(cfg.EffectiveUIPort()); err == nil {
			portMap["ui"] = p
		}
	}

	for _, svc := range knownServices {
		info := api.ServiceInfo{
			Name:   svc,
			Status: "stopped",
			Port:   portMap[svc],
		}

		// Check PID file
		pid := findServicePID(installDir, svc)
		if pid > 0 {
			info.PID = pid
			if isProcessRunning(pid) {
				info.Status = "running"
			}
		}

		// Detect manager
		info.Manager = detectManager(svc, installDir)

		services = append(services, info)
	}

	return services
}

func findServicePID(installDir, service string) int {
	// Check PID files
	pidPaths := []string{
		filepath.Join(installDir, service+".pid"),
		filepath.Join(installDir, "pids", service+".pid"),
		filepath.Join(installDir, service, service+".pid"),
	}
	for _, p := range pidPaths {
		data, err := os.ReadFile(p)
		if err != nil {
			continue
		}
		pid, err := strconv.Atoi(strings.TrimSpace(string(data)))
		if err == nil && pid > 0 {
			return pid
		}
	}

	// Fallback: grep process table
	return findPIDByName(service)
}

func findPIDByName(service string) int {
	patterns := map[string][]string{
		"api":      {"node", "reposwarm-api"},
		"worker":   {"python", "src.worker"},
		"temporal": {"temporal-server"},
		"ui":       {"next-server", "reposwarm-ui"},
	}

	for _, pattern := range patterns[service] {
		out, err := exec.Command("pgrep", "-f", pattern).Output()
		if err == nil {
			lines := strings.Split(strings.TrimSpace(string(out)), "\n")
			if len(lines) > 0 {
				pid, err := strconv.Atoi(lines[0])
				if err == nil {
					return pid
				}
			}
		}
	}
	return 0
}

func isProcessRunning(pid int) bool {
	process, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	err = process.Signal(syscall.Signal(0))
	return err == nil
}

func detectManager(service, installDir string) string {
	// Check systemd
	out, err := exec.Command("systemctl", "is-active", "reposwarm-"+service).Output()
	if err == nil && strings.TrimSpace(string(out)) == "active" {
		return "systemd"
	}

	// Check docker
	out, err = exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", service), "--format", "{{.Names}}").Output()
	if err == nil && strings.TrimSpace(string(out)) != "" {
		return "docker"
	}

	// Check PM2
	out, err = exec.Command("pm2", "id", service).Output()
	if err == nil && strings.TrimSpace(string(out)) != "[]" && strings.TrimSpace(string(out)) != "" {
		return "pm2"
	}

	// Check for PID file (raw process)
	pidPaths := []string{
		filepath.Join(installDir, service+".pid"),
		filepath.Join(installDir, "pids", service+".pid"),
	}
	for _, p := range pidPaths {
		if _, err := os.Stat(p); err == nil {
			return "pid-file"
		}
	}

	if findPIDByName(service) > 0 {
		return "process"
	}

	return ""
}

func restartService(service string) error {
	result := restartServiceWithStatus(service, true, 30)
	if result["status"] == "error" {
		if e, ok := result["error"].(string); ok {
			return fmt.Errorf("%s", e)
		}
	}
	return nil
}

func restartServiceWithStatus(service string, wait bool, timeout int) map[string]any {
	result := map[string]any{"service": service}

	// Stop
	stopResult := stopService(service)
	if stopResult["status"] == "error" {
		// Not fatal — might not be running
	}

	// Small delay
	time.Sleep(500 * time.Millisecond)

	// Start
	startResult := startService(service)
	if startResult["status"] == "error" {
		result["status"] = "error"
		result["error"] = startResult["error"]
		return result
	}

	result["pid"] = startResult["pid"]

	if wait {
		if waitForHealthy(service, timeout) {
			result["status"] = "healthy"
		} else {
			result["status"] = "restarted"
		}
	} else {
		result["status"] = "restarted"
	}

	return result
}

func stopService(service string) map[string]any {
	result := map[string]any{"service": service}

	// Try systemd first
	if err := exec.Command("systemctl", "stop", "reposwarm-"+service).Run(); err == nil {
		result["status"] = "stopped"
		return result
	}

	// Try docker
	if service == "temporal" {
		cfg, _ := config.Load()
		installDir := ""
		if cfg != nil {
			installDir = cfg.EffectiveInstallDir()
		}
		composeFile := filepath.Join(installDir, "temporal", "docker-compose.yml")
		if _, err := os.Stat(composeFile); err == nil {
			exec.Command("docker", "compose", "-f", composeFile, "down").Run()
			result["status"] = "stopped"
			return result
		}
	}

	// Try PID
	pid := findServicePID("", service)
	if pid == 0 {
		pid = findPIDByName(service)
	}

	if pid > 0 {
		process, err := os.FindProcess(pid)
		if err == nil {
			process.Signal(syscall.SIGTERM)
			// Wait briefly for graceful shutdown
			time.Sleep(2 * time.Second)
			if isProcessRunning(pid) {
				process.Signal(syscall.SIGKILL)
			}
		}
		result["status"] = "stopped"
		result["pid"] = pid
		return result
	}

	result["status"] = "not_found"
	return result
}

func startService(service string) map[string]any {
	result := map[string]any{"service": service}

	// Try systemd first
	if err := exec.Command("systemctl", "start", "reposwarm-"+service).Run(); err == nil {
		time.Sleep(time.Second)
		pid := findPIDByName(service)
		result["status"] = "started"
		result["pid"] = pid
		return result
	}

	// Try docker for temporal
	if service == "temporal" {
		cfg, _ := config.Load()
		installDir := ""
		if cfg != nil {
			installDir = cfg.EffectiveInstallDir()
		}
		composeFile := filepath.Join(installDir, "temporal", "docker-compose.yml")
		if _, err := os.Stat(composeFile); err == nil {
			if err := exec.Command("docker", "compose", "-f", composeFile, "up", "-d").Run(); err == nil {
				result["status"] = "started"
				result["pid"] = 0
				return result
			}
		}
	}

	// Try starting from install dir
	cfg, _ := config.Load()
	installDir := ""
	if cfg != nil {
		installDir = cfg.EffectiveInstallDir()
	}

	switch service {
	case "api":
		apiDir := filepath.Join(installDir, "api")
		cmd := exec.Command("node", "dist/index.js")
		cmd.Dir = apiDir
		cmd.Env = append(os.Environ(), readEnvAsSlice(filepath.Join(apiDir, ".env"))...)
		if err := cmd.Start(); err == nil {
			result["status"] = "started"
			result["pid"] = cmd.Process.Pid
			return result
		} else {
			result["status"] = "error"
			result["error"] = err.Error()
			return result
		}

	case "worker":
		workerDir := filepath.Join(installDir, "worker")
		cmd := exec.Command("python3", "-m", "src.worker")
		cmd.Dir = workerDir
		cmd.Env = append(os.Environ(), readEnvAsSlice(filepath.Join(workerDir, ".env"))...)
		if err := cmd.Start(); err == nil {
			result["status"] = "started"
			result["pid"] = cmd.Process.Pid
			return result
		} else {
			result["status"] = "error"
			result["error"] = err.Error()
			return result
		}
	}

	result["status"] = "error"
	result["error"] = fmt.Sprintf("don't know how to start %s — no systemd unit or docker compose found", service)
	return result
}

func readEnvAsSlice(path string) []string {
	vars := readOrderedEnv(path)
	var result []string
	for k, v := range vars {
		result = append(result, fmt.Sprintf("%s=%s", k, v))
	}
	return result
}

func waitForHealthy(service string, timeoutSec int) bool {
	deadline := time.Now().Add(time.Duration(timeoutSec) * time.Second)
	for time.Now().Before(deadline) {
		switch service {
		case "api":
			client, err := getClient()
			if err == nil {
				if _, err := client.Health(ctx()); err == nil {
					return true
				}
			}
		case "worker":
			client, err := getClient()
			if err == nil {
				health, err := client.Health(ctx())
				if err == nil && health.Worker.Connected {
					return true
				}
			}
		case "temporal":
			// Check UI port
			cfg, _ := config.Load()
			port := "8233"
			if cfg != nil {
				port = cfg.EffectiveTemporalUIPort()
			}
			resp, err := exec.Command("curl", "-s", "-o", "/dev/null", "-w", "%{http_code}",
				fmt.Sprintf("http://localhost:%s/api/v1/namespaces", port)).Output()
			if err == nil && strings.TrimSpace(string(resp)) == "200" {
				return true
			}
		default:
			// Generic: check if process is running
			pid := findPIDByName(service)
			if pid > 0 {
				return true
			}
		}
		time.Sleep(2 * time.Second)
	}
	return false
}

func isKnownService(name string) bool {
	for _, s := range knownServices {
		if s == name {
			return true
		}
	}
	// Also accept worker-N
	if strings.HasPrefix(name, "worker-") {
		return true
	}
	return false
}
