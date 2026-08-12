package application

import (
	"context"

	"jago-bahe-backend/internal/shared/notification/domain"
)

// MarkAllRead clears the caller's whole badge, and is the escape hatch for someone
// who has fallen behind rather than the default way to read.
//
// It is an EXPLICIT action, never a side effect of opening the list. Marking
// everything read on GET would make a read mutate — which this codebase does
// nowhere — and it would destroy the bell for anyone who opens the page to see what
// is waiting rather than to dismiss it.
//
// Unaudited, for the reason given in mark_read.go.
type MarkAllRead struct {
	repo domain.Repository
}

// NewMarkAllRead wires the use case.
func NewMarkAllRead(r domain.Repository) *MarkAllRead {
	return &MarkAllRead{repo: r}
}

// Execute marks every unread message read and reports how many were changed.
func (uc *MarkAllRead) Execute(ctx context.Context, r domain.Recipients) (int, error) {
	if r.Empty() {
		return 0, domain.ErrNoCaller
	}
	return uc.repo.MarkAllRead(ctx, r)
}
