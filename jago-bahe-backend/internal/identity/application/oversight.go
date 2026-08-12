package application

import (
	"context"

	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
)

// adminActions are the audit actions that represent a moderator exercising power
// over someone else's report, case, or standing. These are what oversight watches.
// adminActions are the audit actions that represent a moderator exercising power
// over someone else's report, case, or standing. These are what oversight watches.
//
// The problem half of this set is now ALSO served publicly, at GET /api/seat/activity
// (shared/audit/application/list_activity.go). This feed keeps both halves, because
// the three person-actions below are the super admin's remaining reason to exist:
// publishing `verified` would index who is a verified resident of which union, and
// `claim_rejected` would broadcast a refused identity claim.
//
// Two corrections were made when the public feed was built, by checking these
// strings against the NewAuditEntry call sites instead of assuming:
//
//   - `adjudicated` was never written by anything. adjudicate_blocker.go writes
//     `blocker_confirmed` / `blocker_denied`, so this feed has shown ZERO
//     adjudications since B11 — silently, because a feed missing a row looks the
//     same as a seat where nobody adjudicated.
//   - `assignment_overridden` was missing. assign_within_union.go writes it INSTEAD
//     of `assigned` when an admin passes over the public's choice, so the one
//     assignment event most worth overseeing was the one this could not see.
//
// If you add an action here, grep for the literal in a NewAuditEntry call first.
var adminActions = map[string]bool{
	"approved":              true, // a report published
	"rejected":              true, // a report screened out
	"assigned":              true, // a problem handed to the official the public asked for
	"assignment_overridden": true, // ...or to someone else, over the public's choice
	// `vote_opened` / `vote_no_majority` were here until B20 and are written by
	// NOTHING now — the binding admin vote is deleted (CLAUDE.md A.3.8). Left in
	// place they would be indistinguishable from a seat where nobody voted, which is
	// the exact failure the `adjudicated` note above records.
	"forwarding_suggested": true, // a union admin advised where an above-union report should go
	"blocker_confirmed":    true, // an admin ruled a blocker genuine
	"blocker_denied":       true, // ...or ruled it not
	"claim_approved":       true,
	"claim_rejected":       true,
	"verified":             true, // a resident admitted to the community filter
}

// Oversight is the super admin's seat-wide view of what the union admins have
// been doing.
//
// It is a read and only a read. There is no counterpart use case that reverses
// any of these decisions, and none should be added: an account able to overrule
// every union admin would sit above every check in this design with nothing above
// it, which is the concentration Concept §10 refuses. Sight is the power here —
// a union admin burying reports becomes visible, and the remedy is removing them
// in public, not silently undoing them.
//
// It reads the audit log rather than a table of its own, because the audit log is
// already the record of every state change; a second store could disagree with it.
type Oversight struct {
	audit auditdomain.Repository
}

// NewOversight wires the use case.
func NewOversight(a auditdomain.Repository) *Oversight {
	return &Oversight{audit: a}
}

// Execute returns recent moderator decisions across the seat, newest first.
func (uc *Oversight) Execute(ctx context.Context, limit int) ([]auditdomain.AuditEntry, error) {
	entries, err := uc.audit.ListByActions(ctx, keys(adminActions), limit)
	if err != nil {
		return nil, err
	}
	return entries, nil
}

func keys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
