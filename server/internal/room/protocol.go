package room

import (
	"encoding/json"
	"time"

	"mindhunt/internal/game"
)

// ClientMsg is every message a client sends over the socket.
type ClientMsg struct {
	Type string          `json:"type"`
	Cseq int64           `json:"cseq"`
	Data json.RawMessage `json:"data"`
}

// ServerMsg is every message the server sends over the socket.
type ServerMsg struct {
	Type    string `json:"type"`
	Seq     int64  `json:"seq"`
	ReplyTo int64  `json:"reply_to,omitempty"`
	Data    any    `json:"data"`
}

// Client → server message types.
const (
	MsgSetConfig  = "set_config"
	MsgReady      = "ready"
	MsgStart      = "start"
	MsgPickSecret = "pick_secret"
	MsgAsk        = "ask"
	MsgGuess      = "guess"
	MsgGiveUp     = "give_up"
	MsgClaimWin   = "claim_win"
	MsgRematch    = "rematch"
	MsgToLobby    = "to_lobby"
	MsgLeave      = "leave"
	MsgPing       = "ping"
	MsgAskText    = "ask_text"    // typed or spoken question
	MsgAnswerText = "answer_text" // manual answer to the opponent's question
	MsgCancelText = "cancel_text" // withdraw your waiting question
	MsgMatchText  = "match_text"  // preview: which question does this text mean?
)

// Server → client message types.
const (
	EvState       = "state"
	EvPool        = "pool"
	EvAnswer      = "answer"
	EvGuessResult = "guess_result"
	EvError       = "error"
	EvPong        = "pong"
	EvKicked      = "kicked"
	EvAsked       = "asked" // your typed question is waiting for the opponent
	EvMatch       = "match" // reply to match_text
)

// Phase of a room.
type Phase string

const (
	PhaseLobby   Phase = "lobby"
	PhasePicking Phase = "picking"
	PhaseHunting Phase = "hunting"
	PhaseResults Phase = "results"
	PhaseVoid    Phase = "void"
)

// Config is what the host chooses in the lobby.
type Config struct {
	PackID     string `json:"pack_id"`
	Difficulty string `json:"difficulty"`
	TimerS     int    `json:"timer_s"`
	FreeText   bool   `json:"free_text"`
}

// PoolSizes maps difficulty to pool size; 0 means the whole pack.
var PoolSizes = map[string]int{"easy": 24, "normal": 48, "hard": 0}

// AllowedTimers are the selectable per-move timers in seconds (0 = off).
var AllowedTimers = map[int]bool{0: true, 30: true, 60: true, 90: true, 120: true}

// ConfigView is Config plus derived values.
type ConfigView struct {
	Config
	PackTitle      string `json:"pack_title"`
	PackVersion    int    `json:"pack_version"`
	PackDraft      bool   `json:"pack_draft"`
	PackAuthor     string `json:"pack_author,omitempty"`
	PoolSize       int    `json:"pool_size"`
	WrongGuessCost int    `json:"wrong_guess_cost"`
}

// PlayerView describes one seat.
type PlayerView struct {
	Seat      int    `json:"seat"`
	Name      string `json:"name"`
	Host      bool   `json:"host"`
	Bot       bool   `json:"bot"`
	Connected bool   `json:"connected"`
	Ready     bool   `json:"ready"`
	Status    string `json:"status,omitempty"`
	// ClaimableAt is set on the opponent while they are disconnected
	// mid-hunt: after this time the viewer may claim the win.
	ClaimableAt *time.Time `json:"claimable_at,omitempty"`
}

// HuntView is the viewer's own progress. It never contains anything about
// the opponent's secret or questions.
type HuntView struct {
	YourSecretID string      `json:"your_secret_id,omitempty"`
	YourStatus   string      `json:"your_status"`
	YourMoves    int         `json:"your_moves"`
	YourLog      []game.Move `json:"your_log"`
	RemainingIDs []string    `json:"remaining_ids"`
	// Pending is your typed question waiting for the opponent to answer.
	Pending *game.Pending `json:"pending,omitempty"`
	// Incoming is the opponent's typed question you must answer.
	Incoming     *game.Pending `json:"incoming,omitempty"`
	UnknownsLeft int           `json:"unknowns_left"`
	Deadline     *time.Time    `json:"deadline,omitempty"`
}

// StateView is the full per-player snapshot sent after every change.
type StateView struct {
	Room       string      `json:"room"`
	Phase      Phase       `json:"phase"`
	Practice   bool        `json:"practice"`
	Round      int         `json:"round"`
	You        PlayerView  `json:"you"`
	Opponent   *PlayerView `json:"opponent"`
	Config     ConfigView  `json:"config"`
	Hunt       *HuntView   `json:"hunt,omitempty"`
	Result     *ResultView `json:"result,omitempty"`
	RematchYou bool        `json:"rematch_you"`
	RematchOpp bool        `json:"rematch_opponent"`
	ServerTime time.Time   `json:"server_time"`
	GraceS     int         `json:"grace_s"`
}

// AttrView is an attribute as shown in the question builder.
type AttrView struct {
	Key    string        `json:"key"`
	Label  string        `json:"label"`
	Type   game.AttrType `json:"type"`
	Values []string      `json:"values,omitempty"`
	Min    *int          `json:"min,omitempty"`
	Max    *int          `json:"max,omitempty"`
	Step   int           `json:"step,omitempty"`
	Help   string        `json:"help,omitempty"`
	Ops    []string      `json:"ops"`
}

// PoolCandidate is a candidate as shown during a round: no image reference.
type PoolCandidate struct {
	ID    string            `json:"id"`
	Name  string            `json:"name"`
	Attrs map[string]string `json:"attrs"`
}

// PoolView is sent once per round to both players.
type PoolView struct {
	Round      int             `json:"round"`
	PackID     string          `json:"pack_id"`
	PackTitle  string          `json:"pack_title"`
	Version    int             `json:"version"`
	AsOf       string          `json:"as_of"`
	Attributes []AttrView      `json:"attributes"`
	Candidates []PoolCandidate `json:"candidates"`
}

// CandidateCard is a revealed secret on the results screen.
type CandidateCard struct {
	ID    string     `json:"id"`
	Name  string     `json:"name"`
	Attrs []AttrLine `json:"attrs"`
	Facts []string   `json:"facts,omitempty"`
	Image *ImageView `json:"image,omitempty"`
}

// AttrLine is a label/value pair.
type AttrLine struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

// ImageView is a licensed image, sent only in results.
type ImageView struct {
	URL        string `json:"url"`
	License    string `json:"license"`
	LicenseURL string `json:"license_url,omitempty"`
	Credit     string `json:"credit"`
	SourceURL  string `json:"source_url"`
}

// ResultPlayer is one seat's revealed round.
type ResultPlayer struct {
	Seat      int           `json:"seat"`
	Name      string        `json:"name"`
	Bot       bool          `json:"bot"`
	Secret    CandidateCard `json:"secret"`
	Status    string        `json:"status"`
	Moves     int           `json:"moves"`
	Score     int           `json:"score"`
	ActiveMs  int64         `json:"active_ms"`
	Log       []game.Move   `json:"log"`
	CrackedBy *game.Move    `json:"cracked_by,omitempty"`
	Grid      string        `json:"grid"`
	// AnswerFlags are manual answers this player gave that the audit found
	// contradict the data about their own secret.
	AnswerFlags []game.Flag `json:"answer_flags,omitempty"`
}

// MatchView answers match_text. Predicate is nil when nothing matched.
// Exact means asking the text will be answered from the data as is.
type MatchView struct {
	Text      string          `json:"text,omitempty"`
	Predicate *game.Predicate `json:"predicate,omitempty"`
	Exact     bool            `json:"exact"`
}

// ResultView is sent to both players only after the round ends.
type ResultView struct {
	Winner  int             `json:"winner"`
	Reason  string          `json:"reason"`
	Players [2]ResultPlayer `json:"players"`
	EndedAt time.Time       `json:"ended_at"`
}
