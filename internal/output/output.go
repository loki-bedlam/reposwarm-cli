// Package output handles CLI output formatting (pretty tables, JSON, raw).
package output

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/fatih/color"
)

var (
	Bold    = color.New(color.Bold).SprintFunc()
	Green   = color.New(color.FgGreen).SprintFunc()
	Red     = color.New(color.FgRed).SprintFunc()
	Yellow  = color.New(color.FgYellow).SprintFunc()
	Cyan    = color.New(color.FgCyan).SprintFunc()
	Dim     = color.New(color.Faint).SprintFunc()
	Success = color.New(color.FgGreen, color.Bold).SprintFunc()
	Error   = color.New(color.FgRed, color.Bold).SprintFunc()
)

// JSON prints data as indented JSON to stdout.
func JSON(data any) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(data)
}

// Table prints a simple table with headers and rows.
func Table(headers []string, rows [][]string) {
	if len(rows) == 0 {
		fmt.Println(Dim("  (no results)"))
		return
	}

	// Calculate column widths
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

	// Print header
	var hdr []string
	for i, h := range headers {
		hdr = append(hdr, Bold(pad(h, widths[i])))
	}
	fmt.Println("  " + strings.Join(hdr, "  "))
	var sep []string
	for _, w := range widths {
		sep = append(sep, strings.Repeat("─", w))
	}
	fmt.Println("  " + Dim(strings.Join(sep, "──")))

	// Print rows
	for _, row := range rows {
		var cells []string
		for i, cell := range row {
			if i < len(widths) {
				cells = append(cells, pad(cell, widths[i]))
			}
		}
		fmt.Println("  " + strings.Join(cells, "  "))
	}
}

func pad(s string, width int) string {
	if len(s) >= width {
		return s
	}
	return s + strings.Repeat(" ", width-len(s))
}

// StatusColor returns a colored status string.
func StatusColor(status string) string {
	lower := strings.ToLower(status)
	switch {
	case lower == "running":
		return Yellow(status)
	case lower == "completed":
		return Green(status)
	case lower == "failed":
		return Red(status)
	case lower == "terminated" || lower == "cancelled":
		return Dim(status)
	default:
		return status
	}
}

// Successf prints a success message.
func Successf(format string, args ...any) {
	fmt.Printf("  %s %s\n", Green("✓"), fmt.Sprintf(format, args...))
}

// Errorf prints an error message to stderr.
func Errorf(format string, args ...any) {
	fmt.Fprintf(os.Stderr, "  %s %s\n", Red("✗"), fmt.Sprintf(format, args...))
}

// Infof prints an info message.
func Infof(format string, args ...any) {
	fmt.Printf("  %s %s\n", Cyan("ℹ"), fmt.Sprintf(format, args...))
}
