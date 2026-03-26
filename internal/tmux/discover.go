package tmux

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/moonstream-labs/oc-dash/internal/log"
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
	l := log.Get().With("pkg", "tmux")

	// Step 1: Find opencode TUI processes via pgrep
	procs, err := findOCProcesses()
	if err != nil {
		l.Err("pgrep failed", err)
		return nil, err
	}
	if len(procs) == 0 {
		l.Debug("no opencode TUI processes found")
		return map[string]string{}, nil
	}
	l.Debug("found opencode processes", "count", fmt.Sprintf("%d", len(procs)))

	// Step 2: Get TTYs for those PIDs via ps
	if err := fillTTYs(procs); err != nil {
		l.Err("ps tty lookup failed", err)
		return nil, err
	}

	// Step 3: Get all tmux panes
	panes, err := ListPanes()
	if err != nil {
		l.Err("tmux list-panes failed", err)
		return nil, err
	}
	l.Debug("tmux panes", "count", fmt.Sprintf("%d", len(panes)))

	// Step 4+5: Join on TTY
	result := JoinProcessesToPanes(procs, panes)
	l.Info("discovery complete", "mapped", fmt.Sprintf("%d", len(result)))
	return result, nil
}

// DiscoverServer finds the opencode serve process and returns its address.
// Falls back to "http://127.0.0.1:4096" if not found.
func DiscoverServer() string {
	l := log.Get().With("pkg", "tmux")

	out, err := exec.Command("pgrep", "-a", "opencode").Output()
	if err != nil {
		l.Debug("pgrep for server failed, using default")
		return "http://127.0.0.1:4096"
	}

	for _, line := range strings.Split(string(out), "\n") {
		if addr := ParseServerLine(line); addr != "" {
			l.Info("discovered server", "addr", addr)
			return addr
		}
	}
	l.Debug("no opencode serve process found, using default")
	return "http://127.0.0.1:4096"
}

// findOCProcesses uses pgrep to find opencode TUI processes (those with -s flag).
func findOCProcesses() ([]OCProcess, error) {
	out, err := exec.Command("pgrep", "-a", "opencode").Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 {
			return nil, nil
		}
		return nil, err
	}
	return ParsePgrepOutput(string(out)), nil
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

	pidToTTY := ParsePsOutput(string(out))
	for i := range procs {
		if tty, ok := pidToTTY[procs[i].PID]; ok {
			procs[i].TTY = tty
		}
	}
	return nil
}
