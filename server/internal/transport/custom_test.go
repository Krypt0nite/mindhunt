package transport

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"mindhunt/internal/custom"
	"mindhunt/internal/room"
	"mindhunt/internal/store"
)

func fruitDraft() custom.Draft {
	d := custom.Draft{Title: "Fruit bowl", Noun: "fruit", Attributes: []custom.DraftAttr{
		{Label: "Colour", Kind: custom.KindChoice, Choices: []string{"Red", "Yellow", "Green", "Orange"}, Question: "Is it {value}?"},
		{Label: "Grows on trees", Kind: custom.KindYesNo, Question: "Does it grow on trees?"},
		{Label: "Seeds", Kind: custom.KindNumber, Question: "Does it have {value} seeds?"},
	}}
	for _, r := range []string{"Apple|Red|yes|5", "Banana|Yellow|no|0", "Cherry|Red|yes|1", "Lime|Green|yes|0",
		"Mango|Orange|yes|1", "Orange|Orange|yes|10", "Pear|Green|yes|6", "Kiwi|Green|no|99", "Lemon|Yellow|yes|8"} {
		p := strings.Split(r, "|")
		d.Items = append(d.Items, custom.DraftItem{Name: p[0], Values: p[1:]})
	}
	return d
}

func TestCustomPackOverHTTP(t *testing.T) {
	ts := newTestServer(t)
	var owner, friend store.Guest
	api(t, ts, "POST", "/api/guest", "", map[string]string{"display_name": "Asha"}, &owner)
	api(t, ts, "POST", "/api/guest", "", map[string]string{"display_name": "Bilal"}, &friend)

	// Live check reports problems without saving.
	bad := fruitDraft()
	bad.Items = bad.Items[:3]
	var check struct {
		OK     bool           `json:"ok"`
		Issues []custom.Issue `json:"issues"`
	}
	if s := api(t, ts, "POST", "/api/custom/check", owner.Token, map[string]any{"draft": bad}, &check); s != 200 || check.OK {
		t.Fatalf("check: %d %+v", s, check)
	}
	if s := api(t, ts, "POST", "/api/custom/check", "", map[string]any{"draft": bad}, nil); s != 401 {
		t.Fatalf("anonymous check: %d", s)
	}
	var rej struct {
		Code   string         `json:"code"`
		Issues []custom.Issue `json:"issues"`
	}
	if s := api(t, ts, "POST", "/api/custom", owner.Token, map[string]any{"draft": bad}, &rej); s != 422 || len(rej.Issues) == 0 {
		t.Fatalf("invalid pack saved: %d %+v", s, rej)
	}

	var made struct {
		ID       string `json:"id"`
		Version  int    `json:"version"`
		ShareURL string `json:"share_url"`
	}
	if s := api(t, ts, "POST", "/api/custom", owner.Token, map[string]any{"draft": fruitDraft()}, &made); s != 201 {
		t.Fatalf("create: %d", s)
	}
	if !custom.IsCustomID(made.ID) || !strings.HasSuffix(made.ShareURL, "/pack/?id="+made.ID) {
		t.Fatalf("made %+v", made)
	}

	// Anyone with the link sees the summary; only the owner gets the draft.
	var view CustomView
	api(t, ts, "GET", "/api/custom/"+made.ID, friend.Token, nil, &view)
	if view.Title != "Fruit bowl" || view.Author != "Asha" || view.Candidates != 9 || view.IsOwner || view.EditDraft != nil {
		t.Fatalf("friend view %+v", view)
	}
	api(t, ts, "GET", "/api/custom/"+made.ID, owner.Token, nil, &view)
	if !view.IsOwner || view.EditDraft == nil {
		t.Fatal("owner can't edit")
	}
	if s := api(t, ts, "PUT", "/api/custom/"+made.ID, friend.Token, map[string]any{"draft": fruitDraft()}, nil); s != 403 {
		t.Fatalf("friend edited: %d", s)
	}

	// The built-in listing never includes player-made packs.
	var list []map[string]any
	api(t, ts, "GET", "/api/packs", "", nil, &list)
	for _, p := range list {
		if p["id"] == made.ID {
			t.Fatal("custom pack listed publicly")
		}
	}

	// A practice room on the pack: the bot plays it to the end.
	var room1 struct {
		Code string `json:"code"`
	}
	if s := api(t, ts, "POST", "/api/rooms", friend.Token, map[string]any{"practice": true, "pack_id": made.ID}, &room1); s != 201 {
		t.Fatalf("room: %d", s)
	}
	a, _, err := dial(t, ts, room1.Code, friend.Token)
	if err != nil {
		t.Fatal(err)
	}
	a.next(room.EvState, `"pack_id":"`+made.ID+`"`, `"pack_author":"Asha"`)
	a.send(room.MsgStart, nil)
	var pool struct {
		Data room.PoolView `json:"data"`
	}
	json.Unmarshal([]byte(a.next(room.EvPool, "Fruit bowl")), &pool)
	if len(pool.Data.Candidates) != 9 {
		t.Fatalf("pool %d", len(pool.Data.Candidates))
	}
	a.send(room.MsgPickSecret, map[string]string{"candidate_id": pool.Data.Candidates[2].ID})
	a.next(room.EvState, `"phase":"hunting"`)
	a.send(room.MsgAsk, map[string]any{"predicate": map[string]any{"attr": "a1", "op": "eq", "value": "Red"}})
	a.next(room.EvState, "Is it Red?")
	a.next(room.EvState, `"bot":true,"connected":true,"ready":true,"status":"solved"`)

	// The owner edits: a new version, the same link.
	edit := fruitDraft()
	edit.Items = append(edit.Items, custom.DraftItem{Name: "Grape", Values: []string{"Green", "no", "0"}})
	if s := api(t, ts, "PUT", "/api/custom/"+made.ID, owner.Token, map[string]any{"draft": edit}, &made); s != 200 || made.Version != 2 {
		t.Fatalf("edit: %d %+v", s, made)
	}
	var mine []store.CustomPack
	api(t, ts, "GET", "/api/custom", owner.Token, nil, &mine)
	if len(mine) != 1 || mine[0].Version != 2 {
		t.Fatalf("mine %+v", mine)
	}

	// Reports: three different players hide it.
	if s := api(t, ts, "POST", "/api/custom/"+made.ID+"/report", owner.Token, map[string]string{"reason": "x"}, nil); s != 400 {
		t.Fatalf("self report: %d", s)
	}
	for i := range 3 {
		var g store.Guest
		api(t, ts, "POST", "/api/guest", "", map[string]string{"display_name": fmt.Sprint("R", i)}, &g)
		api(t, ts, "POST", "/api/custom/"+made.ID+"/report", g.Token, map[string]string{"reason": "offensive"}, nil)
	}
	if s := api(t, ts, "GET", "/api/custom/"+made.ID, friend.Token, nil, nil); s != 404 {
		t.Fatalf("hidden pack still visible: %d", s)
	}
	if s := api(t, ts, "POST", "/api/rooms", friend.Token, map[string]any{"pack_id": made.ID}, nil); s != 404 {
		t.Fatalf("room on hidden pack: %d", s)
	}

	// Admin endpoints are off without ADMIN_TOKEN.
	if s := api(t, ts, "GET", "/api/admin/reports", "", nil, nil); s != 404 {
		t.Fatalf("admin open: %d", s)
	}

	// Owner deletes.
	if s := api(t, ts, "DELETE", "/api/custom/"+made.ID, owner.Token, nil, nil); s != http.StatusNoContent {
		t.Fatalf("delete: %d", s)
	}
	if s := api(t, ts, "GET", "/api/custom/"+made.ID, owner.Token, nil, nil); s != 404 {
		t.Fatalf("deleted pack: %d", s)
	}
}
