// Package tui implements the bubbletea TUI for oc-dash.
package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/moonstream-labs/oc-dash/internal/model"
	"github.com/moonstream-labs/oc-dash/internal/opencode"
	tmuxpkg "github.com/moonstream-labs/oc-dash/internal/tmux"
)

// --- Messages ---

type tickMsg time.Time
type sessionsMsg struct {
	sessions []opencode.Session
	statuses map[string]opencode.SessionStatus
}
type tmuxMsg map[string]string // session ID -> tmux target
type messagesMsg struct {
	sessionID string
	messages  []opencode.MessageWithParts
}
type healthMsg struct {
	version string
	err     error
}
type sseMsg opencode.SSEvent
type errMsg error

// --- Model ---

// Model is the root bubbletea model.
type Model struct {
	state    *model.State
	client   *opencode.Client
	cursor   int
	width    int
	height   int
	quitting bool
	sseCtx   context.Context
	sseStop  context.CancelFunc
}

// NewModel creates the initial TUI model.
func NewModel(serverURL string) Model {
	ctx, cancel := context.WithCancel(context.Background())
	return Model{
		state:   model.NewState(serverURL),
		client:  opencode.NewClient(serverURL),
		sseCtx:  ctx,
		sseStop: cancel,
	}
}

// --- Init ---

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		checkHealth(m.client),
		fetchSessions(m.client),
		fetchTmux(),
		tickCmd(),
		subscribeSSE(m.client, m.sseCtx),
	)
}

// --- Update ---

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.KeyMsg:
		switch msg.String() {
		case "q", "ctrl+c":
			m.quitting = true
			m.sseStop()
			return m, tea.Quit
		case "j", "down":
			if m.cursor < len(m.state.Sessions)-1 {
				m.cursor++
			}
			return m, fetchSelectedMessages(m.client, m.state, m.cursor)
		case "k", "up":
			if m.cursor > 0 {
				m.cursor--
			}
			return m, fetchSelectedMessages(m.client, m.state, m.cursor)
		case "enter":
			if m.cursor < len(m.state.Sessions) {
				sess := m.state.Sessions[m.cursor]
				if sess.TmuxTarget != "" {
					_ = tmuxpkg.SelectPane(sess.TmuxTarget)
				}
			}
			return m, nil
		case "r":
			return m, tea.Batch(fetchSessions(m.client), fetchTmux())
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case healthMsg:
		if msg.err != nil {
			m.state.Connected = false
			m.state.Error = msg.err.Error()
		} else {
			m.state.Connected = true
			m.state.Version = msg.version
			m.state.Error = ""
		}

	case sessionsMsg:
		m.state.UpdateSessions(msg.sessions, msg.statuses)
		if m.cursor >= len(m.state.Sessions) {
			m.cursor = max(0, len(m.state.Sessions)-1)
		}
		return m, fetchSelectedMessages(m.client, m.state, m.cursor)

	case tmuxMsg:
		m.state.UpdateTmuxTargets(map[string]string(msg))

	case messagesMsg:
		m.state.UpdateMessages(msg.sessionID, msg.messages)

	case sseMsg:
		evt := opencode.SSEvent(msg)
		_ = evt // TODO: update state based on event type
		return m, tea.Batch(fetchSessions(m.client), fetchTmux())

	case tickMsg:
		return m, tea.Batch(
			fetchSessions(m.client),
			fetchTmux(),
			checkHealth(m.client),
			tickCmd(),
		)

	case errMsg:
		m.state.Error = msg.Error()
	}

	return m, nil
}

// --- View ---

func (m Model) View() string {
	if m.quitting {
		return ""
	}

	if m.width == 0 {
		return "Initializing..."
	}

	// Layout constants
	listWidth := min(40, m.width/3)
	detailWidth := m.width - listWidth - 3 // borders + gap

	// Header
	header := m.renderHeader()

	// Left panel: session list
	listPanel := m.renderSessionList(listWidth, m.height-4)

	// Right panel: detail + activity
	detailPanel := m.renderDetail(detailWidth, m.height-4)

	// Combine panels
	body := lipgloss.JoinHorizontal(lipgloss.Top, listPanel, " ", detailPanel)

	// Status bar
	statusBar := m.renderStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left, header, body, statusBar)
}

// --- Render helpers ---

func (m Model) renderHeader() string {
	var counts [5]int
	for _, s := range m.state.Sessions {
		switch s.Status {
		case "running":
			counts[0]++
		case "waiting":
			counts[1]++
		case "idle":
			counts[2]++
		case "error":
			counts[3]++
		default:
			counts[4]++
		}
	}
	summary := fmt.Sprintf("Sessions: %d total", len(m.state.Sessions))
	parts := []string{}
	if counts[0] > 0 {
		parts = append(parts, fmt.Sprintf("%d running", counts[0]))
	}
	if counts[1] > 0 {
		parts = append(parts, fmt.Sprintf("%d waiting", counts[1]))
	}
	if counts[2] > 0 {
		parts = append(parts, fmt.Sprintf("%d idle", counts[2]))
	}
	if counts[3] > 0 {
		parts = append(parts, fmt.Sprintf("%d error", counts[3]))
	}
	if len(parts) > 0 {
		summary += " (" + strings.Join(parts, ", ") + ")"
	}

	left := titleStyle.Render("oc-dash")
	right := lipgloss.NewStyle().Foreground(colorDim).Render(summary)
	gap := strings.Repeat(" ", max(0, m.width-lipgloss.Width(left)-lipgloss.Width(right)-2))
	return left + gap + right + "\n"
}

func (m Model) renderSessionList(width, height int) string {
	var sb strings.Builder
	sb.WriteString(sectionTitleStyle.Render("Sessions") + "\n")

	visibleHeight := height - 3
	startIdx := 0
	if m.cursor >= visibleHeight {
		startIdx = m.cursor - visibleHeight + 1
	}

	for i := startIdx; i < len(m.state.Sessions) && i < startIdx+visibleHeight; i++ {
		sess := m.state.Sessions[i]
		indicator := StatusIndicator(sess.Status)
		tag := StatusTag(sess.Status)

		title := sess.Title
		if title == "" {
			title = sess.ID[:min(16, len(sess.ID))]
		}
		maxTitleLen := width - 10
		if len(title) > maxTitleLen {
			title = title[:maxTitleLen-1] + "~"
		}

		tmuxInfo := ""
		if sess.TmuxTarget != "" {
			tmuxInfo = lipgloss.NewStyle().Foreground(colorDim).Render(" " + sess.TmuxTarget)
		}

		line := fmt.Sprintf(" %s %s %s%s", indicator, title, tag, tmuxInfo)

		if i == m.cursor {
			sb.WriteString(selectedItemStyle.Width(width).Render(line) + "\n")
		} else {
			sb.WriteString(listItemStyle.Width(width).Render(line) + "\n")
		}
	}

	// Pad remaining space
	rendered := sb.String()
	lines := strings.Count(rendered, "\n")
	for lines < height {
		rendered += "\n"
		lines++
	}
	return rendered
}

func (m Model) renderDetail(width, height int) string {
	if len(m.state.Sessions) == 0 || m.cursor >= len(m.state.Sessions) {
		return sectionTitleStyle.Render("No sessions") + "\n"
	}

	sess := m.state.Sessions[m.cursor]
	var sb strings.Builder

	// Detail section
	sb.WriteString(sectionTitleStyle.Render("Detail") + "\n")
	sb.WriteString(detailRow("Title", sess.Title, width) + "\n")
	sb.WriteString(detailRow("ID", sess.ID, width) + "\n")
	sb.WriteString(detailRow("Status", StatusIndicator(sess.Status)+" "+sess.Status, width) + "\n")
	if sess.TmuxTarget != "" {
		sb.WriteString(detailRow("tmux", sess.TmuxTarget, width) + "\n")
	}
	sb.WriteString(detailRow("Updated", timeAgo(sess.UpdatedAt), width) + "\n")
	sb.WriteString(detailRow("Created", timeAgo(sess.CreatedAt), width) + "\n")
	sb.WriteString("\n")

	// Activity section
	sb.WriteString(sectionTitleStyle.Render("Recent Activity") + "\n")

	activityHeight := height - 10
	if len(sess.Messages) == 0 {
		sb.WriteString(lipgloss.NewStyle().Foreground(colorDim).Render("  No messages loaded") + "\n")
	} else {
		// Show last N messages that fit
		startIdx := max(0, len(sess.Messages)-activityHeight)
		for i := startIdx; i < len(sess.Messages); i++ {
			msg := sess.Messages[i]
			line := renderMessageLine(msg, width-4)
			sb.WriteString("  " + line + "\n")
		}
	}

	// Pad remaining space
	rendered := sb.String()
	lines := strings.Count(rendered, "\n")
	for lines < height {
		rendered += "\n"
		lines++
	}
	return rendered
}

func (m Model) renderStatusBar() string {
	connStatus := lipgloss.NewStyle().Foreground(colorGreen).Render("connected")
	if !m.state.Connected {
		connStatus = lipgloss.NewStyle().Foreground(colorRed).Render("disconnected")
	}

	server := fmt.Sprintf("Server: %s %s", m.state.ServerURL, connStatus)
	if m.state.Version != "" {
		server += fmt.Sprintf("  v%s", m.state.Version)
	}

	keys := lipgloss.NewStyle().Foreground(colorDim).Render("j/k:nav  enter:jump  r:refresh  q:quit")
	gap := strings.Repeat(" ", max(0, m.width-lipgloss.Width(server)-lipgloss.Width(keys)-2))

	return statusBarStyle.Render(server + gap + keys)
}

func detailRow(label, value string, width int) string {
	return detailLabelStyle.Render(label+":") + " " + detailValueStyle.Width(width-14).Render(value)
}

func renderMessageLine(msg opencode.MessageWithParts, maxWidth int) string {
	ts := msg.Info.CreatedAt.Format("15:04")

	switch msg.Info.Role {
	case "user":
		text := extractText(msg.Parts)
		if len(text) > maxWidth-12 {
			text = text[:maxWidth-15] + "..."
		}
		return lipgloss.NewStyle().Foreground(colorDim).Render("["+ts+"] ") +
			activityUserStyle.Render("User: ") + text
	case "assistant":
		// Show tool calls or text
		for _, p := range msg.Parts {
			if p.Type == "tool-invocation" || p.ToolName != "" {
				name := p.ToolName
				if name == "" {
					name = "tool"
				}
				return lipgloss.NewStyle().Foreground(colorDim).Render("["+ts+"] ") +
					activityToolStyle.Render("Tool: "+name)
			}
		}
		text := extractText(msg.Parts)
		if len(text) > maxWidth-12 {
			text = text[:maxWidth-15] + "..."
		}
		return lipgloss.NewStyle().Foreground(colorDim).Render("["+ts+"] ") +
			activityAssistantStyle.Render("Asst: ") + text
	default:
		return lipgloss.NewStyle().Foreground(colorDim).Render("[" + ts + "] " + msg.Info.Role)
	}
}

func extractText(parts []opencode.Part) string {
	for _, p := range parts {
		if p.Type == "text" && p.Text != "" {
			// Single line, strip newlines
			text := strings.ReplaceAll(p.Text, "\n", " ")
			return strings.TrimSpace(text)
		}
	}
	return ""
}

func timeAgo(t time.Time) string {
	if t.IsZero() {
		return "unknown"
	}
	d := time.Since(t)
	switch {
	case d < time.Minute:
		return "just now"
	case d < time.Hour:
		return fmt.Sprintf("%dm ago", int(d.Minutes()))
	case d < 24*time.Hour:
		return fmt.Sprintf("%dh ago", int(d.Hours()))
	default:
		return fmt.Sprintf("%dd ago", int(d.Hours()/24))
	}
}

// --- Commands ---

func tickCmd() tea.Cmd {
	return tea.Tick(5*time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

func checkHealth(c *opencode.Client) tea.Cmd {
	return func() tea.Msg {
		h, err := c.Health()
		if err != nil {
			return healthMsg{err: err}
		}
		return healthMsg{version: h.Version}
	}
}

func fetchSessions(c *opencode.Client) tea.Cmd {
	return func() tea.Msg {
		sessions, err := c.ListSessions()
		if err != nil {
			return errMsg(err)
		}
		statuses, err := c.SessionStatuses()
		if err != nil {
			statuses = map[string]opencode.SessionStatus{}
		}
		return sessionsMsg{sessions: sessions, statuses: statuses}
	}
}

func fetchTmux() tea.Cmd {
	return func() tea.Msg {
		mapping, err := tmuxpkg.DiscoverOpenCode()
		if err != nil {
			return tmuxMsg{}
		}
		return tmuxMsg(mapping)
	}
}

func fetchSelectedMessages(c *opencode.Client, s *model.State, cursor int) tea.Cmd {
	if cursor >= len(s.Sessions) {
		return nil
	}
	sessionID := s.Sessions[cursor].ID
	return func() tea.Msg {
		msgs, err := c.ListMessages(sessionID, 20)
		if err != nil {
			return errMsg(err)
		}
		return messagesMsg{sessionID: sessionID, messages: msgs}
	}
}

func subscribeSSE(c *opencode.Client, ctx context.Context) tea.Cmd {
	return func() tea.Msg {
		// This runs in a goroutine; we send one event at a time back to the
		// bubbletea runtime. For simplicity we just use the first event to
		// trigger a refresh.
		var firstEvent opencode.SSEvent
		received := make(chan struct{}, 1)

		go c.SubscribeEvents(ctx, func(evt opencode.SSEvent) {
			select {
			case received <- struct{}{}:
				firstEvent = evt
			default:
			}
		})

		select {
		case <-received:
			return sseMsg(firstEvent)
		case <-ctx.Done():
			return nil
		}
	}
}
