package opencode

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// EventCallback is called for each SSE event received.
type EventCallback func(event SSEvent)

// SubscribeEvents connects to the SSE /event stream and calls cb for each
// event. It blocks until the context is cancelled or the connection drops.
// On disconnect it will retry after a short delay.
func (c *Client) SubscribeEvents(ctx context.Context, cb EventCallback) error {
	for {
		err := c.readEventStream(ctx, cb)
		if ctx.Err() != nil {
			return ctx.Err()
		}
		if err != nil {
			// Brief backoff before reconnecting
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(2 * time.Second):
			}
		}
	}
}

func (c *Client) readEventStream(ctx context.Context, cb EventCallback) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.BaseURL+"/event", nil)
	if err != nil {
		return fmt.Errorf("sse: create request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")

	// Use a client without the default timeout for long-lived SSE.
	sseClient := &http.Client{}
	resp, err := sseClient.Do(req)
	if err != nil {
		return fmt.Errorf("sse: connect: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("sse: status %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	var eventType string
	var dataLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// Empty line = end of event
			if len(dataLines) > 0 {
				data := strings.Join(dataLines, "\n")
				evt := SSEvent{Type: eventType}
				// Try to parse data as JSON properties
				_ = json.Unmarshal([]byte(data), &evt.Properties)
				if evt.Type == "" {
					evt.Type = "message"
				}
				cb(evt)
			}
			eventType = ""
			dataLines = nil
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		}
	}

	return scanner.Err()
}
