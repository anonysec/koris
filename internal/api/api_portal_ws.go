package api

import (
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// portalUsageWS streams a customer's live usage over WebSocket, mirroring the
// admin /api/realtime endpoint but scoped to a single authenticated customer.
// GET /api/portal/usage/ws → { type: "usage", data: <UsageSummary> } every 5s.
func (s *Server) portalUsageWS(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "method", http.StatusMethodNotAllowed)
		return
	}
	username, ok := s.currentCustomer(r)
	if !ok {
		writeJSONCode(w, http.StatusUnauthorized, map[string]any{"ok": false, "error": "unauthorized"})
		return
	}

	upgrader := websocket.Upgrader{
		ReadBufferSize:  1024,
		WriteBufferSize: 1024,
		CheckOrigin: func(r *http.Request) bool {
			return s.checkWSOrigin(r)
		},
	}
	conn, err := upgrader.Upgrade(w, r, nil)
	if err != nil {
		return
	}
	defer conn.Close()

	_ = conn.SetReadDeadline(time.Now().Add(65 * time.Second))
	conn.SetPongHandler(func(string) error {
		_ = conn.SetReadDeadline(time.Now().Add(65 * time.Second))
		return nil
	})

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			if _, _, err := conn.ReadMessage(); err != nil {
				return
			}
		}
	}()

	var mu sync.Mutex
	writeUsage := func() error {
		usage, err := s.usageForUsername(username)
		if err != nil {
			mu.Lock()
			defer mu.Unlock()
			return conn.WriteJSON(map[string]any{"type": "usage", "error": err.Error()})
		}
		mu.Lock()
		defer mu.Unlock()
		return conn.WriteJSON(map[string]any{"type": "usage", "data": usage})
	}

	_ = writeUsage()

	ticker := time.NewTicker(5 * time.Second)
	pingTicker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	defer pingTicker.Stop()

	for {
		select {
		case <-done:
			return
		case <-ticker.C:
			if err := writeUsage(); err != nil {
				return
			}
		case <-pingTicker.C:
			mu.Lock()
			err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(5*time.Second))
			mu.Unlock()
			if err != nil {
				return
			}
		}
	}
}
