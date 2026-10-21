// Package packs embeds the built-in topic packs.
package packs

import "embed"

// FS holds every *.json pack shipped with the server.
//
//go:embed *.json
var FS embed.FS
