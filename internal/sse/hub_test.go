package sse

import (
	"sync"
	"testing"
	"time"
)

func TestNewHub(t *testing.T) {
	hub := NewHub()
	if hub == nil {
		t.Fatal("NewHub returned nil")
	}
	if len(hub.clients) != 0 {
		t.Errorf("new hub should have 0 clients, got %d", len(hub.clients))
	}
}

func TestHub_RegisterAndUnregister(t *testing.T) {
	hub := NewHub()

	ch, unregister := hub.Register()
	if ch == nil {
		t.Fatal("Register returned nil channel")
	}

	hub.mu.Lock()
	count := len(hub.clients)
	hub.mu.Unlock()
	if count != 1 {
		t.Errorf("expected 1 client, got %d", count)
	}

	// Close the channel before unregistering so the drain loop can finish
	close(ch)
	unregister()

	hub.mu.Lock()
	count = len(hub.clients)
	hub.mu.Unlock()
	if count != 0 {
		t.Errorf("expected 0 clients after unregister, got %d", count)
	}
}

func TestHub_BroadcastToSingleClient(t *testing.T) {
	hub := NewHub()
	ch, unregister := hub.Register()
	defer func() {
		close(ch)
		unregister()
	}()

	event := Event{Type: "ping_received", Data: `{"check_id":"123"}`}
	hub.Broadcast(event)

	select {
	case received := <-ch:
		if received.Type != event.Type {
			t.Errorf("Type = %q, want %q", received.Type, event.Type)
		}
		if received.Data != event.Data {
			t.Errorf("Data = %q, want %q", received.Data, event.Data)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast")
	}
}

func TestHub_BroadcastToMultipleClients(t *testing.T) {
	hub := NewHub()

	const numClients = 5
	channels := make([]chan Event, numClients)
	unregisters := make([]func(), numClients)

	for i := 0; i < numClients; i++ {
		channels[i], unregisters[i] = hub.Register()
	}
	defer func() {
		for i := 0; i < numClients; i++ {
			close(channels[i])
			unregisters[i]()
		}
	}()

	event := Event{Type: "status_changed", Data: `{"status":"down"}`}
	hub.Broadcast(event)

	for i, ch := range channels {
		select {
		case received := <-ch:
			if received.Type != event.Type {
				t.Errorf("client %d: Type = %q, want %q", i, received.Type, event.Type)
			}
		case <-time.After(time.Second):
			t.Fatalf("client %d: timed out", i)
		}
	}
}

func TestHub_BroadcastSkipsSlowClient(t *testing.T) {
	hub := NewHub()
	ch, unregister := hub.Register()
	defer func() {
		close(ch)
		unregister()
	}()

	// Fill the channel buffer (capacity is 64)
	for i := 0; i < 64; i++ {
		hub.Broadcast(Event{Type: "fill", Data: "x"})
	}

	// This broadcast should be dropped, not block
	done := make(chan struct{})
	go func() {
		hub.Broadcast(Event{Type: "dropped", Data: "y"})
		close(done)
	}()

	select {
	case <-done:
		// Broadcast returned without blocking — correct behavior
	case <-time.After(time.Second):
		t.Fatal("Broadcast blocked on slow client")
	}
}

func TestHub_ConcurrentBroadcast(t *testing.T) {
	hub := NewHub()
	ch, unregister := hub.Register()
	defer func() {
		close(ch)
		unregister()
	}()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hub.Broadcast(Event{Type: "test", Data: "concurrent"})
		}()
	}

	wg.Wait()

	// Drain — should have received up to 10 events
	count := 0
	for {
		select {
		case <-ch:
			count++
		default:
			goto done
		}
	}
done:
	if count != 10 {
		t.Errorf("expected 10 events, got %d", count)
	}
}

func TestHub_NoClientsNoPanic(t *testing.T) {
	hub := NewHub()
	// Should not panic with no clients
	hub.Broadcast(Event{Type: "test", Data: "nobody"})
}
