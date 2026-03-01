package output

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/fatih/color"
)

var (
	// F is the global formatter, set during initialization.
	F Formatter = &AgentFormatter{w: os.Stdout}
	// IsHuman indicates whether rich output mode is active.
	IsHuman bool
)

// Formatter provides structured output methods for CLI commands.
type Formatter interface {
	Table(headers []string, rows [][]string)
	Section(title string)
	KeyValue(key, value string)
	Success(msg string)
	Error(msg string)
	Finish()
	Info(msg string)
	Warning(msg string)
	List(items []string)
	Progress(completed, total int)
	CheckResult(name, status, message string)
	CheckSummary(ok, warn, fail int)
	StatusText(status string) string
	SectionIcon(id string) string
	Println(a ...any)
	Printf(format string, a ...any)
}

// InitFormatter sets up the global formatter based on mode.
func InitFormatter(human bool) {
	IsHuman = human
	if human {
		F = &HumanFormatter{w: os.Stdout}
		Bold = color.New(color.Bold).SprintFunc()
		Green = color.New(color.FgGreen).SprintFunc()
		Red = color.New(color.FgRed).SprintFunc()
		Yellow = color.New(color.FgYellow).SprintFunc()
		Cyan = color.New(color.FgCyan).SprintFunc()
		Dim = color.New(color.Faint).SprintFunc()
		Success = color.New(color.FgGreen, color.Bold).SprintFunc()
		Error = color.New(color.FgRed, color.Bold).SprintFunc()
	} else {
		F = &AgentFormatter{w: os.Stdout}
		Bold = fmt.Sprint
		Green = fmt.Sprint
		Red = fmt.Sprint
		Yellow = fmt.Sprint
		Cyan = fmt.Sprint
		Dim = fmt.Sprint
		Success = fmt.Sprint
		Error = fmt.Sprint
	}
}

// ---------------------------------------------------------------------------
// AgentFormatter — plain text, markdown-compatible, no emojis/colors
// ---------------------------------------------------------------------------

type AgentFormatter struct {
	w io.Writer
}

func (f *AgentFormatter) Table(headers []string, rows [][]string) {
	if len(rows) == 0 {
		fmt.Fprintln(f.w, "(no results)")
		return
	}
	widths := make([]int, len(headers))
	for i, h := range headers {
		widths[i] = len(h)
	}
	for _, row := range rows {
		for i, cell := range row {
			if i < len(widths) && len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}
	var hdr []string
	for i, h := range headers {
		hdr = append(hdr, pad(h, widths[i]))
	}
	fmt.Fprintln(f.w, strings.Join(hdr, "  "))
	var sep []string
	for _, w := range widths {
		sep = append(sep, strings.Repeat("-", w))
	}
	fmt.Fprintln(f.w, strings.Join(sep, "+-"))
	for _, row := range rows {
		var cells []string
		for i, cell := range row {
			if i < len(widths) {
				cells = append(cells, pad(cell, widths[i]))
			}
		}
		fmt.Fprintln(f.w, strings.Join(cells, "  "))
	}
}

func (f *AgentFormatter) Section(title string) {
	fmt.Fprintf(f.w, "\n## %s\n\n", title)
}

func (f *AgentFormatter) KeyValue(key, value string) {
	fmt.Fprintf(f.w, "%-20s %s\n", key+":", value)
}

func (f *AgentFormatter) Success(msg string) {
	fmt.Fprintf(f.w, "OK: %s\n", msg)
}

func (f *AgentFormatter) Error(msg string) {
	fmt.Fprintf(os.Stderr, "ERROR: %s\n", msg)
}

func (f *AgentFormatter) Info(msg string) {
	fmt.Fprintln(f.w, msg)
}

func (f *AgentFormatter) Warning(msg string) {
	fmt.Fprintf(os.Stderr, "WARNING: %s\n", msg)
}

func (f *AgentFormatter) List(items []string) {
	for _, item := range items {
		fmt.Fprintf(f.w, "- %s\n", item)
	}
}

func (f *AgentFormatter) Progress(completed, total int) {
	pct := 0
	if total > 0 {
		pct = completed * 100 / total
	}
	fmt.Fprintf(f.w, "Progress: %d/%d (%d%%)\n", completed, total, pct)
}

func (f *AgentFormatter) CheckResult(name, status, message string) {
	fmt.Fprintf(f.w, "[%s] %s: %s\n", strings.ToUpper(status), name, message)
}

func (f *AgentFormatter) CheckSummary(ok, warn, fail int) {
	if fail == 0 && warn == 0 {
		fmt.Fprintf(f.w, "\nAll %d checks passed\n", ok)
	} else if fail == 0 {
		fmt.Fprintf(f.w, "\n%d passed, %d warnings\n", ok, warn)
	} else {
		fmt.Fprintf(f.w, "\n%d passed, %d warnings, %d failed\n", ok, warn, fail)
	}
}

func (f *AgentFormatter) StatusText(status string) string {
	return status
}

func (f *AgentFormatter) SectionIcon(_ string) string {
	return ""
}

func (f *AgentFormatter) Println(a ...any) {
	fmt.Fprintln(f.w, a...)
}

func (f *AgentFormatter) Printf(format string, a ...any) {
	fmt.Fprintf(f.w, format, a...)
}

// ---------------------------------------------------------------------------
// HumanFormatter — rich output with colors, emojis, progress bars
// ---------------------------------------------------------------------------

type HumanFormatter struct {
	w io.Writer
}

func (f *HumanFormatter) Table(headers []string, rows [][]string) {
	Table(headers, rows)
}

func (f *HumanFormatter) Section(title string) {
	fmt.Printf("\n  %s\n\n", Bold(title))
}

func (f *HumanFormatter) KeyValue(key, value string) {
	fmt.Printf("  %-18s  %s\n", Dim(key), value)
}

func (f *HumanFormatter) Success(msg string) {
	fmt.Printf("  %s %s\n", Green("✓"), msg)
}

func (f *HumanFormatter) Error(msg string) {
	fmt.Fprintf(os.Stderr, "  %s %s\n", Red("✗"), msg)
}

func (f *HumanFormatter) Info(msg string) {
	fmt.Printf("  %s %s\n", Cyan("ℹ"), msg)
}

func (f *HumanFormatter) Warning(msg string) {
	fmt.Printf("  %s %s\n", Yellow("⚠"), msg)
}

func (f *HumanFormatter) List(items []string) {
	for _, item := range items {
		fmt.Printf("  • %s\n", item)
	}
}

func (f *HumanFormatter) Progress(completed, total int) {
	pct := 0
	if total > 0 {
		pct = completed * 100 / total
	}
	barWidth := 30
	filled := 0
	if total > 0 {
		filled = barWidth * completed / total
	}
	bar := strings.Repeat("█", filled) + strings.Repeat("░", barWidth-filled)
	fmt.Printf("  %s %d%% (%d/%d)\n", bar, pct, completed, total)
}

func (f *HumanFormatter) CheckResult(name, status, message string) {
	icon := Green("✓")
	if status == "warn" {
		icon = Yellow("⚠")
	} else if status == "fail" {
		icon = Red("✗")
	}
	fmt.Printf("  %s %s — %s\n", icon, name, message)
}

func (f *HumanFormatter) CheckSummary(ok, warn, fail int) {
	fmt.Println()
	if fail == 0 && warn == 0 {
		fmt.Printf("  %s All %d checks passed\n\n", Green("✅"), ok)
	} else if fail == 0 {
		fmt.Printf("  %s %d passed, %d warnings\n\n", Yellow("⚠️"), ok, warn)
	} else {
		fmt.Printf("  %s %d passed, %d warnings, %d failed\n\n", Red("❌"), ok, warn, fail)
	}
}

func (f *HumanFormatter) StatusText(status string) string {
	return StatusColor(status)
}

func (f *HumanFormatter) SectionIcon(id string) string {
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
		return icon + " "
	}
	return "📄 "
}

func (f *HumanFormatter) Println(a ...any) {
	fmt.Fprintln(f.w, a...)
}

func (f *HumanFormatter) Printf(format string, a ...any) {
	fmt.Fprintf(f.w, format, a...)
}

// Finish prints the agent hint at the end of human-mode output
func (f *HumanFormatter) Finish() {
	fmt.Fprintf(f.w, "\n  %s\n\n", Dim("If you're an agent and not a human, add --for-agent to any command for machine-friendly output."))
}

func (f *AgentFormatter) Finish() {}
