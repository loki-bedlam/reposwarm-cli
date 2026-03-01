package output

import (
	"bytes"
	"os"
	"strings"
	"testing"
)

func TestJSON(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	data := map[string]string{"key": "value"}
	JSON(data)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), `"key": "value"`) {
		t.Errorf("JSON output = %s, want key:value", buf.String())
	}
}

func TestStatusColor(t *testing.T) {
	tests := []struct {
		input string
	}{
		{"Running"},
		{"Completed"},
		{"Failed"},
		{"Terminated"},
		{"Unknown"},
	}
	for _, tt := range tests {
		got := StatusColor(tt.input)
		if got == "" {
			t.Errorf("StatusColor(%s) returned empty", tt.input)
		}
	}
}

func TestTable(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	Table([]string{"Name", "Age"}, [][]string{{"Alice", "30"}, {"Bob", "25"}})

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	out := buf.String()
	if !strings.Contains(out, "Alice") {
		t.Error("table should contain Alice")
	}
	if !strings.Contains(out, "Bob") {
		t.Error("table should contain Bob")
	}
}

func TestTableEmpty(t *testing.T) {
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	Table([]string{"Name"}, nil)

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	buf.ReadFrom(r)
	if !strings.Contains(buf.String(), "no results") {
		t.Error("empty table should show 'no results'")
	}
}

func TestHumanFormatterFinishShowsAgentHint(t *testing.T) {
	var buf bytes.Buffer
	f := &HumanFormatter{w: &buf}
	f.Finish()
	out := buf.String()
	if !strings.Contains(out, "--for-agent") {
		t.Errorf("HumanFormatter.Finish() should mention --for-agent, got: %s", out)
	}
	if !strings.Contains(out, "agent and not a human") {
		t.Errorf("HumanFormatter.Finish() should contain hint text, got: %s", out)
	}
}

func TestAgentFormatterFinishIsEmpty(t *testing.T) {
	var buf bytes.Buffer
	f := &AgentFormatter{w: &buf}
	f.Finish()
	if buf.Len() != 0 {
		t.Errorf("AgentFormatter.Finish() should produce no output, got: %s", buf.String())
	}
}

func TestForAgentFlagSuppressesHint(t *testing.T) {
	// When InitFormatter(false) is called (agent mode), Finish should be no-op
	InitFormatter(false)
	var buf bytes.Buffer
	// Swap the global formatter's writer
	agent := F.(*AgentFormatter)
	agent.w = &buf
	F.Finish()
	if buf.Len() != 0 {
		t.Errorf("Agent mode Finish() should be silent, got: %s", buf.String())
	}

	// Restore
	InitFormatter(true)
}
