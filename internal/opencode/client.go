package opencode

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/moonstream-labs/oc-dash/internal/log"
)

// Client talks to an OpenCode server over HTTP.
type Client struct {
	BaseURL    string
	HTTPClient *http.Client
	log        *log.Logger
}

// NewClient creates a client for the given server URL (e.g. "http://127.0.0.1:4096").
func NewClient(baseURL string) *Client {
	return &Client{
		BaseURL: baseURL,
		HTTPClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		log: log.Get().With("pkg", "opencode"),
	}
}

// Health checks the server and returns version info.
func (c *Client) Health() (*HealthResponse, error) {
	var resp HealthResponse
	if err := c.get("/global/health", &resp); err != nil {
		return nil, err
	}
	c.log.Debug("health check", "healthy", fmt.Sprintf("%v", resp.Healthy), "version", resp.Version)
	return &resp, nil
}

// ListSessions returns all sessions.
func (c *Client) ListSessions() ([]Session, error) {
	var sessions []Session
	if err := c.get("/session", &sessions); err != nil {
		return nil, err
	}
	c.log.Debug("listed sessions", "count", fmt.Sprintf("%d", len(sessions)))
	return sessions, nil
}

// SessionStatuses returns the status of all sessions.
func (c *Client) SessionStatuses() (map[string]SessionStatus, error) {
	var statuses map[string]SessionStatus
	if err := c.get("/session/status", &statuses); err != nil {
		return nil, err
	}
	c.log.Debug("fetched statuses", "count", fmt.Sprintf("%d", len(statuses)))
	return statuses, nil
}

// ListMessages returns the messages for a session. Limit controls how many
// to return (0 = server default).
func (c *Client) ListMessages(sessionID string, limit int) ([]MessageWithParts, error) {
	path := fmt.Sprintf("/session/%s/message", sessionID)
	if limit > 0 {
		path = fmt.Sprintf("%s?limit=%d", path, limit)
	}
	var messages []MessageWithParts
	if err := c.get(path, &messages); err != nil {
		return nil, err
	}
	c.log.Debug("listed messages", "session", sessionID, "count", fmt.Sprintf("%d", len(messages)))
	return messages, nil
}

// get performs a GET request and JSON-decodes the response into dest.
func (c *Client) get(path string, dest interface{}) error {
	url := c.BaseURL + path
	resp, err := c.HTTPClient.Get(url)
	if err != nil {
		c.log.Error("HTTP request failed", "path", path, "err", err.Error())
		return fmt.Errorf("GET %s: %w", path, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		c.log.Warn("HTTP non-200", "path", path, "status", fmt.Sprintf("%d", resp.StatusCode))
		return fmt.Errorf("GET %s: status %d: %s", path, resp.StatusCode, string(body))
	}

	if err := json.NewDecoder(resp.Body).Decode(dest); err != nil {
		c.log.Error("JSON decode failed", "path", path, "err", err.Error())
		return fmt.Errorf("GET %s: decode: %w", path, err)
	}
	return nil
}
