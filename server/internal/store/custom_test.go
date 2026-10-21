package store

import (
	"context"
	"errors"
	"os"
	"testing"
	"time"
)

// The same behaviour from both custom pack stores.
func TestCustomPackStores(t *testing.T) {
	stores := map[string]CustomPacks{"memory": NewMemoryCustomPacks()}
	if url := os.Getenv("MINDHUNT_TEST_DATABASE_URL"); url != "" {
		p, err := OpenPostgres(context.Background(), url)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := p.db.Exec(`TRUNCATE custom_packs, custom_pack_reports`); err != nil {
			t.Fatal(err)
		}
		stores["postgres"] = p
	}
	for name, st := range stores {
		t.Run(name, func(t *testing.T) { exerciseCustom(t, st) })
	}
}

func exerciseCustom(t *testing.T, st CustomPacks) {
	ctx := context.Background()
	now := time.Now().UTC().Truncate(time.Millisecond)
	cp := CustomPack{ID: "u-testpack01", OwnerID: "p_a", Author: "Asha", Title: "Fruit", Version: 1,
		Draft: []byte(`{"title":"Fruit"}`), Pack: []byte(`{"id":"u-testpack01"}`), CreatedAt: now, UpdatedAt: now}
	if err := st.CreatePack(ctx, cp); err != nil {
		t.Fatal(err)
	}
	got, err := st.GetPack(ctx, cp.ID)
	if err != nil || got.Title != "Fruit" || got.OwnerID != "p_a" || string(got.Pack) != `{"id": "u-testpack01"}` && string(got.Pack) != `{"id":"u-testpack01"}` {
		t.Fatalf("get %+v %v", got, err)
	}
	if _, err := st.GetPack(ctx, "u-missing000"); !errors.Is(err, ErrNotFound) {
		t.Fatalf("missing: %v", err)
	}
	cp.Version, cp.Title = 2, "Fruit bowl"
	if err := st.UpdatePack(ctx, cp); err != nil {
		t.Fatal(err)
	}
	stranger := cp
	stranger.OwnerID = "p_b"
	if err := st.UpdatePack(ctx, stranger); !errors.Is(err, ErrNotOwner) {
		t.Fatalf("stranger update: %v", err)
	}
	mine, _ := st.PacksByOwner(ctx, "p_a")
	if len(mine) != 1 || mine[0].Version != 2 || mine[0].Title != "Fruit bowl" {
		t.Fatalf("mine %+v", mine)
	}
	if n, _ := st.CountRecentPacks(ctx, "p_a", now.Add(-time.Minute)); n != 1 {
		t.Fatalf("recent %d", n)
	}
	n1, _ := st.ReportPack(ctx, Report{PackID: cp.ID, ReporterID: "p_r1", Reason: "rude", At: now})
	n2, _ := st.ReportPack(ctx, Report{PackID: cp.ID, ReporterID: "p_r1", Reason: "still rude", At: now})
	n3, _ := st.ReportPack(ctx, Report{PackID: cp.ID, ReporterID: "p_r2", Reason: "rude", At: now})
	if n1 != 1 || n2 != 1 || n3 != 2 {
		t.Fatalf("report counts %d %d %d", n1, n2, n3)
	}
	if _, err := st.ReportPack(ctx, Report{PackID: "u-missing000", ReporterID: "p_r1", Reason: "x", At: now}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("report missing: %v", err)
	}
	packs, reps, err := st.ReportedPacks(ctx, 10)
	if err != nil || len(packs) != 1 || packs[0].Reports != 2 || len(reps) != 2 {
		t.Fatalf("reported %+v %+v %v", packs, reps, err)
	}
	if err := st.SetPackFlags(ctx, cp.ID, true, true); err != nil {
		t.Fatal(err)
	}
	got, _ = st.GetPack(ctx, cp.ID)
	if !got.Hidden || !got.Deleted {
		t.Fatal("flags not saved")
	}
	if mine, _ := st.PacksByOwner(ctx, "p_a"); len(mine) != 0 {
		t.Fatal("deleted pack listed")
	}
}
