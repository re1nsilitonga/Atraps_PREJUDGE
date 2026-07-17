// Single-source-access follow-up: this is the Blocker's only realtime
// transport now. It fans out db.ListenDomainBlocked's Postgres NOTIFY
// payloads to connected WebSocket clients, so the extension never needs
// Supabase credentials — everything, live updates included, goes through
// this API (blocker/lib/realtime.js talks to GET /api/v1/realtime).
package main

import (
	"context"
	"log"
	"net/http"
	"sync"
	"time"

	"nhooyr.io/websocket"

	"prejudge/db"
)

// wsHeartbeat matches the interval the old Supabase-direct realtime.js used
// — frequent enough to keep an edge proxy (e.g. Cloudflare Tunnel, ~100s
// idle timeout) from dropping an otherwise-quiet connection.
const wsHeartbeat = 25 * time.Second

// realtimeHub fans out domain_blocked payloads to every connected client.
// A slow/stuck client is dropped from that one broadcast rather than
// blocking the others — it catches up on reconnect via GET /blocklist.
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

// serveWS upgrades to a write-only WebSocket: the client never sends
// anything but pong frames (handled by CloseRead), so there's nothing to
// read back.
func (h *realtimeHub) serveWS(w http.ResponseWriter, r *http.Request) {
	c, err := websocket.Accept(w, r, nil)
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

// runRealtimeListener bridges Postgres NOTIFY to the hub with a fixed
// reconnect backoff — same tradeoff blocker/lib/realtime.js's old adapter
// documented (ponytail: fine for a demo, exponential backoff if this runs
// unattended for real). Exits immediately, once, if DATABASE_DIRECT_URL
// isn't configured, instead of looping on the same error forever.
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
