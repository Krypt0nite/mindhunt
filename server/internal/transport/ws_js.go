//go:build js

package transport

import "net/http"

// In the WebAssembly build the Cloudflare Worker owns the sockets and calls
// JoinSocket directly; /ws never reaches the Go handler.
func (s *Server) ws(w http.ResponseWriter, _ *http.Request) {
	writeErr(w, http.StatusNotFound, "not_found", "no such endpoint")
}
