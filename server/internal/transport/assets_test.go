package transport

import (
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"mindhunt/internal/config"
	"mindhunt/internal/store"
)

func TestAssetsServeOnlyImages(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "cricket-men-intl"), 0o755); err != nil {
		t.Fatal(err)
	}
	webp := []byte("RIFF\x1a\x00\x00\x00WEBPVP8 ")
	os.WriteFile(filepath.Join(dir, "cricket-men-intl", "c_44c51f.webp"), webp, 0o644)
	os.WriteFile(filepath.Join(dir, "cricket-men-intl", "notes.txt"), []byte("secret"), 0o644)
	os.WriteFile(filepath.Join(dir, "top.webp"), webp, 0o644)

	s := &Server{cfg: config.Config{AssetsDir: dir}, guests: store.NewMemoryGuests()}
	ts := httptest.NewServer(s.assets())
	defer ts.Close()

	get := func(p string) (int, http.Header, string) {
		res, err := http.Get(ts.URL + p)
		if err != nil {
			t.Fatal(err)
		}
		defer res.Body.Close()
		b, _ := io.ReadAll(res.Body)
		return res.StatusCode, res.Header, string(b)
	}
	code, h, body := get("/assets/cricket-men-intl/c_44c51f.webp")
	if code != 200 || body != string(webp) || h.Get("Cache-Control") == "" || h.Get("X-Content-Type-Options") != "nosniff" {
		t.Fatalf("image: %d %v", code, h)
	}
	for _, p := range []string{
		"/assets/cricket-men-intl/notes.txt",     // not an image
		"/assets/cricket-men-intl/",              // no listings
		"/assets/cricket-men-intl/c_000000.webp", // missing: client shows initials
		"/assets/../go.mod",                      // traversal
		"/assets/cricket-men-intl/..%2f..%2fgo.mod",
		"/assets/top.webp", // must sit in a pack folder
	} {
		if code, _, _ := get(p); code != 404 {
			t.Errorf("%s: status %d, want 404", p, code)
		}
	}
}
