package transport

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"

	"mindhunt/internal/config"
	"mindhunt/internal/custom"
	"mindhunt/internal/daily"
	"mindhunt/internal/game"
	"mindhunt/internal/room"
	"mindhunt/internal/store"
	"mindhunt/packs"
)

func newTestServer(t *testing.T) *httptest.Server {
	t.Helper()
	reg, err := game.LoadRegistry(packs.FS)
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{AppName: "MindHunt", WebDir: t.TempDir(), WrongGuessCost: 3,
		ReconnectGrace: 90 * time.Second, RoomsPerIPHour: 3}
	set := room.DefaultSettings()
	set.BotDelay = 10 * time.Millisecond
	lib := custom.NewLibrary(store.NewMemoryCustomPacks(), custom.Config{})
	mgr := room.NewManager(game.Catalog{Registry: reg, Custom: lib.Cached}, set, store.Noop{})
	loc, _ := time.LoadLocation("Asia/Kolkata")
	d, err := daily.New(daily.Config{PackIDs: []string{"cricket-men-intl"}, PoolSize: 48, Key: []byte("test"), Location: loc,
		Epoch: "2026-10-18", Rules: game.DefaultRules, ImageBase: "/assets"}, reg, store.NewMemoryDaily())
	if err != nil {
		t.Fatal(err)
	}
	ts := httptest.NewServer(New(cfg, store.NewMemoryGuests(), mgr, reg, d, lib).Handler())
	t.Cleanup(ts.Close)
	return ts
}

func api(t *testing.T, ts *httptest.Server, method, path, token string, body any, out any) int {
	t.Helper()
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	req, _ := http.NewRequest(method, ts.URL+path, &buf)
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	if out != nil {
		json.NewDecoder(res.Body).Decode(out)
	}
	return res.StatusCode
}

type wsClient struct {
	t      *testing.T
	c      *websocket.Conn
	cseq   int64
	msgs   chan room.ServerMsg
	raw    chan string
	closed chan struct{}
	err    error // read error that ended the socket (holds the close code)
}

func dial(t *testing.T, ts *httptest.Server, code, token string) (*wsClient, *websocket.Conn, error) {
	t.Helper()
	url := "ws" + strings.TrimPrefix(ts.URL, "http") + "/ws?room=" + code + "&token=" + token
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	c, _, err := websocket.Dial(ctx, url, nil)
	if err != nil {
		return nil, nil, err
	}
	w := &wsClient{t: t, c: c, msgs: make(chan room.ServerMsg, 256), raw: make(chan string, 256), closed: make(chan struct{})}
	go func() {
		for {
			_, data, err := c.Read(context.Background())
			if err != nil {
				w.err = err
				close(w.closed)
				close(w.msgs)
				return
			}
			var m room.ServerMsg
			json.Unmarshal(data, &m)
			w.raw <- string(data)
			w.msgs <- m
		}
	}()
	return w, c, nil
}

func (w *wsClient) send(typ string, data any) {
	w.cseq++
	raw, _ := json.Marshal(data)
	b, _ := json.Marshal(room.ClientMsg{Type: typ, Cseq: w.cseq, Data: raw})
	if err := w.c.Write(context.Background(), websocket.MessageText, b); err != nil {
		w.t.Fatal(err)
	}
}

// next waits for a message of type typ whose raw JSON contains all needles.
func (w *wsClient) next(typ string, needles ...string) string {
	w.t.Helper()
	timeout := time.After(3 * time.Second)
	for {
		select {
		case m, ok := <-w.msgs:
			if !ok {
				w.t.Fatalf("socket closed waiting for %s %v", typ, needles)
			}
			raw := <-w.raw
			if m.Type != typ {
				continue
			}
			match := true
			for _, n := range needles {
				if !strings.Contains(raw, n) {
					match = false
				}
			}
			if match {
				return raw
			}
		case <-timeout:
			w.t.Fatalf("timed out waiting for %s %v", typ, needles)
		}
	}
}

func closeCode(t *testing.T, w *wsClient) websocket.StatusCode {
	t.Helper()
	select {
	case <-w.closed:
		return websocket.CloseStatus(w.err)
	case <-time.After(3 * time.Second):
		t.Fatal("socket not closed")
		return 0
	}
}

func TestRESTAndFullRoundOverWebSocket(t *testing.T) {
	ts := newTestServer(t)

	var cfg map[string]any
	if api(t, ts, "GET", "/api/config", "", nil, &cfg) != 200 || cfg["app_name"] != "MindHunt" {
		t.Fatalf("config: %v", cfg)
	}
	if api(t, ts, "POST", "/api/guest", "", map[string]string{"display_name": ""}, nil) != 400 {
		t.Fatal("empty name accepted")
	}
	var ga, gb, gc store.Guest
	api(t, ts, "POST", "/api/guest", "", map[string]string{"display_name": "Asha"}, &ga)
	api(t, ts, "POST", "/api/guest", "", map[string]string{"display_name": "Bilal"}, &gb)
	api(t, ts, "POST", "/api/guest", "", map[string]string{"display_name": "Chen"}, &gc)
	if ga.Token == "" || ga.ID == "" {
		t.Fatalf("guest: %+v", ga)
	}
	var me store.Guest
	if api(t, ts, "GET", "/api/me", ga.Token, nil, &me) != 200 || me.Token != "" || me.Name != "Asha" {
		t.Fatalf("me: %+v", me)
	}
	if api(t, ts, "POST", "/api/rooms", "", nil, nil) != 401 {
		t.Fatal("room created without token")
	}
	var packsOut []game.PackSummary
	api(t, ts, "GET", "/api/packs", "", nil, &packsOut)
	hasCricket := false
	for _, p := range packsOut {
		hasCricket = hasCricket || p.ID == "cricket-men-intl"
	}
	if len(packsOut) < 2 || !hasCricket {
		t.Fatalf("packs: %+v", packsOut)
	}

	var created struct {
		Code      string `json:"code"`
		InviteURL string `json:"invite_url"`
	}
	if api(t, ts, "POST", "/api/rooms", ga.Token, map[string]bool{"practice": false}, &created) != 201 {
		t.Fatal("create room")
	}
	if len(created.Code) != 6 || !strings.Contains(created.InviteURL, "/room/?code="+created.Code) {
		t.Fatalf("room: %+v", created)
	}

	a, _, err := dial(t, ts, created.Code, ga.Token)
	if err != nil {
		t.Fatal(err)
	}
	b, _, err := dial(t, ts, strings.ToLower(created.Code), gb.Token) // codes are case-insensitive
	if err != nil {
		t.Fatal(err)
	}
	a.next(room.EvState, `"name":"Bilal"`)

	// Third player, bad token and unknown room get distinct close codes.
	c3, _, err := dial(t, ts, created.Code, gc.Token)
	if err != nil {
		t.Fatal(err)
	}
	if code := closeCode(t, c3); code != closeRoomFull {
		t.Fatalf("third player close code %d", code)
	}
	cBad, _, _ := dial(t, ts, created.Code, "nope")
	if code := closeCode(t, cBad); code != closeUnauthorized {
		t.Fatalf("bad token close code %d", code)
	}
	cNo, _, _ := dial(t, ts, "ZZZZZZ", ga.Token)
	if code := closeCode(t, cNo); code != closeRoomNotFound {
		t.Fatalf("unknown room close code %d", code)
	}

	var info room.Info
	api(t, ts, "GET", "/api/rooms/"+created.Code, "", nil, &info)
	if !info.Full || info.Phase != room.PhaseLobby {
		t.Fatalf("info: %+v", info)
	}

	a.send(room.MsgSetConfig, room.Config{PackID: "cricket-men-intl", Difficulty: "easy"})
	b.next(room.EvState, `"difficulty":"easy"`) // settings changes un-ready the guest, so wait
	b.send(room.MsgReady, map[string]bool{"ready": true})
	a.next(room.EvState, `"ready":true`)
	a.send(room.MsgStart, nil)
	poolRaw := a.next(room.EvPool)
	b.next(room.EvPool)
	var pool struct {
		Data room.PoolView `json:"data"`
	}
	json.Unmarshal([]byte(poolRaw), &pool)
	if len(pool.Data.Candidates) != 24 {
		t.Fatalf("easy pool size %d", len(pool.Data.Candidates))
	}
	secretA, secretB := pool.Data.Candidates[0].ID, pool.Data.Candidates[1].ID
	a.send(room.MsgPickSecret, map[string]string{"candidate_id": secretA})
	b.send(room.MsgPickSecret, map[string]string{"candidate_id": secretB})
	a.next(room.EvState, `"phase":"hunting"`)

	raw, _ := json.Marshal("India")
	a.send(room.MsgAsk, map[string]any{"predicate": game.Predicate{Attr: "country", Op: "eq", Value: raw}})
	ans := a.next(room.EvAnswer)
	if !strings.Contains(ans, "Have they represented India?") {
		t.Fatalf("answer: %s", ans)
	}
	a.send(room.MsgGuess, map[string]string{"candidate_id": secretB})
	a.next(room.EvGuessResult, `"answer":"correct"`)
	b.next(room.EvState, `"status":"solved"`)
	b.send(room.MsgGiveUp, nil)
	res := a.next(room.EvState, `"phase":"results"`)
	if !strings.Contains(res, `"winner":0`) || !strings.Contains(res, secretA) {
		t.Fatalf("result: %s", res)
	}
}

func TestPracticeRoomOverWebSocket(t *testing.T) {
	ts := newTestServer(t)
	var g store.Guest
	api(t, ts, "POST", "/api/guest", "", map[string]string{"display_name": "Solo"}, &g)
	var created struct {
		Code string `json:"code"`
	}
	api(t, ts, "POST", "/api/rooms", g.Token, map[string]bool{"practice": true}, &created)
	a, _, err := dial(t, ts, created.Code, g.Token)
	if err != nil {
		t.Fatal(err)
	}
	a.next(room.EvState, `"bot":true`)
	a.send(room.MsgStart, nil)
	poolRaw := a.next(room.EvPool)
	var pool struct {
		Data room.PoolView `json:"data"`
	}
	json.Unmarshal([]byte(poolRaw), &pool)
	a.send(room.MsgPickSecret, map[string]string{"candidate_id": pool.Data.Candidates[3].ID})
	a.next(room.EvState, `"phase":"hunting"`)
	a.next(room.EvState, `"bot":true,"connected":true,"ready":true,"status":"solved"`)
}

func TestRoomCreationRateLimit(t *testing.T) {
	ts := newTestServer(t) // limit 3 per IP per hour
	var g store.Guest
	api(t, ts, "POST", "/api/guest", "", map[string]string{"display_name": "Spam"}, &g)
	for i := range 3 {
		if s := api(t, ts, "POST", "/api/rooms", g.Token, nil, nil); s != 201 {
			t.Fatalf("room %d: status %d", i, s)
		}
	}
	if s := api(t, ts, "POST", "/api/rooms", g.Token, nil, nil); s != 429 {
		t.Fatalf("4th room: status %d", s)
	}
}

func TestDailyOverHTTP(t *testing.T) {
	ts := newTestServer(t)
	var g store.Guest
	api(t, ts, "POST", "/api/guest", "", map[string]string{"display_name": "Daily"}, &g)
	if s := api(t, ts, "GET", "/api/daily", "", nil, nil); s != 401 {
		t.Fatalf("daily without token: %d", s)
	}
	var v daily.View
	if s := api(t, ts, "GET", "/api/daily", g.Token, nil, &v); s != 200 || v.Pool == nil || v.Result != nil {
		t.Fatalf("daily get: %d %+v", s, v.Status)
	}
	var m room.MatchView
	api(t, ts, "GET", "/api/daily/match?text=did+they+represent+India", "", nil, &m)
	if !m.Exact || m.Predicate == nil {
		t.Fatalf("match: %+v", m)
	}
	var e game.Error
	if s := api(t, ts, "POST", "/api/daily/play", g.Token, map[string]string{"text": "is he married"}, &e); s != 422 || e.Code != game.CodeNoMatch {
		t.Fatalf("unmatched: %d %+v", s, e)
	}
	// Guess everyone until correct; the secret is only in the final response.
	for _, c := range v.Pool.Candidates {
		var out daily.View
		s := api(t, ts, "POST", "/api/daily/play", g.Token, map[string]string{"candidate_id": c.ID}, &out)
		for s == 429 { // rate limited: back off like a client would
			time.Sleep(250 * time.Millisecond)
			s = api(t, ts, "POST", "/api/daily/play", g.Token, map[string]string{"candidate_id": c.ID}, &out)
		}
		if s != 200 {
			t.Fatalf("guess: %d", s)
		}
		if out.Status == game.StatusSolved {
			if out.Result == nil || out.Result.Secret.ID != c.ID {
				t.Fatal("no reveal after solving")
			}
			return
		}
		if out.Result != nil {
			t.Fatal("reveal before solving")
		}
	}
	t.Fatal("never solved")
}
