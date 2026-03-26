package tmux

import (
	"os/exec"
	"strings"
)

// OCProcess represents a running opencode TUI process.
type OCProcess struct {
	PID       string
	TTY       string
	SessionID string // the -s <session_id> from the command line
}

// DiscoverOpenCode finds all running opencode TUI processes and maps them
// to tmux panes via TTY matching.
//
// Returns a map of OpenCode session ID -> tmux pane target (e.g. "main:4.1").
func DiscoverOpenCode() (map[string]string, error) {
	// Step 1: Find opencode TUI processes via pgrep
	procs, err := findOCProcesses()
	if err != nil {
		return nil, err
	}
	if len(procs) == 0 {
		return map[string]string{}, nil
	}

	// Step 2: Get TTYs for those PIDs via ps
	if err := fillTTYs(procs); err != nil {
		return nil, err
	}

	// Step 3: Get all tmux panes
	panes, err := ListPanes()
	if err != nil {
		return nil, err
	}

	// Step 4: Build TTY -> tmux target map
	ttyToTarget := make(map[string]string)
	for _, p := range panes {
		ttyToTarget[p.TTY] = p.Target()
	}

	// Step 5: Join on TTY
	result := make(map[string]string)
	for _, proc := range procs {
		if proc.SessionID == "" || proc.TTY == "" {
			continue
		}
		if target, ok := ttyToTarget[proc.TTY]; ok {
			result[proc.SessionID] = target
		}
	}
	return result, nil
}

// DiscoverServer finds the opencode serve process and returns its address.
// Falls back to "http://127.0.0.1:4096" if not found.
func DiscoverServer() string {
	out, err := exec.Command("pgrep", "-a", "opencode").Output()
	if err != nil {
		return "http://127.0.0.1:4096"
	}

	for _, line := range strings.Split(string(out), "\n") {
		if !strings.Contains(line, "serve") {
			continue
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
	return "http://127.0.0.1:4096"
}

// findOCProcesses uses pgrep to find opencode TUI processes (those with -s flag).
func findOCProcesses() ([]OCProcess, error) {
	out, err := exec.Command("pgrep", "-a", "opencode").Output()
	if err != nil {
		// pgrep returns exit 1 if no processes found
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, err
	}

	var procs []OCProcess
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 2 {
			continue
		}
		pid := fields[0]

		// Look for -s <session_id> in the command line
		var sessionID string
		for i, f := range fields {
			if f == "-s" && i+1 < len(fields) {
				sessionID = fields[i+1]
				break
			}
		}

		// Skip "opencode serve" and other non-TUI processes
		if sessionID == "" {
			continue
		}

		procs = append(procs, OCProcess{
			PID:       pid,
			SessionID: sessionID,
		})
	}
	return procs, nil
}

// fillTTYs uses ps to look up TTYs for the given processes.
func fillTTYs(procs []OCProcess) error {
	if len(procs) == 0 {
		return nil
	}

	pids := make([]string, len(procs))
	for i, p := range procs {
		pids[i] = p.PID
	}

	out, err := exec.Command("ps", "-o", "pid=,tty=", "-p", strings.Join(pids, ",")).Output()
	if err != nil {
		return err
	}

	pidToTTY := make(map[string]string)
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		fields := strings.Fields(line)
		if len(fields) >= 2 {
			pid := fields[0]
			tty := "/dev/" + fields[1]
			pidToTTY[pid] = tty
		}
	}

	for i := range procs {
		if tty, ok := pidToTTY[procs[i].PID]; ok {
			procs[i].TTY = tty
		}
	}
	return nil
}
