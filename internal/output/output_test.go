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
