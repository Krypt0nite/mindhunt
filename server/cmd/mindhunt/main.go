// Command mindhunt runs the MindHunt game server: REST API, WebSocket
// rooms and the static web client, in one binary.
package main

import (
	"context"
	"errors"
	"io/fs"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"
	"time"
	_ "time/tzdata" // the daily time zone must work in minimal containers

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
	if err := run(); err != nil {
		slog.Error("fatal", "err", err)
		os.Exit(1)
	}
}

func run() error {
	cfg := config.Load()

	var packFS fs.FS = packs.FS
	if cfg.PacksDir != "" {
		packFS = os.DirFS(cfg.PacksDir)
	}
	reg, err := game.LoadRegistry(packFS)
	if err != nil {
		return err
	}
	if err := reg.SetDefault(cfg.DefaultPack); err != nil {
		return err
	}
	for _, p := range reg.Summaries() {
		slog.Info("pack loaded", "id", p.ID, "version", p.Version, "candidates", p.Candidates, "draft", p.Draft)
	}

	var (
		guests  store.Guests      = store.NewMemoryGuests()
		rec     store.Recorder    = store.Noop{}
		days    store.Daily       = store.NewMemoryDaily()
		packsDB store.CustomPacks = store.NewMemoryCustomPacks()
		pg      *store.Postgres
	)
	if cfg.DatabaseURL != "" {
		pg, err = store.OpenPostgres(context.Background(), cfg.DatabaseURL)
		if err != nil {
			return err
		}
		guests, rec, days, packsDB = pg, pg, pg, pg
		slog.Info("persistence: postgres")
	} else {
		slog.Warn("persistence: memory only (set DATABASE_URL to keep guests and results)")
	}

	// A pack may reference an image that was not shipped with this build;
	// drop it so the reveal shows initials rather than a broken picture.
	for id, n := range reg.DropImages(func(asset string) bool {
		st, err := os.Stat(filepath.Join(cfg.AssetsDir, filepath.FromSlash(asset)))
		return transport.AssetPathOK(asset) && err == nil && !st.IsDir()
	}) {
		slog.Warn("images missing from ASSETS_DIR; using initials", "pack", id, "count", n, "assets_dir", cfg.AssetsDir)
	}

	set := room.DefaultSettings()
	set.Rules.WrongGuessCost = cfg.WrongGuessCost
	set.ReconnectGrace = cfg.ReconnectGrace
	custom.AddBlockedWords(cfg.BlockedWords)
	lib := custom.NewLibrary(packsDB, custom.Config{})
	mgr := room.NewManager(game.Catalog{Registry: reg, Custom: lib.Cached}, set, rec)
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
	}, reg, days)
	if err != nil {
		return err
	}
	srv := transport.New(cfg, guests, mgr, reg, dailySvc, lib)

	httpSrv := &http.Server{
		Addr:              cfg.Addr,
		Handler:           srv.Handler(),
		ReadHeaderTimeout: 10 * time.Second,
	}
	errc := make(chan error, 1)
	go func() {
		slog.Info("listening", "addr", cfg.Addr, "app", cfg.AppName, "web_dir", cfg.WebDir)
		errc <- httpSrv.ListenAndServe()
	}()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	select {
	case err := <-errc:
		if !errors.Is(err, http.ErrServerClosed) {
			return err
		}
	case <-stop:
		slog.Info("shutting down")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = httpSrv.Shutdown(ctx)
	if pg != nil {
		pg.Close(5 * time.Second)
	}
	return nil
}
