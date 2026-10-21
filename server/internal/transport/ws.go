//go:build !js

package transport

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"time"

	"github.com/coder/websocket"

	"mindhunt/internal/room"
)

func (s *Server) ws(w http.ResponseWriter, r *http.Request) {
	opts := &websocket.AcceptOptions{OriginPatterns: s.cfg.AllowedOrigins}
	c, err := websocket.Accept(w, r, opts)
	if err != nil {
		return // Accept already wrote the response
	}
	c.SetReadLimit(8 << 10)

	rm, conn, code, reason := s.JoinSocket(r.Context(), r.URL.Query().Get("token"), r.URL.Query().Get("room"))
	if rm == nil {
		c.Close(websocket.StatusCode(code), reason)
		return
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go writeLoop(ctx, c, conn)

	for {
		_, data, err := c.Read(ctx)
		if err != nil {
			break
		}
		var m room.ClientMsg
		if err := json.Unmarshal(data, &m); err != nil || m.Type == "" {
			continue
		}
		rm.Deliver(conn, m)
	}
	rm.Leave(conn)
	conn.Close()
}

// writeLoop sends room messages to the socket and keeps it alive.
func writeLoop(ctx context.Context, c *websocket.Conn, conn *room.Conn) {
	ping := time.NewTicker(25 * time.Second)
	defer ping.Stop()
	write := func(b []byte) error {
		wctx, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		return c.Write(wctx, websocket.MessageText, b)
	}
	for {
		select {
		case b := <-conn.Outbox():
			if err := write(b); err != nil {
				conn.Close()
				c.CloseNow()
				return
			}
		case <-ping.C:
			pctx, cancel := context.WithTimeout(ctx, 10*time.Second)
			err := c.Ping(pctx)
			cancel()
			if err != nil {
				slog.Debug("ping failed", "err", err)
				c.CloseNow()
				return
			}
		case <-conn.Done():
			// Flush anything queued (e.g. a "kicked" notice) before closing.
			for {
				select {
				case b := <-conn.Outbox():
					_ = write(b)
					continue
				default:
				}
				break
			}
			c.Close(closeReplaced, "connection closed by server")
			return
		case <-ctx.Done():
			return
		}
	}
}
