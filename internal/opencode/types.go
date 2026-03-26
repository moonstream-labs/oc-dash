// Package opencode provides an HTTP client for the OpenCode server API.
package opencode

import "time"

// HealthResponse from GET /global/health.
type HealthResponse struct {
	Healthy bool   `json:"healthy"`
	Version string `json:"version"`
}

// Session represents an OpenCode session from GET /session.
type Session struct {
	ID        string    `json:"id"`
	ParentID  string    `json:"parentID,omitempty"`
	Title     string    `json:"title"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	Share     string    `json:"share,omitempty"`
}

// SessionStatus from GET /session/status.
// The exact shape depends on the OpenCode version; we capture the fields
// that are useful for the dashboard.
type SessionStatus struct {
	Status string `json:"status,omitempty"` // e.g. "running", "idle", "error"
}

// Message represents an OpenCode message.
type Message struct {
	ID        string    `json:"id"`
	SessionID string    `json:"sessionID"`
	Role      string    `json:"role"` // "user", "assistant"
	CreatedAt time.Time `json:"createdAt"`
}

// Part represents a message part.
type Part struct {
	Type string `json:"type"` // "text", "tool-invocation", "tool-result", etc.
	Text string `json:"text,omitempty"`
	// Tool invocation fields
	ToolName string `json:"toolName,omitempty"`
	State    string `json:"state,omitempty"` // "running", "complete", etc.
}

// MessageWithParts combines a message with its parts.
type MessageWithParts struct {
	Info  Message `json:"info"`
	Parts []Part  `json:"parts"`
}

// SSEvent represents a Server-Sent Event from the /event stream.
type SSEvent struct {
	Type       string                 `json:"type"`
	Properties map[string]interface{} `json:"properties,omitempty"`
}
