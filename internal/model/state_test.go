package model

import (
	"testing"
	"time"

	"github.com/moonstream-labs/oc-dash/internal/opencode"
)

func TestUpdateSessionsSortsbyPriority(t *testing.T) {
	s := NewState("http://localhost:4096")

	now := time.Now()
	sessions := []opencode.Session{
		{ID: "idle1", Title: "Idle Session", UpdatedAt: now.Add(-1 * time.Hour)},
		{ID: "run1", Title: "Running Session", UpdatedAt: now},
		{ID: "wait1", Title: "Waiting Session", UpdatedAt: now.Add(-30 * time.Minute)},
		{ID: "err1", Title: "Error Session", UpdatedAt: now.Add(-2 * time.Hour)},
	}
	statuses := map[string]opencode.SessionStatus{
		"idle1": {Status: "idle"},
		"run1":  {Status: "running"},
		"wait1": {Status: "waiting"},
		"err1":  {Status: "error"},
	}

	s.UpdateSessions(sessions, statuses)

	if len(s.Sessions) != 4 {
		t.Fatalf("expected 4 sessions, got %d", len(s.Sessions))
	}

	// Expected order: running, waiting, idle, error
	expected := []string{"running", "waiting", "idle", "error"}
	for i, exp := range expected {
		if s.Sessions[i].Status != exp {
			t.Errorf("sessions[%d].Status = %q, want %q", i, s.Sessions[i].Status, exp)
		}
	}
}

func TestUpdateSessionsDefaultStatus(t *testing.T) {
	s := NewState("http://localhost:4096")

	sessions := []opencode.Session{
		{ID: "s1", Title: "No Status"},
	}
	statuses := map[string]opencode.SessionStatus{} // empty

	s.UpdateSessions(sessions, statuses)

	if s.Sessions[0].Status != "completed" {
		t.Errorf("default status = %q, want 'completed'", s.Sessions[0].Status)
	}
}

func TestUpdateSessionsPreservesTmuxTargets(t *testing.T) {
	s := NewState("http://localhost:4096")

	now := time.Now()
	sessions := []opencode.Session{
		{ID: "s1", Title: "Session 1", UpdatedAt: now},
	}
	statuses := map[string]opencode.SessionStatus{
		"s1": {Status: "running"},
	}

	s.UpdateSessions(sessions, statuses)
	s.UpdateTmuxTargets(map[string]string{"s1": "main:1.0"})

	if s.Sessions[0].TmuxTarget != "main:1.0" {
		t.Fatalf("tmux target not set: %q", s.Sessions[0].TmuxTarget)
	}

	// Update sessions again — target should be preserved
	s.UpdateSessions(sessions, statuses)
	if s.Sessions[0].TmuxTarget != "main:1.0" {
		t.Errorf("tmux target lost after UpdateSessions: %q", s.Sessions[0].TmuxTarget)
	}
}

func TestUpdateSessionsPreservesMessages(t *testing.T) {
	s := NewState("http://localhost:4096")

	now := time.Now()
	sessions := []opencode.Session{
		{ID: "s1", Title: "Session 1", UpdatedAt: now},
	}
	statuses := map[string]opencode.SessionStatus{
		"s1": {Status: "idle"},
	}

	s.UpdateSessions(sessions, statuses)

	msgs := []opencode.MessageWithParts{
		{Info: opencode.Message{ID: "m1", Role: "user"}},
	}
	s.UpdateMessages("s1", msgs)

	if len(s.Sessions[0].Messages) != 1 {
		t.Fatalf("messages not set")
	}

	// Update sessions again — messages should be preserved
	s.UpdateSessions(sessions, statuses)
	if len(s.Sessions[0].Messages) != 1 {
		t.Error("messages lost after UpdateSessions")
	}
}

func TestUpdateTmuxTargets(t *testing.T) {
	s := NewState("http://localhost:4096")

	now := time.Now()
	sessions := []opencode.Session{
		{ID: "s1", Title: "A", UpdatedAt: now},
		{ID: "s2", Title: "B", UpdatedAt: now},
	}
	statuses := map[string]opencode.SessionStatus{
		"s1": {Status: "running"},
		"s2": {Status: "idle"},
	}
	s.UpdateSessions(sessions, statuses)

	s.UpdateTmuxTargets(map[string]string{
		"s1": "main:1.0",
		"s2": "work:0.1",
	})

	// Verify targets set
	for _, ss := range s.Sessions {
		if ss.ID == "s1" && ss.TmuxTarget != "main:1.0" {
			t.Errorf("s1 target = %q", ss.TmuxTarget)
		}
		if ss.ID == "s2" && ss.TmuxTarget != "work:0.1" {
			t.Errorf("s2 target = %q", ss.TmuxTarget)
		}
	}

	// Update with only s1 — s2 should be cleared
	s.UpdateTmuxTargets(map[string]string{"s1": "main:2.0"})
	for _, ss := range s.Sessions {
		if ss.ID == "s1" && ss.TmuxTarget != "main:2.0" {
			t.Errorf("s1 target = %q", ss.TmuxTarget)
		}
		if ss.ID == "s2" && ss.TmuxTarget != "" {
			t.Errorf("s2 target should be cleared, got %q", ss.TmuxTarget)
		}
	}
}

func TestUpdateMessagesUnknownSession(t *testing.T) {
	s := NewState("http://localhost:4096")
	// Should not panic
	s.UpdateMessages("nonexistent", []opencode.MessageWithParts{})
}

func TestIndexIsCorrect(t *testing.T) {
	s := NewState("http://localhost:4096")

	now := time.Now()
	sessions := []opencode.Session{
		{ID: "a", Title: "A", UpdatedAt: now},
		{ID: "b", Title: "B", UpdatedAt: now.Add(-time.Hour)},
		{ID: "c", Title: "C", UpdatedAt: now.Add(-2 * time.Hour)},
	}
	statuses := map[string]opencode.SessionStatus{
		"a": {Status: "idle"},
		"b": {Status: "running"},
		"c": {Status: "idle"},
	}
	s.UpdateSessions(sessions, statuses)

	// After sorting, "b" (running) should be first
	for id, idx := range s.Index {
		if s.Sessions[idx].ID != id {
			t.Errorf("Index[%q] = %d, but Sessions[%d].ID = %q", id, idx, idx, s.Sessions[idx].ID)
		}
	}
}

func TestSortSameStatusByUpdatedAt(t *testing.T) {
	s := NewState("http://localhost:4096")

	now := time.Now()
	sessions := []opencode.Session{
		{ID: "old", Title: "Old", UpdatedAt: now.Add(-2 * time.Hour)},
		{ID: "new", Title: "New", UpdatedAt: now},
		{ID: "mid", Title: "Mid", UpdatedAt: now.Add(-1 * time.Hour)},
	}
	statuses := map[string]opencode.SessionStatus{
		"old": {Status: "idle"},
		"new": {Status: "idle"},
		"mid": {Status: "idle"},
	}
	s.UpdateSessions(sessions, statuses)

	// All idle, so should be sorted by UpdatedAt descending: new, mid, old
	if s.Sessions[0].ID != "new" {
		t.Errorf("sessions[0].ID = %q, want 'new'", s.Sessions[0].ID)
	}
	if s.Sessions[1].ID != "mid" {
		t.Errorf("sessions[1].ID = %q, want 'mid'", s.Sessions[1].ID)
	}
	if s.Sessions[2].ID != "old" {
		t.Errorf("sessions[2].ID = %q, want 'old'", s.Sessions[2].ID)
	}
}
