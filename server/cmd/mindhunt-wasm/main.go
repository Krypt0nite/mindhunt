//go:build js && wasm

// Command mindhunt-wasm is the game server built for a Cloudflare Durable
// Object (worker/). It wires the same packages as cmd/mindhunt, but the
// Worker owns HTTP and sockets and calls into Go through globalThis.mindhunt:
//
//	http(method, url, headersJSON, body, clientIP) -> {status, headers, body}
//	wsOpen(id, token, room) -> null, or {code, reason} to refuse the socket
//	wsMessage(id, text)
//	wsClose(id)
//
// Go calls back through the host object passed to the Durable Object:
// host.send(id, text), host.close(id, code, reason) and host.sql(query, argsJSON).
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"syscall/js"
	"time"
	_ "time/tzdata"

	"mindhunt/internal/config"
	"mindhunt/internal/custom"
	"mindhunt/internal/daily"
	"mindhunt/internal/game"
	"mindhunt/internal/room"
	"mindhunt/internal/store"
	"mindhunt/internal/transport"
	"mindhunt/packs"
)

func main() {
	slog.SetDefault(slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelInfo})))
	host := js.Global().Get("mindhuntHost")
	if err := run(host); err != nil {
		slog.Error("fatal", "err", err)
		js.Global().Set("mindhuntError", err.Error())
		return
	}
	select {} // keep the callbacks alive
}

func run(host js.Value) error {
	cfg := config.Load()

	reg, err := game.LoadRegistry(packs.FS)
	if err != nil {
		return err
	}
	if err := reg.SetDefault(cfg.DefaultPack); err != nil {
		return err
	}

	db := &sqlDB{fn: host.Get("sql")}
	st, err := newSQLStore(db)
	if err != nil {
		return fmt.Errorf("storage: %w", err)
	}

	// Images are served by the Worker's static assets; MINDHUNT_ASSETS lists
	// the files that were shipped (comma separated, relative to /assets/).
	shipped := map[string]bool{}
	for _, a := range strings.Split(os.Getenv("MINDHUNT_ASSETS"), ",") {
		if a = strings.TrimSpace(a); a != "" {
			shipped[a] = true
		}
	}
	for id, n := range reg.DropImages(func(asset string) bool {
		return transport.AssetPathOK(asset) && shipped[asset]
	}) {
		slog.Warn("images missing from the build; using initials", "pack", id, "count", n)
	}

	set := room.DefaultSettings()
	set.Rules.WrongGuessCost = cfg.WrongGuessCost
	set.ReconnectGrace = cfg.ReconnectGrace
	custom.AddBlockedWords(cfg.BlockedWords)
	lib := custom.NewLibrary(st, custom.Config{})
	mgr := room.NewManager(game.Catalog{Registry: reg, Custom: lib.Cached}, set, store.Noop{})
	key := cfg.DailyKey
	if key == "" {
		key = "mindhunt-development-daily-key"
		slog.Warn("DAILY_KEY not set: using a development key, so daily answers are predictable from the source")
	}
	loc, err := time.LoadLocation(cfg.DailyTZ)
	if err != nil {
		return err
	}
	dailyPacks := []string{cfg.DailyPack}
	if cfg.DailyPack == "rotate" { // default pack first, then the rest alphabetically
		dailyPacks = []string{cfg.DefaultPack}
		for _, id := range reg.IDs() {
			if id != cfg.DefaultPack {
				dailyPacks = append(dailyPacks, id)
			}
		}
	}
	dailySvc, err := daily.New(daily.Config{
		PackIDs: dailyPacks, PoolSize: cfg.DailyPool, Key: []byte(key), Location: loc,
		Epoch: cfg.DailyEpoch, Rules: set.Rules, ImageBase: set.ImageBaseURL,
	}, reg, st)
	if err != nil {
		return err
	}
	srv := transport.New(cfg, st, mgr, reg, dailySvc, lib)
	handler := srv.Handler()
	sockets := newSockets(srv, host)

	api := map[string]any{
		"http": js.FuncOf(func(_ js.Value, a []js.Value) any {
			return serveHTTP(handler, a[0].String(), a[1].String(), a[2].String(), a[3].String(), a[4].String())
		}),
		"wsOpen": js.FuncOf(func(_ js.Value, a []js.Value) any {
			return sockets.open(a[0].Int(), a[1].String(), a[2].String())
		}),
		"wsMessage": js.FuncOf(func(_ js.Value, a []js.Value) any {
			sockets.message(a[0].Int(), a[1].String())
			return nil
		}),
		"wsClose": js.FuncOf(func(_ js.Value, a []js.Value) any {
			sockets.close(a[0].Int())
			return nil
		}),
	}
	js.Global().Set("mindhunt", js.ValueOf(api))
	slog.Info("ready", "app", cfg.AppName, "packs", len(reg.IDs()))
	return nil
}

// serveHTTP runs one request through the normal handler.
func serveHTTP(h http.Handler, method, url, headersJSON, body, ip string) (out any) {
	defer func() {
		if r := recover(); r != nil {
			slog.Error("http panic", "err", r, "url", url)
			out = map[string]any{"status": 500, "headers": `{"Content-Type":["application/json"]}`,
				"body": `{"code":"internal","message":"server error"}`}
		}
	}()
	req := httptest.NewRequest(method, url, strings.NewReader(body))
	var hdr map[string]string
	if err := json.Unmarshal([]byte(headersJSON), &hdr); err == nil {
		for k, v := range hdr {
			req.Header.Set(k, v)
		}
	}
	if body == "" {
		req.ContentLength = 0
	}
	req.RemoteAddr = net.JoinHostPort(ip, "0")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	hb, _ := json.Marshal(rec.Header())
	return map[string]any{"status": rec.Code, "headers": string(hb), "body": rec.Body.String()}
}

// sockets tracks the Worker's WebSockets by id.
type sockets struct {
	srv  *transport.Server
	host js.Value

	mu    sync.Mutex
	conns map[int]*socket
}

type socket struct {
	rm   *room.Room
	conn *room.Conn
	stop chan struct{}
	once sync.Once
}

func newSockets(srv *transport.Server, host js.Value) *sockets {
	return &sockets{srv: srv, host: host, conns: map[int]*socket{}}
}

func (s *sockets) open(id int, token, code string) any {
	rm, conn, closeCode, reason := s.srv.JoinSocket(context.Background(), token, code)
	if rm == nil {
		return map[string]any{"code": closeCode, "reason": reason}
	}
	sk := &socket{rm: rm, conn: conn, stop: make(chan struct{})}
	s.mu.Lock()
	s.conns[id] = sk
	s.mu.Unlock()
	go s.writeLoop(id, sk)
	return nil
}

// writeLoop sends room messages to the socket, like transport's writeLoop.
func (s *sockets) writeLoop(id int, sk *socket) {
	for {
		select {
		case b := <-sk.conn.Outbox():
			s.host.Call("send", id, string(b))
		case <-sk.conn.Done():
			// Flush anything queued (e.g. a "kicked" notice) before closing.
			for {
				select {
				case b := <-sk.conn.Outbox():
					s.host.Call("send", id, string(b))
					continue
				default:
				}
				break
			}
			s.host.Call("close", id, transport.CloseReplaced, "connection closed by server")
			return
		case <-sk.stop:
			return
		}
	}
}

func (s *sockets) message(id int, text string) {
	s.mu.Lock()
	sk := s.conns[id]
	s.mu.Unlock()
	if sk == nil || len(text) > 8<<10 {
		return
	}
	var m room.ClientMsg
	if err := json.Unmarshal([]byte(text), &m); err != nil || m.Type == "" {
		return
	}
	sk.rm.Deliver(sk.conn, m)
}

func (s *sockets) close(id int) {
	s.mu.Lock()
	sk := s.conns[id]
	delete(s.conns, id)
	s.mu.Unlock()
	if sk == nil {
		return
	}
	sk.once.Do(func() {
		close(sk.stop)
		sk.rm.Leave(sk.conn)
		sk.conn.Close()
	})
}
