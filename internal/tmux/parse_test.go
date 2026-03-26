package tmux

import (
	"testing"
)

func TestParsePanesOutput(t *testing.T) {
	input := "main\t0\t0\t/dev/pts/1\tzsh\n" +
		"main\t1\t0\t/dev/pts/2\topencode\n" +
		"work\t0\t0\t/dev/pts/5\tvim\n" +
		"work\t0\t1\t/dev/pts/6\topencode\n"

	panes := ParsePanesOutput(input)
	if len(panes) != 4 {
		t.Fatalf("expected 4 panes, got %d", len(panes))
	}

	// Check first pane
	if panes[0].SessionName != "main" || panes[0].WindowIndex != "0" || panes[0].PaneIndex != "0" {
		t.Errorf("pane[0] = %+v", panes[0])
	}
	if panes[0].TTY != "/dev/pts/1" || panes[0].Command != "zsh" {
		t.Errorf("pane[0] tty/cmd = %s / %s", panes[0].TTY, panes[0].Command)
	}

	// Check target formatting
	if panes[1].Target() != "main:1.0" {
		t.Errorf("pane[1].Target() = %q, want %q", panes[1].Target(), "main:1.0")
	}
	if panes[3].Target() != "work:0.1" {
		t.Errorf("pane[3].Target() = %q, want %q", panes[3].Target(), "work:0.1")
	}
}

func TestParsePanesOutputEmpty(t *testing.T) {
	panes := ParsePanesOutput("")
	if len(panes) != 0 {
		t.Fatalf("expected 0 panes, got %d", len(panes))
	}
}

func TestParsePanesOutputMalformed(t *testing.T) {
	// Lines with fewer than 5 tab-separated fields should be skipped
	input := "main\t0\t0\n" +
		"main\t1\t0\t/dev/pts/2\topencode\n"
	panes := ParsePanesOutput(input)
	if len(panes) != 1 {
		t.Fatalf("expected 1 pane (skipping malformed), got %d", len(panes))
	}
}

func TestParsePgrepOutput(t *testing.T) {
	input := "750 opencode serve --port 4096\n" +
		"1001 opencode -s abc123\n" +
		"1002 opencode -s def456\n" +
		"1003 opencode --version\n"

	procs := ParsePgrepOutput(input)
	if len(procs) != 2 {
		t.Fatalf("expected 2 TUI procs, got %d", len(procs))
	}
	if procs[0].PID != "1001" || procs[0].SessionID != "abc123" {
		t.Errorf("procs[0] = %+v", procs[0])
	}
	if procs[1].PID != "1002" || procs[1].SessionID != "def456" {
		t.Errorf("procs[1] = %+v", procs[1])
	}
}

func TestParsePgrepOutputNoSessions(t *testing.T) {
	input := "750 opencode serve --port 4096\n"
	procs := ParsePgrepOutput(input)
	if len(procs) != 0 {
		t.Fatalf("expected 0 procs, got %d", len(procs))
	}
}

func TestParsePsOutput(t *testing.T) {
	input := " 1001 pts/2\n 1002 pts/6\n"
	m := ParsePsOutput(input)
	if m["1001"] != "/dev/pts/2" {
		t.Errorf("pid 1001 tty = %q", m["1001"])
	}
	if m["1002"] != "/dev/pts/6" {
		t.Errorf("pid 1002 tty = %q", m["1002"])
	}
}

func TestParseServerLine(t *testing.T) {
	tests := []struct {
		name string
		line string
		want string
	}{
		{"default port", "750 opencode serve", "http://127.0.0.1:4096"},
		{"custom port", "750 opencode serve --port 8080", "http://127.0.0.1:8080"},
		{"custom host and port", "750 opencode serve --hostname 0.0.0.0 --port 9090", "http://0.0.0.0:9090"},
		{"not serve", "1001 opencode -s abc123", ""},
		{"empty", "", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ParseServerLine(tt.line)
			if got != tt.want {
				t.Errorf("ParseServerLine(%q) = %q, want %q", tt.line, got, tt.want)
			}
		})
	}
}

func TestJoinProcessesToPanes(t *testing.T) {
	procs := []OCProcess{
		{PID: "1001", TTY: "/dev/pts/2", SessionID: "abc123"},
		{PID: "1002", TTY: "/dev/pts/6", SessionID: "def456"},
		{PID: "1003", TTY: "", SessionID: "ghi789"},            // no TTY — should be skipped
		{PID: "1004", TTY: "/dev/pts/99", SessionID: "jkl000"}, // TTY not in any pane
	}
	panes := []Pane{
		{SessionName: "main", WindowIndex: "1", PaneIndex: "0", TTY: "/dev/pts/2", Command: "opencode"},
		{SessionName: "work", WindowIndex: "0", PaneIndex: "1", TTY: "/dev/pts/6", Command: "opencode"},
		{SessionName: "main", WindowIndex: "0", PaneIndex: "0", TTY: "/dev/pts/1", Command: "zsh"},
	}

	result := JoinProcessesToPanes(procs, panes)
	if len(result) != 2 {
		t.Fatalf("expected 2 mappings, got %d: %v", len(result), result)
	}
	if result["abc123"] != "main:1.0" {
		t.Errorf("abc123 -> %q, want %q", result["abc123"], "main:1.0")
	}
	if result["def456"] != "work:0.1" {
		t.Errorf("def456 -> %q, want %q", result["def456"], "work:0.1")
	}
}

func TestJoinProcessesToPanesEmpty(t *testing.T) {
	result := JoinProcessesToPanes(nil, nil)
	if len(result) != 0 {
		t.Fatalf("expected 0 mappings, got %d", len(result))
	}
}
