// Package application (audit) exposes the seat's public decision record. The
// audit log is already the record of every state change, so this reads it rather
// than keeping a projection of its own that could disagree with it.
package application

import (
	"context"
	"sort"

	"jago-bahe-backend/internal/shared/audit/domain"
)

// publicActions are the moderator decisions the PUBLIC may see: every one of them
// is about a PROBLEM, and every one is already visible per-problem through
// GET /api/problems/{id}/audit. Aggregating them seat-wide adds pattern, not
// disclosure — one union admin quietly rejecting everything is invisible report by
// report and obvious here, which is the whole reason this endpoint exists.
//
// THE OMISSIONS ARE THE DESIGN. `verified`, `claim_approved` and `claim_rejected`
// are about PEOPLE, and they stay in the super admin's oversight feed
// (identity/application/oversight.go) where they already were:
//
//   - `verified` names a resident who passed the community filter. Publishing it
//     builds a public index of who is a verified resident of which union — the
//     person-graph that A.3.2 rule 1 treats as a safety matter on a politically
//     sensitive civic platform, not a preference. The problems are public; who
//     stands behind them is not.
//   - `claim_approved` / `claim_rejected` name a private individual's attempt to
//     claim a public office. An approval already becomes public the moment that
//     official acts. A REJECTION never should: it is an unproven identity claim,
//     and publishing "this person claimed to be the MP and was refused" is an
//     accusation the platform has no standing to make.
//
// Do not add them here. If a later phase wants more in the public feed, it must
// argue past those two paragraphs first.
// These are the ACTION STRINGS THE CODE ACTUALLY WRITES, verified against every
// NewAuditEntry call site — not the ones it would be reasonable to assume. Two of
// them were nearly missed, and the misses were not symmetric:
//
//   - `assignment_overridden` is the action when an admin passes over the
//     official the public asked for (assign_within_union.go:75-77 writes it
//     INSTEAD of `assigned`). Omitting it would have hidden the single most
//     accountability-relevant assignment event from the accountability feed, which
//     is precisely backwards — the override is the one the problem DTO already
//     calls "meant to show rather than merely store".
//   - `blocker_confirmed` / `blocker_denied` are what adjudicate_blocker.go
//     writes. The super admin's oversight set has asked for `"adjudicated"` since
//     B11 and nothing has ever written that string, so that feed has silently
//     shown zero adjudications for its whole existence.
//
// If you add an action here, grep for the literal in a NewAuditEntry call first.
var publicActions = map[string]bool{
	"approved":              true, // a report cleared screening and became public
	"rejected":              true, // a report was screened out, with its ground
	"assigned":              true, // a problem went to the official the public asked for
	"assignment_overridden": true, // ...or to someone else, with the admin's public reason
	"forwarding_suggested":  true, // a union admin advised where an above-union report should go
	"blocker_confirmed":     true, // an admin ruled a blocker genuine
	"blocker_denied":        true, // ...or ruled it not
}

// `vote_opened` and `vote_no_majority` were here until B20 and are now written by
// NOTHING — the binding admin vote they belonged to is deleted (CLAUDE.md A.3.8).
// They were removed rather than left harmlessly in place, because a set listing an
// action nobody writes is indistinguishable from a seat where nobody took it, which
// is the failure the `adjudicated` note above records.
//
// `forwarding_suggested` replaces them, and it is public where `vote_cast` was NOT.
// That reversal is deliberate and turns entirely on advice being non-binding: a
// secret ballot protects someone whose choice SETTLES something, and this one
// settles nothing — the super admin decides and may forward against all of it. What
// publishing it buys is that a forward going against the seat's admins is visible as
// such. If anything here ever becomes binding again, the old rule comes back with it.
//
// Deliberately NOT public, beyond the three person-actions:
//   - `reported`, `validated`, `edited`, `withdrawn`, `suggestion_proposed` and the
//     official's own lifecycle actions are not moderator decisions at all. They
//     already appear on the problem's own trail, and putting them here would make
//     this a second copy of the feed rather than a record of power exercised.

// PublicActions returns the action names the public feed covers, sorted so the
// set is stable to assert on. Exported for the test that pins the privacy split.
func PublicActions() []string {
	out := make([]string, 0, len(publicActions))
	for a := range publicActions {
		out = append(out, a)
	}
	sort.Strings(out)
	return out
}

// ListActivity is the seat's public decision record: what the union admins have
// done, in the open, newest first.
//
// It is a read and only a read, and it does not soften "oversight, not override"
// (Scaffold §2) — it widens who gets to do the overseeing. There is still no use
// case anywhere that reverses one of these decisions, and none should be added:
// the remedy for an admin abusing their position is removing them in public, which
// this feed is what makes possible.
type ListActivity struct {
	audit domain.Repository
}

// NewListActivity wires the use case.
func NewListActivity(a domain.Repository) *ListActivity {
	return &ListActivity{audit: a}
}

// Execute returns recent public decisions across the seat, newest first.
//
// Takes no caller: the feed is identical for everyone, signed in or not. A
// per-viewer variation would make it a different record for different readers,
// which is the opposite of a public record.
func (uc *ListActivity) Execute(ctx context.Context, limit int) ([]domain.AuditEntry, error) {
	return uc.audit.ListByActions(ctx, PublicActions(), limit)
}
