package application

import (
	"context"

	"jago-bahe-backend/internal/shared/notification/domain"
)

// UnreadCount is the bell's badge.
//
// It is a separate use case and a separate endpoint from the list, rather than a
// number derived from it, because the bell mounts in the nav on EVERY page: making
// it pull fifty rows would put that payload behind every navigation, and would make
// the badge wrong the moment the list hit its cap.
//
// The count leaks nothing. Its denominator is exactly the set the caller may read —
// their own messages — which is what separates it from the B10 SeatProblemCount
// regression, where an unfiltered count published how many unscreened reports
// existed.
type UnreadCount struct {
	repo domain.Repository
}

// NewUnreadCount wires the use case.
func NewUnreadCount(r domain.Repository) *UnreadCount {
	return &UnreadCount{repo: r}
}

// Execute counts the caller's unread messages.
func (uc *UnreadCount) Execute(ctx context.Context, r domain.Recipients) (int, error) {
	if r.Empty() {
		return 0, domain.ErrNoCaller
	}
	return uc.repo.CountUnread(ctx, r)
}
