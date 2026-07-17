package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"nhooyr.io/websocket"
)

func TestRealtimeHubBroadcastReachesSubscriber(t *testing.T) {
	hub := newRealtimeHub()
	ch := hub.subscribe()
	defer hub.unsubscribe(ch)

	hub.broadcast([]byte(`{"domain":"test.com"}`))

	select {
	case got := <-ch:
		if string(got) != `{"domain":"test.com"}` {
			t.Fatalf("unexpected payload: %s", got)
		}
	case <-time.After(time.Second):
		t.Fatal("timed out waiting for broadcast")
	}
}

func TestRealtimeHubDropsUnsubscribedClient(t *testing.T) {
	hub := newRealtimeHub()
	ch := hub.subscribe()
	hub.unsubscribe(ch)

	if hub.clientCount() != 0 {
		t.Fatalf("expected 0 clients after unsubscribe, got %d", hub.clientCount())
	}
	if _, ok := <-ch; ok {
		t.Fatal("expected channel to be closed after unsubscribe")
	}
}

func TestRealtimeWSEndToEnd(t *testing.T) {
	hub := newRealtimeHub()
	srv := httptest.NewServer(http.HandlerFunc(hub.serveWS))
	defer srv.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wsURL := "ws" + srv.URL[len("http"):]
	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer conn.CloseNow()

	deadline := time.Now().Add(time.Second)
	for hub.clientCount() == 0 && time.Now().Before(deadline) {
		time.Sleep(10 * time.Millisecond)
	}
	if hub.clientCount() != 1 {
		t.Fatalf("expected 1 subscribed client, got %d", hub.clientCount())
	}

	hub.broadcast([]byte(`{"domain":"sembilan.bobaluck.com"}`))

	_, data, err := conn.Read(ctx)
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}
	if string(data) != `{"domain":"sembilan.bobaluck.com"}` {
		t.Fatalf("unexpected payload: %s", data)
	}
}
