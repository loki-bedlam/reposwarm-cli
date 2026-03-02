package bootstrap

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// LocalSetupResult holds the outcome of each setup step.
type LocalSetupResult struct {
	InstallDir string            `json:"installDir"`
	Token      string            `json:"token"`
	Steps      []LocalStepResult `json:"steps"`
	Success    bool              `json:"success"`
}

// LocalStepResult is one step in the setup process.
type LocalStepResult struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // ok, fail, skip
	Message string `json:"message,omitempty"`
}

// Printer abstracts formatted output so SetupLocal doesn't depend on the output package.
type Printer interface {
	Section(title string)
	Info(msg string)
	Success(msg string)
	Warning(msg string)
	Error(msg string)
	Printf(format string, args ...any)
}

// SetupLocal orchestrates a complete local RepoSwarm environment.
func SetupLocal(env *Environment, installDir string, printer Printer) (*LocalSetupResult, error) {
	result := &LocalSetupResult{InstallDir: installDir}

	// Step 0: Check prerequisites
	printer.Section("Checking prerequisites")
	if missing := env.MissingDeps(); len(missing) > 0 {
		for _, dep := range missing {
			printer.Error(fmt.Sprintf("Missing: %s", dep))
		}
		result.Steps = append(result.Steps, LocalStepResult{"prerequisites", "fail", "missing: " + strings.Join(missing, ", ")})
		return result, fmt.Errorf("missing prerequisites: %s — install them first", strings.Join(missing, ", "))
	}
	printer.Success("All prerequisites found")
	result.Steps = append(result.Steps, LocalStepResult{"prerequisites", "ok", ""})

	// Generate a bearer token for local auth
	token, err := randomHex(32)
	if err != nil {
		return result, fmt.Errorf("generating token: %w", err)
	}
	result.Token = token

	// Step 1: Create directory structure
	printer.Section("Creating directory structure")
	if err := os.MkdirAll(installDir, 0755); err != nil {
		result.Steps = append(result.Steps, LocalStepResult{"directories", "fail", err.Error()})
		return result, fmt.Errorf("creating install directory: %w", err)
	}
	printer.Success(fmt.Sprintf("Install directory: %s", installDir))
	result.Steps = append(result.Steps, LocalStepResult{"directories", "ok", installDir})

	// Step 2: Start Temporal
	printer.Section("Starting Temporal (Docker Compose)")
	if err := setupTemporal(installDir, printer); err != nil {
		result.Steps = append(result.Steps, LocalStepResult{"temporal", "fail", err.Error()})
		return result, fmt.Errorf("temporal setup: %w", err)
	}
	result.Steps = append(result.Steps, LocalStepResult{"temporal", "ok", "http://localhost:8233"})

	// Step 3: Clone and start API
	printer.Section("Setting up API server")
	if err := setupAPI(installDir, env.AWSRegion, token, printer); err != nil {
		result.Steps = append(result.Steps, LocalStepResult{"api", "fail", err.Error()})
		return result, fmt.Errorf("API setup: %w", err)
	}
	result.Steps = append(result.Steps, LocalStepResult{"api", "ok", "http://localhost:3000"})

	// Step 4: Clone and start Worker
	printer.Section("Setting up Worker")
	if err := setupWorker(installDir, env.AWSRegion, printer); err != nil {
		printer.Warning(fmt.Sprintf("Worker setup failed: %s (investigations won't run, but API/UI will work)", err))
		result.Steps = append(result.Steps, LocalStepResult{"worker", "fail", err.Error()})
		// Don't return error — worker is optional for basic functionality
	} else {
		result.Steps = append(result.Steps, LocalStepResult{"worker", "ok", ""})
	}

	// Step 5: Clone and start UI
	printer.Section("Setting up UI")
	if err := setupUI(installDir, printer); err != nil {
		printer.Warning(fmt.Sprintf("UI setup failed: %s (CLI still works)", err))
		result.Steps = append(result.Steps, LocalStepResult{"ui", "fail", err.Error()})
	} else {
		result.Steps = append(result.Steps, LocalStepResult{"ui", "ok", "http://localhost:3001"})
	}

	// Step 6: Configure CLI
	printer.Section("Configuring CLI")
	if err := configureCLI(token); err != nil {
		result.Steps = append(result.Steps, LocalStepResult{"cli-config", "fail", err.Error()})
		return result, fmt.Errorf("CLI configuration: %w", err)
	}
	printer.Success("CLI configured for local API")
	result.Steps = append(result.Steps, LocalStepResult{"cli-config", "ok", ""})

	// Step 7: Verify
	printer.Section("Verifying services")
	verifyResult := verifyServices(printer)
	result.Steps = append(result.Steps, verifyResult)

	result.Success = verifyResult.Status != "fail"

	// Print summary
	printer.Section("Setup Complete")
	if result.Success {
		printer.Success("RepoSwarm local environment is running!")
	} else {
		printer.Warning("RepoSwarm started with some issues (see above)")
	}
	printer.Printf("")
	printer.Printf("  Temporal UI:  http://localhost:8233")
	printer.Printf("  API Server:   http://localhost:3000")
	printer.Printf("  UI:           http://localhost:3001")
	printer.Printf("")
	printer.Printf("  API Token:    %s", token)
	printer.Printf("  Logs:         %s/*/%.log", installDir)
	printer.Printf("")
	printer.Printf("  Try:")
	printer.Printf("    reposwarm status")
	printer.Printf("    reposwarm repos add is-odd --url https://github.com/jonschlinkert/is-odd --source GitHub")
	printer.Printf("    reposwarm investigate is-odd")
	printer.Printf("")

	return result, nil
}

func setupTemporal(installDir string, printer Printer) error {
	temporalDir := filepath.Join(installDir, "temporal")
	if err := os.MkdirAll(temporalDir, 0755); err != nil {
		return err
	}

	composePath := filepath.Join(temporalDir, "docker-compose.yml")
	if err := os.WriteFile(composePath, []byte(TemporalComposeLocal()), 0644); err != nil {
		return fmt.Errorf("writing docker-compose.yml: %w", err)
	}
	printer.Info("Wrote docker-compose.yml")

	// docker compose up -d
	cmd := exec.Command("docker", "compose", "up", "-d")
	cmd.Dir = temporalDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("docker compose up failed: %w\n%s", err, string(out))
	}
	printer.Info("Docker containers starting...")

	// Wait for Temporal to be ready (up to 60s)
	printer.Info("Waiting for Temporal to be ready (this may take up to 60s)...")
	if err := waitForHTTP("http://localhost:7233/api/v1/namespaces", 60*time.Second); err != nil {
		// Check container status for debugging
		statusCmd := exec.Command("docker", "compose", "ps", "--format", "{{.Name}}\t{{.Status}}")
		statusCmd.Dir = temporalDir
		statusOut, _ := statusCmd.CombinedOutput()
		return fmt.Errorf("temporal not ready after 60s: %w\nContainer status:\n%s", err, string(statusOut))
	}
	printer.Success("Temporal is ready")
	return nil
}

func setupAPI(installDir, region, token string, printer Printer) error {
	apiDir := filepath.Join(installDir, "api")

	// Clone
	if _, err := os.Stat(apiDir); os.IsNotExist(err) {
		printer.Info("Cloning API server...")
		cmd := exec.Command("git", "clone", "https://github.com/loki-bedlam/reposwarm-api.git", "api")
		cmd.Dir = installDir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git clone failed: %w\n%s", err, string(out))
		}
	} else {
		printer.Info("API directory exists, skipping clone")
	}

	// npm install
	printer.Info("Installing dependencies...")
	npmInstall := exec.Command("npm", "install")
	npmInstall.Dir = apiDir
	if out, err := npmInstall.CombinedOutput(); err != nil {
		return fmt.Errorf("npm install failed: %w\n%s", err, string(out))
	}

	// npm run build
	printer.Info("Building...")
	npmBuild := exec.Command("npm", "run", "build")
	npmBuild.Dir = apiDir
	if out, err := npmBuild.CombinedOutput(); err != nil {
		return fmt.Errorf("npm build failed: %w\n%s", err, string(out))
	}

	// Write .env
	envContent := fmt.Sprintf(`PORT=3000
TEMPORAL_ADDRESS=localhost:7233
TEMPORAL_NAMESPACE=default
TEMPORAL_TASK_QUEUE=investigate-task-queue
AWS_REGION=%s
DYNAMODB_TABLE=reposwarm-cache
BEARER_TOKEN=%s
AUTH_MODE=local
`, region, token)

	if err := os.WriteFile(filepath.Join(apiDir, ".env"), []byte(envContent), 0600); err != nil {
		return fmt.Errorf("writing .env: %w", err)
	}

	// Start API in background
	printer.Info("Starting API server...")
	logFile, err := os.Create(filepath.Join(apiDir, "api.log"))
	if err != nil {
		return fmt.Errorf("creating log file: %w", err)
	}

	startCmd := exec.Command("npm", "start")
	startCmd.Dir = apiDir
	startCmd.Stdout = logFile
	startCmd.Stderr = logFile
	if err := startCmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("starting API: %w", err)
	}
	logFile.Close()

	// Write PID file for later management
	pidFile := filepath.Join(apiDir, "api.pid")
	os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", startCmd.Process.Pid)), 0644)

	// Wait for API
	printer.Info("Waiting for API to be ready...")
	if err := waitForHTTP("http://localhost:3000/v1/health", 30*time.Second); err != nil {
		return fmt.Errorf("API not ready after 30s: %w", err)
	}
	printer.Success("API server is ready")
	return nil
}

func setupWorker(installDir, region string, printer Printer) error {
	workerDir := filepath.Join(installDir, "worker")

	// Clone
	if _, err := os.Stat(workerDir); os.IsNotExist(err) {
		printer.Info("Cloning worker...")
		cmd := exec.Command("git", "clone", "https://github.com/royosherove/repo-swarm.git", "worker")
		cmd.Dir = installDir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git clone failed: %w\n%s", err, string(out))
		}
	} else {
		printer.Info("Worker directory exists, skipping clone")
	}

	// Create venv
	printer.Info("Creating Python virtual environment...")
	venvCmd := exec.Command("python3", "-m", "venv", ".venv")
	venvCmd.Dir = workerDir
	if out, err := venvCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("venv creation failed: %w\n%s", err, string(out))
	}

	// pip install
	printer.Info("Installing Python dependencies...")
	pipPath := filepath.Join(workerDir, ".venv", "bin", "pip")
	pipCmd := exec.Command(pipPath, "install", "-r", "requirements.txt")
	pipCmd.Dir = workerDir
	if out, err := pipCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("pip install failed: %w\n%s", err, string(out))
	}

	// Write .env
	envContent := fmt.Sprintf(`TEMPORAL_ADDRESS=localhost:7233
TEMPORAL_NAMESPACE=default
TEMPORAL_TASK_QUEUE=investigate-task-queue
AWS_REGION=%s
DYNAMODB_TABLE=reposwarm-cache
DEFAULT_MODEL=us.anthropic.claude-sonnet-4-6
`, region)

	if err := os.WriteFile(filepath.Join(workerDir, ".env"), []byte(envContent), 0600); err != nil {
		return fmt.Errorf("writing .env: %w", err)
	}

	// Start worker in background
	printer.Info("Starting worker...")
	logFile, err := os.Create(filepath.Join(workerDir, "worker.log"))
	if err != nil {
		return fmt.Errorf("creating log file: %w", err)
	}

	pythonPath := filepath.Join(workerDir, ".venv", "bin", "python")
	startCmd := exec.Command(pythonPath, "-m", "worker.main")
	startCmd.Dir = workerDir
	startCmd.Stdout = logFile
	startCmd.Stderr = logFile
	// Pass env vars explicitly since .env isn't auto-loaded
	startCmd.Env = append(os.Environ(),
		"TEMPORAL_ADDRESS=localhost:7233",
		"TEMPORAL_NAMESPACE=default",
		"TEMPORAL_TASK_QUEUE=investigate-task-queue",
		fmt.Sprintf("AWS_REGION=%s", region),
		"DYNAMODB_TABLE=reposwarm-cache",
		"DEFAULT_MODEL=us.anthropic.claude-sonnet-4-6",
	)
	if err := startCmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("starting worker: %w", err)
	}
	logFile.Close()

	pidFile := filepath.Join(workerDir, "worker.pid")
	os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", startCmd.Process.Pid)), 0644)

	printer.Success("Worker started")
	return nil
}

func setupUI(installDir string, printer Printer) error {
	uiDir := filepath.Join(installDir, "ui")

	// Clone
	if _, err := os.Stat(uiDir); os.IsNotExist(err) {
		printer.Info("Cloning UI...")
		cmd := exec.Command("git", "clone", "https://github.com/loki-bedlam/reposwarm-ui.git", "ui")
		cmd.Dir = installDir
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git clone failed: %w\n%s", err, string(out))
		}
	} else {
		printer.Info("UI directory exists, skipping clone")
	}

	// npm install
	printer.Info("Installing dependencies...")
	npmInstall := exec.Command("npm", "install")
	npmInstall.Dir = uiDir
	if out, err := npmInstall.CombinedOutput(); err != nil {
		return fmt.Errorf("npm install failed: %w\n%s", err, string(out))
	}

	// Write .env.local
	envContent := "NEXT_PUBLIC_API_URL=http://localhost:3000\n"
	if err := os.WriteFile(filepath.Join(uiDir, ".env.local"), []byte(envContent), 0644); err != nil {
		return fmt.Errorf("writing .env.local: %w", err)
	}

	// Start UI in background
	printer.Info("Starting UI dev server...")
	logFile, err := os.Create(filepath.Join(uiDir, "ui.log"))
	if err != nil {
		return fmt.Errorf("creating log file: %w", err)
	}

	startCmd := exec.Command("npm", "run", "dev")
	startCmd.Dir = uiDir
	startCmd.Stdout = logFile
	startCmd.Stderr = logFile
	if err := startCmd.Start(); err != nil {
		logFile.Close()
		return fmt.Errorf("starting UI: %w", err)
	}
	logFile.Close()

	pidFile := filepath.Join(uiDir, "ui.pid")
	os.WriteFile(pidFile, []byte(fmt.Sprintf("%d", startCmd.Process.Pid)), 0644)

	// Wait for UI
	printer.Info("Waiting for UI to be ready...")
	if err := waitForHTTP("http://localhost:3001", 30*time.Second); err != nil {
		printer.Warning("UI not responding yet — it may still be compiling (check ui/ui.log)")
		return nil // Non-fatal
	}
	printer.Success("UI is ready")
	return nil
}

func configureCLI(token string) error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	configDir := filepath.Join(home, ".reposwarm")
	if err := os.MkdirAll(configDir, 0700); err != nil {
		return err
	}
	configContent := fmt.Sprintf(`{
  "apiUrl": "http://localhost:3000/v1",
  "apiToken": "%s",
  "region": "us-east-1",
  "defaultModel": "us.anthropic.claude-sonnet-4-6",
  "chunkSize": 10,
  "outputFormat": "pretty"
}
`, token)
	return os.WriteFile(filepath.Join(configDir, "config.json"), []byte(configContent), 0600)
}

func verifyServices(printer Printer) LocalStepResult {
	checks := []struct {
		name string
		url  string
	}{
		{"Temporal", "http://localhost:7233/api/v1/namespaces"},
		{"API", "http://localhost:3000/v1/health"},
		{"UI", "http://localhost:3001"},
	}

	allOK := true
	var messages []string
	for _, c := range checks {
		resp, err := http.Get(c.url)
		if err != nil {
			printer.Warning(fmt.Sprintf("%s: not responding (%s)", c.name, err))
			messages = append(messages, fmt.Sprintf("%s: fail", c.name))
			allOK = false
			continue
		}
		resp.Body.Close()
		if resp.StatusCode >= 200 && resp.StatusCode < 400 {
			printer.Success(fmt.Sprintf("%s: healthy", c.name))
			messages = append(messages, fmt.Sprintf("%s: ok", c.name))
		} else {
			printer.Warning(fmt.Sprintf("%s: HTTP %d", c.name, resp.StatusCode))
			messages = append(messages, fmt.Sprintf("%s: HTTP %d", c.name, resp.StatusCode))
			allOK = false
		}
	}

	status := "ok"
	if !allOK {
		status = "fail"
	}
	return LocalStepResult{"verify", status, strings.Join(messages, "; ")}
}

// TemporalComposeLocal returns the docker-compose YAML for local development.
// Uses postgres instead of the deprecated sqlite driver.
func TemporalComposeLocal() string {
	return `services:
  postgres:
    image: postgres:16-alpine
    ports:
      - "5432:5432"
    environment:
      POSTGRES_USER: temporal
      POSTGRES_PASSWORD: temporal
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U temporal"]
      interval: 5s
      timeout: 5s
      retries: 10
    volumes:
      - temporal-data:/var/lib/postgresql/data

  temporal:
    image: temporalio/auto-setup:latest
    ports:
      - "7233:7233"
    environment:
      - DB=postgres12
      - POSTGRES_USER=temporal
      - POSTGRES_PWD=temporal
      - POSTGRES_SEEDS=postgres
      - DYNAMIC_CONFIG_FILE_PATH=config/dynamicconfig/development-sql.yaml
      - SKIP_DEFAULT_NAMESPACE_CREATION=false
    depends_on:
      postgres:
        condition: service_healthy

  temporal-ui:
    image: temporalio/ui:latest
    ports:
      - "8233:8080"
    environment:
      - TEMPORAL_ADDRESS=temporal:7233
    depends_on:
      - temporal

volumes:
  temporal-data:
`
}

func waitForHTTP(url string, timeout time.Duration) error {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	client := &http.Client{Timeout: 5 * time.Second}
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for %s", url)
		case <-ticker.C:
			resp, err := client.Get(url)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode < 500 {
					return nil
				}
			}
		}
	}
}

func randomHex(n int) (string, error) {
	b := make([]byte, n)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}
