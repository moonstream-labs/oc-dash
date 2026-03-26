package opencode

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestHealth(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/global/health" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(HealthResponse{Healthy: true, Version: "1.2.3"})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	h, err := c.Health()
	if err != nil {
		t.Fatalf("Health() error: %v", err)
	}
	if !h.Healthy {
		t.Error("expected healthy=true")
	}
	if h.Version != "1.2.3" {
		t.Errorf("version = %q, want %q", h.Version, "1.2.3")
	}
}

func TestHealthServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte("server down"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.Health()
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}

func TestListSessions(t *testing.T) {
	sessions := []Session{
		{ID: "s1", Title: "Session One"},
		{ID: "s2", Title: "Session Two"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/session" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(sessions)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	got, err := c.ListSessions()
	if err != nil {
		t.Fatalf("ListSessions() error: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("expected 2 sessions, got %d", len(got))
	}
	if got[0].ID != "s1" || got[1].Title != "Session Two" {
		t.Errorf("sessions = %+v", got)
	}
}

func TestSessionStatuses(t *testing.T) {
	statuses := map[string]SessionStatus{
		"s1": {Status: "running"},
		"s2": {Status: "idle"},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/session/status" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		json.NewEncoder(w).Encode(statuses)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	got, err := c.SessionStatuses()
	if err != nil {
		t.Fatalf("SessionStatuses() error: %v", err)
	}
	if got["s1"].Status != "running" {
		t.Errorf("s1 status = %q", got["s1"].Status)
	}
	if got["s2"].Status != "idle" {
		t.Errorf("s2 status = %q", got["s2"].Status)
	}
}

func TestListMessages(t *testing.T) {
	msgs := []MessageWithParts{
		{
			Info:  Message{ID: "m1", SessionID: "s1", Role: "user"},
			Parts: []Part{{Type: "text", Text: "hello"}},
		},
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/session/s1/message" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("limit") != "5" {
			t.Errorf("expected limit=5, got %s", r.URL.Query().Get("limit"))
		}
		json.NewEncoder(w).Encode(msgs)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	got, err := c.ListMessages("s1", 5)
	if err != nil {
		t.Fatalf("ListMessages() error: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("expected 1 message, got %d", len(got))
	}
	if got[0].Info.Role != "user" {
		t.Errorf("role = %q", got[0].Info.Role)
	}
	if got[0].Parts[0].Text != "hello" {
		t.Errorf("text = %q", got[0].Parts[0].Text)
	}
}

func TestListMessagesNoLimit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("limit") != "" {
			t.Errorf("expected no limit param, got %s", r.URL.Query().Get("limit"))
		}
		json.NewEncoder(w).Encode([]MessageWithParts{})
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	got, err := c.ListMessages("s1", 0)
	if err != nil {
		t.Fatalf("ListMessages() error: %v", err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0 messages, got %d", len(got))
	}
}

func TestClientConnectionRefused(t *testing.T) {
	c := NewClient("http://127.0.0.1:1") // port 1 should fail
	_, err := c.Health()
	if err == nil {
		t.Fatal("expected connection error")
	}
}

func TestClientBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	_, err := c.Health()
	if err == nil {
		t.Fatal("expected decode error")
	}
}
