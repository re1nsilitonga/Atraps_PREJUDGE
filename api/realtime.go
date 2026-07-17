package main

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"nhooyr.io/websocket"

	"prime/db"
)

const wsHeartbeat = 25 * time.Second

// chromeExtensionOriginPattern matches a Chrome extension's Origin header
// host (chrome-extension://<32-char id>, seen by nhooyr's OriginPatterns
// as just the 32-char id -- it matches the Origin URL's Host, not its
// scheme). Extension IDs use only a-p (base16 shifted by 'a'), so this
// can't collide with a real attacker-controlled domain, which needs at
// least one dot. Same-origin requests (e.g. curl against localhost:8000
// directly) are already allowed by nhooyr without needing this pattern.
const chromeExtensionOriginPattern = "[a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p][a-p]"

type realtimeHub struct {
	mu      sync.Mutex
	clients map[chan []byte]struct{}
}

func newRealtimeHub() *realtimeHub {
	return &realtimeHub{clients: make(map[chan []byte]struct{})}
}

func (h *realtimeHub) subscribe() chan []byte {
	ch := make(chan []byte, 8)
	h.mu.Lock()
	h.clients[ch] = struct{}{}
	h.mu.Unlock()
	return ch
}

func (h *realtimeHub) unsubscribe(ch chan []byte) {
	h.mu.Lock()
	delete(h.clients, ch)
	h.mu.Unlock()
	close(ch)
}

func (h *realtimeHub) broadcast(payload []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for ch := range h.clients {
		select {
		case ch <- payload:
		default:
		}
	}
}

func (h *realtimeHub) clientCount() int {
	h.mu.Lock()
	defer h.mu.Unlock()
	return len(h.clients)
}

func (h *realtimeHub) serveWS(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		OriginPatterns: []string{chromeExtensionOriginPattern},
	})
	if err != nil {
		log.Printf("realtime: accept failed: %v", err)
		return
	}
	defer c.CloseNow()

	ch := h.subscribe()
	defer h.unsubscribe(ch)

	ctx := c.CloseRead(r.Context())
	ticker := time.NewTicker(wsHeartbeat)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case payload, ok := <-ch:
			if !ok {
				return
			}
			if err := c.Write(ctx, websocket.MessageText, payload); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.Ping(ctx); err != nil {
				return
			}
		}
	}
}

func runRealtimeListener(hub *realtimeHub) {
	if _, err := db.DirectDSNFromEnv(); err != nil {
		log.Printf("realtime listener disabled: %v", err)
		return
	}

	const backoff = 2 * time.Second
	for {
		err := db.ListenDomainBlocked(context.Background(), func(payload string) {
			hub.broadcast([]byte(payload))
		})
		log.Printf("realtime listener stopped, retrying in %s: %v", backoff, err)
		time.Sleep(backoff)
	}
}
