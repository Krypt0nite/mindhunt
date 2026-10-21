//go:build !js

package store

// The Postgres driver is left out of the WebAssembly build (Cloudflare),
// which keeps its data in Durable Object storage instead.
import _ "github.com/lib/pq"
