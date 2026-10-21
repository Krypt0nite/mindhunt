package room

import (
	"time"

	"mindhunt/internal/game"
)

// viewFor builds the complete snapshot for one seat. Every state message
// goes through here, so this is the single place where privacy is decided:
// the opponent's secret, questions, move count and image are never read
// before the results phase.
func (r *Room) viewFor(i int) StateView {
	v := StateView{
		Room: r.Code, Phase: r.phase, Practice: r.practice, Round: r.round,
		You: r.playerView(i, true), Config: r.configView(),
		RematchYou: r.rematch[i], RematchOpp: r.rematch[1-i],
		ServerTime: r.now().UTC(), GraceS: int(r.set.ReconnectGrace.Seconds()),
	}
	if r.seats[1-i] != nil {
		o := r.playerView(1-i, false)
		v.Opponent = &o
	}
	if r.match != nil && r.phase != PhaseLobby {
		v.Hunt = r.huntView(i)
	}
	if r.phase == PhaseResults {
		v.Result = r.result
	}
	return v
}

func (r *Room) playerView(i int, self bool) PlayerView {
	s := r.seats[i]
	pv := PlayerView{Seat: i, Name: s.name, Host: i == r.host, Bot: s.isBot, Connected: s.connected || s.isBot, Ready: s.ready}
	if r.match != nil && r.phase != PhaseLobby {
		pv.Status = string(r.match.Status(i))
	}
	if !self && r.phase == PhaseHunting && !s.connected && !s.isBot {
		t := s.disconnectedAt.Add(r.set.ReconnectGrace).UTC()
		pv.ClaimableAt = &t
	}
	return pv
}

func (r *Room) configView() ConfigView {
	cv := ConfigView{Config: r.cfg, WrongGuessCost: r.set.Rules.WrongGuessCost}
	if p := r.pack; p != nil {
		cv.PackAuthor = p.Author
		cv.PackTitle, cv.PackVersion, cv.PackDraft = p.Title, p.Version, p.Draft
		cv.PoolSize = PoolSizes[r.cfg.Difficulty]
		if cv.PoolSize == 0 || cv.PoolSize > len(p.Candidates) {
			cv.PoolSize = len(p.Candidates)
		}
	}
	if r.match != nil {
		cv.PoolSize = len(r.match.Pool)
		cv.PackVersion = r.match.Pack.Version
	}
	return cv
}

func (r *Room) huntView(i int) *HuntView {
	m := r.match
	h := &HuntView{YourStatus: string(m.Status(i)), YourMoves: m.Moves(i), YourLog: m.Log(i)}
	if h.YourLog == nil {
		h.YourLog = []game.Move{}
	}
	if sec := m.Secret(i); sec != nil {
		h.YourSecretID = sec.ID
	}
	if r.phase == PhasePicking {
		h.RemainingIDs = []string{}
	} else {
		for _, c := range m.Remaining(i) {
			h.RemainingIDs = append(h.RemainingIDs, c.ID)
		}
	}
	if r.phase == PhaseHunting {
		h.Pending = m.Pending(i)
		h.Incoming = m.Incoming(i)
		h.UnknownsLeft = m.UnknownsLeft(i)
	}
	if d := r.seats[i].deadline; !d.IsZero() {
		t := d.UTC()
		h.Deadline = &t
	}
	return h
}

func (r *Room) poolView() PoolView { return PoolViewOf(r.match, r.round) }

// PoolViewOf lists a match's candidates and attributes for the client.
// It never includes image references.
func PoolViewOf(m *game.Match, round int) PoolView {
	pv := PoolView{Round: round, PackID: m.Pack.ID, PackTitle: m.Pack.Title, Version: m.Pack.Version, AsOf: m.Pack.AsOf}
	for _, a := range m.Pack.Attributes {
		pv.Attributes = append(pv.Attributes, AttrView{
			Key: a.Key, Label: a.Label, Type: a.Type, Values: a.Values,
			Min: a.Min, Max: a.Max, Step: a.Step, Help: a.Help, Ops: game.OpsFor(a.Type),
		})
	}
	for _, c := range m.Pool {
		attrs := make(map[string]string, len(m.Pack.Attributes))
		for _, a := range m.Pack.Attributes {
			if s := c.AttrString(&a); s != "" {
				attrs[a.Key] = s
			}
		}
		// Deliberately no image reference: an asset id or file name seen
		// before the reveal could identify the opponent's secret.
		pv.Candidates = append(pv.Candidates, PoolCandidate{ID: c.ID, Name: c.Name, Attrs: attrs})
	}
	return pv
}

func (r *Room) card(c *game.Candidate) CandidateCard {
	return CardOf(r.match.Pack, c, r.set.ImageBaseURL)
}

// CardOf builds the reveal card for a candidate.
func CardOf(p *game.Pack, c *game.Candidate, imageBase string) CandidateCard {
	card := CandidateCard{ID: c.ID, Name: c.Name, Facts: c.Facts}
	for _, a := range p.Attributes {
		if s := c.AttrString(&a); s != "" {
			card.Attrs = append(card.Attrs, AttrLine{Label: a.Label, Value: s})
		}
	}
	if c.Image != nil {
		card.Image = &ImageView{
			URL: imageBase + "/" + c.Image.Asset, License: c.Image.License, LicenseURL: c.Image.LicenseURL,
			Credit: c.Image.Credit, SourceURL: c.Image.SourceURL,
		}
	}
	return card
}

func (r *Room) resultView(res game.Result, now time.Time) *ResultView {
	m := r.match
	rv := &ResultView{Winner: res.Winner, Reason: res.Reason, EndedAt: now.UTC()}
	for i, s := range r.seats {
		rv.Players[i] = ResultPlayer{
			Seat: i, Name: s.name, Bot: s.isBot, Secret: r.card(m.Secret(i)),
			Status: string(m.Status(i)), Moves: m.Moves(i), Score: res.Scores[i],
			ActiveMs: s.activeMs, Log: m.Log(i), CrackedBy: m.CrackedBy(i), Grid: m.ShareGrid(i),
			AnswerFlags: m.Audit(i),
		}
	}
	return rv
}
