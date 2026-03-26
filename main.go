package main

import (
	"flag"
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/moonstream-labs/oc-dash/internal/log"
	tmuxpkg "github.com/moonstream-labs/oc-dash/internal/tmux"
	"github.com/moonstream-labs/oc-dash/internal/tui"
)

func main() {
	server := flag.String("server", "", "OpenCode server URL (e.g. http://127.0.0.1:4096). Auto-detected if omitted.")
	logPath := flag.String("log", "", "Log file path. Use 'default' for ~/.local/share/oc-dash/oc-dash.log, or pass a path.")
	logLevel := flag.String("log-level", "info", "Log level: debug, info, warn, error")
	flag.Parse()

	// Initialise logger
	lp := *logPath
	if lp == "default" {
		lp = log.DefaultLogPath()
	}
	level := log.LevelInfo
	switch *logLevel {
	case "debug":
		level = log.LevelDebug
	case "warn":
		level = log.LevelWarn
	case "error":
		level = log.LevelError
	}
	if err := log.Init(lp, level); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: could not init logger: %v\n", err)
	}

	serverURL := *server
	if serverURL == "" {
		serverURL = tmuxpkg.DiscoverServer()
	}

	log.Get().Info("starting TUI", "server", serverURL)

	m := tui.NewModel(serverURL)
	p := tea.NewProgram(m, tea.WithAltScreen())

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
}
