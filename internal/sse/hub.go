package sse

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/jackc/pgx/v5"
)

// Event represents an SSE event broadcast to connected clients.
type Event struct {
	Type string `json:"type"` // e.g. "check.status_changed", "ping.received", "alert.fired"
	Data string `json:"data"` // JSON payload
}

// Hub manages connected SSE clients and fans out events.
type Hub struct {
	mu      sync.Mutex
	clients map[chan Event]struct{}
}

// NewHub creates a ready-to-use Hub.
func NewHub() *Hub {
	return &Hub{
		clients: make(map[chan Event]struct{}),
	}
}

// Register adds a new SSE client. It returns the event channel the client
// should read from and an unregister function that must be called when the
// client disconnects.
func (h *Hub) Register() (chan Event, func()) {
	ch := make(chan Event, 64)

	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()

	unregister := func() {
		h.mu.Lock()
		delete(h.clients, ch)
		h.mu.Unlock()
		// Drain remaining events so senders never block.
		for range ch {
		}
	}

	return ch, unregister
}

// Broadcast sends an event to every connected client. Slow clients that
// cannot keep up are skipped (non-blocking send).
func (h *Hub) Broadcast(event Event) {
	h.mu.Lock()
	defer h.mu.Unlock()

	for ch := range h.clients {
		select {
		case ch <- event:
		default:
			// Client is too slow; drop the event for this client.
		}
	}
}

// notificationPayload is the expected JSON shape from the Postgres NOTIFY payload.
type notificationPayload struct {
	CheckID int    `json:"check_id"`
	Status  string `json:"status"`
	Event   string `json:"event"`
}

// StartListener opens a dedicated Postgres connection, subscribes to the
// check_events channel, and broadcasts every notification to the hub. It
// reconnects with exponential backoff on connection errors and never crashes.
func StartListener(ctx context.Context, databaseURL string, hub *Hub) {
	const channel = "check_events"

	backoff := time.Second
	const maxBackoff = 30 * time.Second

	for {
		if err := listenLoop(ctx, databaseURL, channel, hub); err != nil {
			if ctx.Err() != nil {
				log.Println("sse: listener stopped (context cancelled)")
				return
			}
			log.Printf("sse: listen connection lost: %v — reconnecting in %s", err, backoff)

			select {
			case <-time.After(backoff):
			case <-ctx.Done():
				log.Println("sse: listener stopped (context cancelled)")
				return
			}

			// Exponential backoff: 1s → 2s → 4s → … → 30s cap.
			backoff *= 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
			continue
		}

		// Clean exit (context done inside listenLoop).
		return
	}
}

// listenLoop runs a single LISTEN session. It returns an error on connection
// failure so the caller can reconnect.
func listenLoop(ctx context.Context, databaseURL, channel string, hub *Hub) error {
	conn, err := pgx.Connect(ctx, databaseURL)
	if err != nil {
		return fmt.Errorf("connecting: %w", err)
	}
	defer conn.Close(context.Background())

	if _, err := conn.Exec(ctx, "LISTEN "+channel); err != nil {
		return fmt.Errorf("executing LISTEN: %w", err)
	}

	log.Printf("sse: listening on channel %q", channel)

	for {
		notification, err := conn.WaitForNotification(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil // graceful shutdown
			}
			return fmt.Errorf("waiting for notification: %w", err)
		}

		var payload notificationPayload
		if err := json.Unmarshal([]byte(notification.Payload), &payload); err != nil {
			log.Printf("sse: invalid notification payload: %v", err)
			continue
		}

		hub.Broadcast(Event{
			Type: payload.Event,
			Data: notification.Payload,
		})
	}
}
