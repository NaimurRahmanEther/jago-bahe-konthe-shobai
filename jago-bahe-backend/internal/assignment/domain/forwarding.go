package domain

import (
	"time"

	"jago-bahe-backend/pkg/idgen"
)

// Suggestion is one union admin's advice on where an above-union report should be
// forwarded (B20).
//
// It is ADVICE, not a ballot. It settles nothing, has no quorum and no window, and
// binds nobody: the super admin decides above-union forwarding and may act at any
// count, including none (the same "signal, not a gate" rule B17 established for V).
// What it buys is that the decision is made in the open — the tally is public, so a
// super admin who forwards against the seat's admins does so visibly.
//
// An admin may change their advice; one row per admin per problem is the rule, and
// a change replaces it. That is the deliberate difference from the vote this
// replaced, which refused a second ballot: a ballot settled something, so changing
// it would have rewritten a result, while advice that settles nothing should be
// revisable when the other advice teaches you something.
type Suggestion struct {
	ID                  string
	ProblemID           string
	AdminAccountID      string
	SuggestedOfficialID string
	Reason              string // optional; the admin's own public note on why
	CreatedAt           time.Time
}

// NewSuggestion constructs an admin's advice.
func NewSuggestion(problemID, adminAccountID, suggestedOfficialID, reason string) *Suggestion {
	return &Suggestion{
		ID:                  idgen.New("fwd"),
		ProblemID:           problemID,
		AdminAccountID:      adminAccountID,
		SuggestedOfficialID: suggestedOfficialID,
		Reason:              reason,
		CreatedAt:           time.Now().UTC(),
	}
}

// TopSuggestion returns the official the advisers most agree on, and how many said
// so. It returns ("", 0) when nobody has advised — and ALSO when the lead is tied.
//
// The tie is the reason this is a function rather than a sort: a tie is genuine
// disagreement, and breaking it arbitrarily (by id, by insertion order) would
// invent a consensus that does not exist and then make RequiresReason turn on that
// invention. "No top" is the honest answer, and callers must treat it as "the
// advisers did not converge", never as "nobody advised".
func TopSuggestion(suggestions []Suggestion) (officialID string, count int) {
	if len(suggestions) == 0 {
		return "", 0
	}
	tally := make(map[string]int, len(suggestions))
	for _, s := range suggestions {
		tally[s.SuggestedOfficialID]++
	}

	best, bestCount, tied := "", 0, false
	for id, n := range tally {
		switch {
		case n > bestCount:
			best, bestCount, tied = id, n, false
		case n == bestCount:
			tied = true
		}
	}
	if tied {
		return "", 0
	}
	return best, bestCount
}

// RequiresReason reports whether a forward owes the public a written explanation.
//
// Two independent triggers: the forward departs from the advisers' top suggestion,
// or it departs from the official the reporter pointed the report at. Pass top as
// "" when there is no top (nobody advised, or the advisers tied) — that removes the
// first trigger, leaving only the reporter's choice.
//
// The consequence to know before changing this: when the advisers and the reporter
// disagree, NO choice satisfies both, so a reason is always required. That is the
// intent rather than an edge case — disagreement is precisely when the public
// deserves the rationale.
//
// It lives here, in the domain, because it decides whether an assignment is audited
// as `assigned` or as `assignment_overridden`. Re-deriving it in a handler or in
// the UI is how the audit trail would come to disagree with the rule (A.4.6 — the
// UI may mirror this to grey a field, but the backend decides).
func RequiresReason(forwarded, top, pointed string) bool {
	if pointed != "" && forwarded != pointed {
		return true
	}
	if top != "" && forwarded != top {
		return true
	}
	return false
}
