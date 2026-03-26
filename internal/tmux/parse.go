package tmux

import "strings"

// ParsePanesOutput parses the tab-separated output of tmux list-panes -a.
// Each line is: session_name\twindow_index\tpane_index\tpane_tty\tpane_current_command
func ParsePanesOutput(output string) []Pane {
	var panes []Pane
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		fields := strings.SplitN(line, "\t", 5)
		if len(fields) < 5 {
			continue
		}
		panes = append(panes, Pane{
			SessionName: fields[0],
			WindowIndex: fields[1],
			PaneIndex:   fields[2],
			TTY:         fields[3],
			Command:     fields[4],
		})
	}
	return panes
}

// ParsePgrepOutput parses `pgrep -a opencode` output and extracts processes
// that have the -s flag (TUI clients).
func ParsePgrepOutput(output string) []OCProcess {
	var procs []OCProcess
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid := fields[0]

		// Look for -s <session_id>
		var sessionID string
		for i, f := range fields {
			if f == "-s" && i+1 < len(fields) {
				sessionID = fields[i+1]
				break
			}
		}
		if sessionID == "" {
			continue
		}

		procs = append(procs, OCProcess{
			PID:       pid,
			SessionID: sessionID,
		})
	}
	return procs
}

// ParsePsOutput parses `ps -o pid=,tty=` output and returns a pid->tty map.
func ParsePsOutput(output string) map[string]string {
	m := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			m[fields[0]] = "/dev/" + fields[1]
		}
	}
	return m
}

// ParseServerLine parses a single pgrep output line for `opencode serve` and
// extracts the host:port. Returns empty string if this is not a serve line.
func ParseServerLine(line string) string {
	if !strings.Contains(line, "serve") {
		return ""
	}
	host := "127.0.0.1"
	port := "4096"
	parts := strings.Fields(line)
	for i, p := range parts {
		if p == "--hostname" && i+1 < len(parts) {
			host = parts[i+1]
		}
		if p == "--port" && i+1 < len(parts) {
			port = parts[i+1]
		}
	}
	return "http://" + host + ":" + port
}

// JoinProcessesToPanes maps OpenCode session IDs to tmux pane targets by
// joining processes and panes on their TTY.
func JoinProcessesToPanes(procs []OCProcess, panes []Pane) map[string]string {
	ttyToTarget := make(map[string]string)
	for _, p := range panes {
		ttyToTarget[p.TTY] = p.Target()
	}

	result := make(map[string]string)
	for _, proc := range procs {
		if proc.SessionID == "" || proc.TTY == "" {
			continue
		}
		if target, ok := ttyToTarget[proc.TTY]; ok {
			result[proc.SessionID] = target
		}
	}
	return result
}
