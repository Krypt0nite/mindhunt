package game

import "fmt"

// Error codes shared with the client protocol.
const (
	CodeWrongPhase        = "wrong_phase"
	CodeNotYourRoom       = "not_your_room"
	CodeInvalidPredicate  = "invalid_predicate"
	CodeForbiddenAttr     = "forbidden_attr"
	CodeDuplicateQuestion = "duplicate_question"
	CodeDuplicateGuess    = "duplicate_guess"
	CodeNotInPool         = "not_in_pool"
	CodeAlreadyDone       = "already_solved"
	CodeRateLimited       = "rate_limited"
	CodeNotHost           = "not_host"
	CodeNotReady          = "not_ready"
	CodeRoomFull          = "room_full"
	CodeBadRequest        = "bad_request"
	CodeCannotClaim       = "cannot_claim"
	CodeInvalidText       = "invalid_text"
	CodeNoMatch           = "no_match"
	CodePendingQuestion   = "pending_question"
	CodeNoPending         = "no_pending"
	CodeUnknownCap        = "unknown_cap"
	CodeFreeTextOff       = "free_text_off"
)

// Error is a rule violation reported to the player who caused it.
type Error struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

func (e *Error) Error() string { return e.Code + ": " + e.Message }

// Errf builds a game error.
func Errf(code, format string, args ...any) *Error {
	return &Error{Code: code, Message: fmt.Sprintf(format, args...)}
}
