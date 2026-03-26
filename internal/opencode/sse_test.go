package opencode

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

func TestSSEParsesEvents(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		flusher, ok := w.(http.Flusher)
		if !ok {
			t.Fatal("expected flusher")
		}

		fmt.Fprint(w, "event: session.updated\ndata: {\"id\":\"s1\"}\n\n")
		flusher.Flush()
		fmt.Fprint(w, "event: message.created\ndata: {\"id\":\"m1\"}\n\n")
		flusher.Flush()
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var mu sync.Mutex
	var events []SSEvent

	go c.SubscribeEvents(ctx, func(evt SSEvent) {
		mu.Lock()
		events = append(events, evt)
		mu.Unlock()
		if len(events) >= 2 {
			cancel()
		}
	})

	<-ctx.Done()

	mu.Lock()
	defer mu.Unlock()

	if len(events) < 2 {
		t.Fatalf("expected at least 2 events, got %d", len(events))
	}
	if events[0].Type != "session.updated" {
		t.Errorf("event[0].Type = %q", events[0].Type)
	}
	if events[1].Type != "message.created" {
		t.Errorf("event[1].Type = %q", events[1].Type)
	}
}

func TestSSEDefaultEventType(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		// No event: line, just data
		fmt.Fprint(w, "data: {\"hello\":\"world\"}\n\n")
		w.(http.Flusher).Flush()
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	var got SSEvent
	var once sync.Once

	go c.SubscribeEvents(ctx, func(evt SSEvent) {
		once.Do(func() {
			got = evt
			cancel()
		})
	})

	<-ctx.Done()

	if got.Type != "message" {
		t.Errorf("expected default type 'message', got %q", got.Type)
	}
}

func TestSSEServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer srv.Close()

	c := NewClient(srv.URL)
	ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
	defer cancel()

	// Should not panic, just retry and eventually context cancels
	err := c.readEventStream(ctx, func(evt SSEvent) {})
	if err == nil {
		t.Fatal("expected error for 500 response")
	}
}
