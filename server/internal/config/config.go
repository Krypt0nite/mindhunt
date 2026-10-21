// Package config reads server settings from environment variables.
package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config is the server configuration.
type Config struct {
	Addr           string
	AppName        string
	DatabaseURL    string
	WebDir         string
	AssetsDir      string
	PacksDir       string
	PublicURL      string
	AllowedOrigins []string
	WrongGuessCost int
	ReconnectGrace time.Duration
	RoomsPerIPHour int

	DefaultPack string
	DailyKey    string
	DailyTZ     string
	DailyPack   string
	DailyEpoch  string
	DailyPool   int

	AdminToken   string
	BlockedWords string
}

// Load reads the environment, applying defaults.
func Load() Config {
	return Config{
		Addr:           ":" + env("PORT", "8080"),
		AppName:        env("APP_NAME", "MindHunt"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		WebDir:         env("WEB_DIR", "../web/out"),
		AssetsDir:      env("ASSETS_DIR", "../assets"),
		PacksDir:       os.Getenv("PACKS_DIR"),
		PublicURL:      strings.TrimRight(os.Getenv("PUBLIC_URL"), "/"),
		AllowedOrigins: list(os.Getenv("ALLOWED_ORIGINS")),
		WrongGuessCost: envInt("WRONG_GUESS_COST", 3),
		ReconnectGrace: time.Duration(envInt("RECONNECT_GRACE_S", 90)) * time.Second,
		RoomsPerIPHour: envInt("ROOMS_PER_IP_HOUR", 20),
		DefaultPack:    env("DEFAULT_PACK", "cricket-men-intl"),
		DailyKey:       os.Getenv("DAILY_KEY"),
		DailyTZ:        env("DAILY_TZ", "Asia/Kolkata"),
		DailyPack:      env("DAILY_PACK", "rotate"),
		DailyEpoch:     env("DAILY_EPOCH", "2026-10-18"),
		DailyPool:      envInt("DAILY_POOL", 48),
		AdminToken:     os.Getenv("ADMIN_TOKEN"),
		BlockedWords:   os.Getenv("BLOCKED_WORDS"),
	}
}

func env(k, def string) string {
	if v := strings.TrimSpace(os.Getenv(k)); v != "" {
		return v
	}
	return def
}

func envInt(k string, def int) int {
	if v, err := strconv.Atoi(os.Getenv(k)); err == nil && v >= 0 {
		return v
	}
	return def
}

func list(s string) []string {
	var out []string
	for _, p := range strings.Split(s, ",") {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}
