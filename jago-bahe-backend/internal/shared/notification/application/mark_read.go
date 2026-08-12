package application

import (
	"context"
	"strings"

	"jago-bahe-backend/internal/shared/notification/domain"
)

// MarkRead marks one of the caller's own messages read.
//
// THIS IS THE ONE STATE CHANGE IN THE PLATFORM THAT APPENDS NO AUDIT ENTRY, and the
// exception is deliberate. A.4.4 makes the audit append mandatory for every
// state-changing use case, and this is technically one — but the audit log is
// PUBLIC. An entry per mark-read would publish who is reading what: surveillance
// wearing the costume of transparency, which is exactly the reason reads are
// unaudited (A.3.2 rule 4). This is that rule followed one step further, into the
// one write that is ABOUT reading.
//
// If you are here to "fix the missing audit append", do not. See CLAUDE.md A.3.9
// constraint 2.
type MarkRead struct {
	repo domain.Repository
}

// NewMarkRead wires the use case.
func NewMarkRead(r domain.Repository) *MarkRead {
	return &MarkRead{repo: r}
}

// Execute marks the message read.
//
// It does NOT load the row, check its owner, and refuse — the scope travels into
// the UPDATE, so a message belonging to someone else simply matches nothing and is
// reported as ErrNotificationNotFound, the same answer an unknown id gets. The
// handler maps that to 404 and never 403, so this route is no oracle over
// notification ids.
func (uc *MarkRead) Execute(ctx context.Context, id string, r domain.Recipients) error {
	if r.Empty() {
		return domain.ErrNoCaller
	}
	if strings.TrimSpace(id) == "" {
		return domain.ErrNotificationNotFound
	}
	return uc.repo.MarkRead(ctx, id, r)
}
