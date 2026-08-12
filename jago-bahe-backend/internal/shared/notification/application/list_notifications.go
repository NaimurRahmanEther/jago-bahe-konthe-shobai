// Package application (notification) reads the caller's own in-app messages. Every
// use case here takes a domain.Recipients built from the caller's JWT and nothing
// else; there is no parameter anywhere that names a recipient, and none may be
// added (CLAUDE.md A.3.9 constraint 1).
//
// There is deliberately NO ports.go. Unlike audit — which declares Actors and
// Problems because it must resolve ids to names at read time — a notification
// snapshots its problem title and detail at write time, so nothing here has to look
// anything up. If a ports.go appears in this package, something has been
// un-snapshotted; check that before adding one.
package application

import (
	"context"

	"jago-bahe-backend/internal/shared/notification/domain"
)

// defaultLimit and maxLimit bound the list. A resident's own notifications are
// bounded by the reports they filed, so this is a guard against a pathological
// caller rather than pagination — the codebase has none anywhere and this endpoint
// does not introduce it.
const (
	defaultLimit = 50
	maxLimit     = 200
)

// ListNotifications returns the caller's own messages, newest first.
//
// It is a READ and is therefore never audited (A.4.4 binds state-changing use
// cases). The audit log is public, so an entry per view would publish who is
// reading what.
type ListNotifications struct {
	repo domain.Repository
}

// NewListNotifications wires the use case.
func NewListNotifications(r domain.Repository) *ListNotifications {
	return &ListNotifications{repo: r}
}

// Execute lists the caller's messages. An anonymous caller is ErrNoCaller (a 401),
// never an empty slice: "you have no messages" and "we do not know who you are"
// are different claims and must not look the same.
func (uc *ListNotifications) Execute(ctx context.Context, r domain.Recipients, limit int) ([]domain.Notification, error) {
	if r.Empty() {
		return nil, domain.ErrNoCaller
	}
	if limit <= 0 {
		limit = defaultLimit
	}
	if limit > maxLimit {
		limit = maxLimit
	}
	return uc.repo.ListForRecipient(ctx, r, limit)
}
