package transport

import (
	"context"
	"errors"
	"strings"

	"mindhunt/internal/room"
	"mindhunt/internal/store"
)

// WebSocket close codes in the private range, read by the client.
const (
	closeRoomNotFound = 4004
	closeRoomFull     = 4001
	closeReplaced     = 4002
	closeUnauthorized = 4003
)

// CloseReplaced is the close code sent when the room drops a connection.
const CloseReplaced = closeReplaced

// JoinSocket authenticates a guest token and attaches a new connection to
// the room. On failure rm is nil and code/reason say how to close the socket.
// The socket owner drains conn.Outbox, passes messages to rm.Deliver and
// calls rm.Leave when the socket closes.
func (s *Server) JoinSocket(ctx context.Context, token, code string) (rm *room.Room, conn *room.Conn, closeCode int, reason string) {
	g, err := s.guests.ByToken(ctx, token)
	if err != nil {
		reason := "unknown guest token"
		if !errors.Is(err, store.ErrNotFound) {
			reason = "auth failed"
		}
		return nil, nil, closeUnauthorized, reason
	}
	rm, ok := s.mgr.Get(strings.ToUpper(code))
	if !ok {
		return nil, nil, closeRoomNotFound, "room not found"
	}
	conn = room.NewConn()
	if err := rm.Join(g.ID, g.Name, conn); err != nil {
		if errors.Is(err, room.ErrRoomFull) {
			return nil, nil, closeRoomFull, "room is full"
		}
		return nil, nil, closeRoomNotFound, "room closed"
	}
	return rm, conn, 0, ""
}
