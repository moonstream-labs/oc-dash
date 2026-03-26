// Package tmux provides functions to query tmux and discover OpenCode sessions.
package tmux

import (
	"fmt"
	"os/exec"
	"strings"
)

// Pane represents a tmux pane with its metadata.
type Pane struct {
	SessionName string // tmux session name (e.g. "main")
	WindowIndex string // window index (e.g. "4")
	PaneIndex   string // pane index (e.g. "1")
	TTY         string // e.g. "/dev/pts/19"
	Command     string // foreground command (e.g. "opencode")
}

// Target returns the tmux target string for this pane (e.g. "main:4.1").
func (p Pane) Target() string {
	return fmt.Sprintf("%s:%s.%s", p.SessionName, p.WindowIndex, p.PaneIndex)
}

// ListPanes returns all tmux panes across all sessions.
func ListPanes() ([]Pane, error) {
	out, err := exec.Command(
		"tmux", "list-panes", "-a",
		"-F", "#{session_name}\t#{window_index}\t#{pane_index}\t#{pane_tty}\t#{pane_current_command}",
	).Output()
	if err != nil {
		return nil, fmt.Errorf("tmux list-panes: %w", err)
	}

	var panes []Pane
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
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
	return panes, nil
}

// SelectPane switches the tmux client to the given pane target.
func SelectPane(target string) error {
	// Parse "session:window.pane" into session:window and pane
	parts := strings.SplitN(target, ".", 2)
	if len(parts) < 2 {
		return fmt.Errorf("invalid target: %s", target)
	}
	sessionWindow := parts[0]

	// First switch to the session, then select the window, then select the pane
	if err := exec.Command("tmux", "switch-client", "-t", sessionWindow).Run(); err != nil {
		// If switch-client fails (e.g. we're not in tmux), try select-window
		_ = exec.Command("tmux", "select-window", "-t", sessionWindow).Run()
	}
	return exec.Command("tmux", "select-pane", "-t", target).Run()
}
