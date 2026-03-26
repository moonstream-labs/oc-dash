// Package model holds the unified application state.
package model

import (
	"sort"
	"time"

	"github.com/moonstream-labs/oc-dash/internal/opencode"
)

// SessionState represents the dashboard's view of a single session.
type SessionState struct {
	// From the OpenCode API
	ID        string
	Title     string
	Status    string // "running", "idle", "error", "waiting", "completed"
	CreatedAt time.Time
	UpdatedAt time.Time

	// From tmux discovery
	TmuxTarget string // e.g. "main:4.1", empty if not in a tmux pane

	// Recent activity
	Messages []opencode.MessageWithParts
}

// State holds all dashboard state.
type State struct {
	ServerURL string
	Version   string
	Connected bool
	Error     string

	Sessions []SessionState

	// Index maps session ID -> index in Sessions slice
	Index map[string]int
}

// NewState creates an empty state.
func NewState(serverURL string) *State {
	return &State{
		ServerURL: serverURL,
		Index:     make(map[string]int),
	}
}

// UpdateSessions replaces the session list with fresh data from the API.
func (s *State) UpdateSessions(sessions []opencode.Session, statuses map[string]SessionStatus) {
	// Preserve existing tmux targets and messages
	oldTargets := make(map[string]string)
	oldMessages := make(map[string][]opencode.MessageWithParts)
	for _, ss := range s.Sessions {
		if ss.TmuxTarget != "" {
			oldTargets[ss.ID] = ss.TmuxTarget
		}
		if len(ss.Messages) > 0 {
			oldMessages[ss.ID] = ss.Messages
		}
	}

	s.Sessions = make([]SessionState, 0, len(sessions))
	s.Index = make(map[string]int, len(sessions))

	for _, sess := range sessions {
		status := "completed"
		if st, ok := statuses[sess.ID]; ok && st.Status != "" {
			status = st.Status
		}

		ss := SessionState{
			ID:        sess.ID,
			Title:     sess.Title,
			Status:    status,
			CreatedAt: sess.CreatedAt,
			UpdatedAt: sess.UpdatedAt,
		}

		// Restore preserved data
		if target, ok := oldTargets[sess.ID]; ok {
			ss.TmuxTarget = target
		}
		if msgs, ok := oldMessages[sess.ID]; ok {
			ss.Messages = msgs
		}

		s.Sessions = append(s.Sessions, ss)
	}

	// Sort: active sessions first (running > waiting > idle > error > completed),
	// then by UpdatedAt descending.
	sort.Slice(s.Sessions, func(i, j int) bool {
		pi, pj := statusPriority(s.Sessions[i].Status), statusPriority(s.Sessions[j].Status)
		if pi != pj {
			return pi < pj
		}
		return s.Sessions[i].UpdatedAt.After(s.Sessions[j].UpdatedAt)
	})

	// Rebuild index
	for i, ss := range s.Sessions {
		s.Index[ss.ID] = i
	}
}

// UpdateTmuxTargets sets the tmux pane mapping for sessions.
func (s *State) UpdateTmuxTargets(mapping map[string]string) {
	// Clear all targets first
	for i := range s.Sessions {
		s.Sessions[i].TmuxTarget = ""
	}
	// Set discovered targets
	for sessionID, target := range mapping {
		if idx, ok := s.Index[sessionID]; ok {
			s.Sessions[idx].TmuxTarget = target
		}
	}
}

// UpdateMessages sets the recent messages for a session.
func (s *State) UpdateMessages(sessionID string, msgs []opencode.MessageWithParts) {
	if idx, ok := s.Index[sessionID]; ok {
		s.Sessions[idx].Messages = msgs
	}
}

// SessionStatus is a re-export to avoid import cycles.
type SessionStatus = opencode.SessionStatus

func statusPriority(status string) int {
	switch status {
	case "running":
		return 0
	case "waiting":
		return 1
	case "idle":
		return 2
	case "error":
		return 3
	default:
		return 4
	}
}
