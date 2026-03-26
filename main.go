package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	tmuxpkg "github.com/moonstream-labs/oc-dash/internal/tmux"
	"github.com/moonstream-labs/oc-dash/internal/tui"
)

func main() {
	server := flag.String("server", "", "OpenCode server URL (e.g. http://127.0.0.1:4096). Auto-detected if omitted.")
	flag.Parse()

	serverURL := *server
	if serverURL == "" {
		serverURL = tmuxpkg.DiscoverServer()
	}

	m := tui.NewModel(serverURL)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
