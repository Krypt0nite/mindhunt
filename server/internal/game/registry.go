package game

import (
	"fmt"
	"io/fs"
	"path"
	"sort"
)

// Registry holds every loaded pack, keyed by id. Only the newest version of
// each id is offered for new matches; a running match keeps its own pointer.
type Registry struct {
	packs       map[string]*Pack
	defaultPack string
}

// PackSummary is the public listing of a pack (no candidates).
type PackSummary struct {
	ID          string `json:"id"`
	Version     int    `json:"version"`
	AsOf        string `json:"as_of"`
	Category    string `json:"category"`
	Title       string `json:"title"`
	Noun        string `json:"noun"`
	Description string `json:"description"`
	Draft       bool   `json:"draft"`
	Candidates  int    `json:"candidates"`
}

// LoadRegistry parses every *.json file in fsys.
func LoadRegistry(fsys fs.FS) (*Registry, error) {
	r := &Registry{packs: map[string]*Pack{}}
	entries, err := fs.ReadDir(fsys, ".")
	if err != nil {
		return nil, err
	}
	for _, e := range entries {
		if e.IsDir() || path.Ext(e.Name()) != ".json" {
			continue
		}
		data, err := fs.ReadFile(fsys, e.Name())
		if err != nil {
			return nil, err
		}
		p, err := ParsePack(data)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", e.Name(), err)
		}
		if old, ok := r.packs[p.ID]; !ok || p.Version > old.Version {
			r.packs[p.ID] = p
		}
	}
	if len(r.packs) == 0 {
		return nil, fmt.Errorf("no packs found")
	}
	return r, nil
}

// DropImages applies Pack.DropImages to every pack and returns, per pack id,
// how many image references were removed.
func (r *Registry) DropImages(keep func(asset string) bool) map[string]int {
	out := map[string]int{}
	for id, p := range r.packs {
		if n := p.DropImages(keep); n > 0 {
			out[id] = n
		}
	}
	return out
}

// Packs resolves a pack id for a room: a built-in pack or a player-made one.
type Packs interface {
	Get(id string) (*Pack, bool)
	Default() *Pack
}

// Catalog combines the built-in registry with a lookup for player-made packs
// (which must already be in memory: rooms never wait on the database).
type Catalog struct {
	*Registry
	Custom func(id string) (*Pack, bool)
}

// Get finds a built-in pack first, then a player-made one.
func (c Catalog) Get(id string) (*Pack, bool) {
	if p, ok := c.Registry.Get(id); ok {
		return p, true
	}
	if c.Custom != nil {
		return c.Custom(id)
	}
	return nil, false
}

// Get returns the newest version of a pack.
func (r *Registry) Get(id string) (*Pack, bool) {
	p, ok := r.packs[id]
	return p, ok
}

// Default returns the pack new rooms start with: the one set with
// SetDefault, else the first by id.
func (r *Registry) Default() *Pack {
	if p, ok := r.packs[r.defaultPack]; ok {
		return p
	}
	return r.Get2(r.ids()[0])
}

// SetDefault chooses the pack new rooms start with.
func (r *Registry) SetDefault(id string) error {
	if _, ok := r.packs[id]; !ok {
		return fmt.Errorf("unknown default pack %q", id)
	}
	r.defaultPack = id
	return nil
}

// IDs lists pack ids in sorted order.
func (r *Registry) IDs() []string { return r.ids() }

// Get2 is Get without the ok flag, for ids known to exist.
func (r *Registry) Get2(id string) *Pack { return r.packs[id] }

func (r *Registry) ids() []string {
	ids := make([]string, 0, len(r.packs))
	for id := range r.packs {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	return ids
}

// Summaries lists the packs for the pack picker.
func (r *Registry) Summaries() []PackSummary {
	var out []PackSummary
	for _, id := range r.ids() {
		p := r.packs[id]
		out = append(out, PackSummary{
			ID: p.ID, Version: p.Version, AsOf: p.AsOf, Category: p.Category,
			Title: p.Title, Noun: p.Noun, Description: p.Description, Draft: p.Draft, Candidates: len(p.Candidates),
		})
	}
	return out
}
