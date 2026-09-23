package sse

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
)

type Event struct {
	Type string `json:"type"`
	Data any    `json:"data"`
}

type Hub struct {
	clients map[chan []byte]bool
	mu      sync.RWMutex
}

func New() *Hub {
	return &Hub{
		clients: make(map[chan []byte]bool),
	}
}

func (h *Hub) Broadcast(eventType string, data any) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	if len(h.clients) == 0 {
		return
	}

	payload, err := json.Marshal(Event{
		Type: eventType,
		Data: data,
	})
	if err != nil {
		return
	}

	msg := fmt.Sprintf("data: %s\n\n", payload)
	msgBytes := []byte(msg)

	for ch := range h.clients {
		select {
		case ch <- msgBytes:
		default:
		}
	}
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming unsupported", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("Access-Control-Allow-Origin", "*")

	clientChan := make(chan []byte, 32)

	h.mu.Lock()
	h.clients[clientChan] = true
	h.mu.Unlock()

	defer func() {
		h.mu.Lock()
		delete(h.clients, clientChan)
		close(clientChan)
		h.mu.Unlock()
	}()

	// Send initial ping
	_, _ = w.Write([]byte(": connected\n\n"))
	flusher.Flush()

	ctx := r.Context()
	for {
		select {
		case <-ctx.Done():
			return
		case msg, ok := <-clientChan:
			if !ok {
				return
			}
			_, _ = w.Write(msg)
			flusher.Flush()
		}
	}
}
