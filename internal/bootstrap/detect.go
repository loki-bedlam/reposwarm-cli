// Package bootstrap handles local RepoSwarm installation setup.
package bootstrap

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

// Environment holds detected local environment info.
type Environment struct {
	OS           string `json:"os"`
	Arch         string `json:"arch"`
	HomeDir      string `json:"homeDir"`
	WorkDir      string `json:"workDir"`
	Shell        string `json:"shell"`

	// Runtimes
	HasDocker    bool   `json:"hasDocker"`
	DockerVer    string `json:"dockerVersion,omitempty"`
	HasCompose   bool   `json:"hasDockerCompose"`
	ComposeVer   string `json:"composeVersion,omitempty"`
	HasNode      bool   `json:"hasNode"`
	NodeVer      string `json:"nodeVersion,omitempty"`
	HasPython    bool   `json:"hasPython"`
	PythonVer    string `json:"pythonVersion,omitempty"`
	HasGo        bool   `json:"hasGo"`
	GoVer        string `json:"goVersion,omitempty"`
	HasGit       bool   `json:"hasGit"`
	GitVer       string `json:"gitVersion,omitempty"`

	// Coding agents
	HasClaudeCode bool  `json:"hasClaudeCode"`
	HasCursor     bool  `json:"hasCursor"`
	HasCodex      bool  `json:"hasCodex"`
	HasAider      bool  `json:"hasAider"`

	// AWS
	HasAWSCLI    bool   `json:"hasAwsCli"`
	AWSRegion    string `json:"awsRegion,omitempty"`
	AWSProfile   string `json:"awsProfile,omitempty"`

	// Package managers
	HasBrew      bool   `json:"hasBrew"`
	HasApt       bool   `json:"hasApt"`
	HasPip       bool   `json:"hasPip"`
	HasNpm       bool   `json:"hasNpm"`
}

// Detect scans the local environment.
func Detect() *Environment {
	env := &Environment{
		OS:   runtime.GOOS,
		Arch: runtime.GOARCH,
	}

	env.HomeDir, _ = os.UserHomeDir()
	env.WorkDir, _ = os.Getwd()
	env.Shell = os.Getenv("SHELL")

	// Runtimes
	env.DockerVer, env.HasDocker = cmdVersion("docker", "--version")
	env.ComposeVer, env.HasCompose = cmdVersion("docker", "compose", "version")
	env.NodeVer, env.HasNode = cmdVersion("node", "--version")
	env.PythonVer, env.HasPython = cmdVersionAny([][]string{{"python3", "--version"}, {"python", "--version"}})
	env.GoVer, env.HasGo = cmdVersion("go", "version")
	env.GitVer, env.HasGit = cmdVersion("git", "--version")

	// Coding agents
	env.HasClaudeCode = cmdExists("claude")
	env.HasCursor = cmdExists("cursor")
	env.HasCodex = cmdExists("codex")
	env.HasAider = cmdExists("aider")

	// AWS
	_, env.HasAWSCLI = cmdVersion("aws", "--version")
	env.AWSRegion = firstNonEmpty(os.Getenv("AWS_REGION"), os.Getenv("AWS_DEFAULT_REGION"), "us-east-1")
	env.AWSProfile = os.Getenv("AWS_PROFILE")

	// Package managers
	env.HasBrew = cmdExists("brew")
	env.HasApt = cmdExists("apt-get")
	env.HasPip = cmdExists("pip3") || cmdExists("pip")
	env.HasNpm = cmdExists("npm")

	return env
}

// AgentName returns the best available coding agent name, or "".
func (e *Environment) AgentName() string {
	if e.HasClaudeCode {
		return "claude"
	}
	if e.HasCodex {
		return "codex"
	}
	if e.HasCursor {
		return "cursor"
	}
	if e.HasAider {
		return "aider"
	}
	return ""
}

// MissingDeps returns a list of missing required dependencies.
func (e *Environment) MissingDeps() []string {
	var missing []string
	if !e.HasDocker {
		missing = append(missing, "docker")
	}
	if !e.HasCompose {
		missing = append(missing, "docker-compose")
	}
	if !e.HasNode {
		missing = append(missing, "node (v22+)")
	}
	if !e.HasPython {
		missing = append(missing, "python3 (3.11+)")
	}
	if !e.HasGit {
		missing = append(missing, "git")
	}
	return missing
}

// InstallDir returns the target installation directory.
func (e *Environment) InstallDir() string {
	return filepath.Join(e.WorkDir, "reposwarm")
}

func cmdExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

func cmdVersion(args ...string) (string, bool) {
	cmd := exec.Command(args[0], args[1:]...)
	out, err := cmd.Output()
	if err != nil {
		return "", false
	}
	return strings.TrimSpace(string(out)), true
}

func cmdVersionAny(cmds [][]string) (string, bool) {
	for _, args := range cmds {
		if v, ok := cmdVersion(args...); ok {
			return v, true
		}
	}
	return "", false
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if v != "" {
			return v
		}
	}
	return ""
}

// Summary returns a human-readable environment summary.
func (e *Environment) Summary() string {
	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("  System:   %s/%s\n", e.OS, e.Arch))
	sb.WriteString(fmt.Sprintf("  Shell:    %s\n", e.Shell))
	sb.WriteString(fmt.Sprintf("  Work dir: %s\n", e.WorkDir))
	sb.WriteString("\n  Runtimes:\n")
	rt := func(name string, has bool, ver string) {
		if has {
			sb.WriteString(fmt.Sprintf("    ✅ %s — %s\n", name, ver))
		} else {
			sb.WriteString(fmt.Sprintf("    ❌ %s — not found\n", name))
		}
	}
	rt("Docker", e.HasDocker, e.DockerVer)
	rt("Docker Compose", e.HasCompose, e.ComposeVer)
	rt("Node.js", e.HasNode, e.NodeVer)
	rt("Python", e.HasPython, e.PythonVer)
	rt("Go", e.HasGo, e.GoVer)
	rt("Git", e.HasGit, e.GitVer)
	rt("AWS CLI", e.HasAWSCLI, "")

	sb.WriteString("\n  Coding Agents:\n")
	agents := []struct{ name string; has bool }{
		{"Claude Code", e.HasClaudeCode},
		{"Codex", e.HasCodex},
		{"Cursor", e.HasCursor},
		{"Aider", e.HasAider},
	}
	anyAgent := false
	for _, a := range agents {
		if a.has {
			sb.WriteString(fmt.Sprintf("    ✅ %s\n", a.name))
			anyAgent = true
		}
	}
	if !anyAgent {
		sb.WriteString("    (none detected)\n")
	}

	return sb.String()
}
