package application_test

import (
	"context"

	notificationdomain "jago-bahe-backend/internal/shared/notification/domain"
)

// fakeNotifier records every notification a use case appends, so a test can assert
// on WHO was told WHAT — and, just as importantly, that a REFUSED path told nobody.
//
// It mirrors the real repository's empty-recipient guard, so a test that hands it a
// blank recipient sees the same "nothing was written" the production path produces.
type fakeNotifier struct {
	sent []notificationdomain.Notification
}

func (f *fakeNotifier) Append(_ context.Context, n notificationdomain.Notification) error {
	if n.RecipientID == "" {
		return notificationdomain.ErrNoRecipient
	}
	f.sent = append(f.sent, n)
	return nil
}

func (f *fakeNotifier) ListForRecipient(context.Context, notificationdomain.Recipients, int) ([]notificationdomain.Notification, error) {
	return nil, nil
}
func (f *fakeNotifier) CountUnread(context.Context, notificationdomain.Recipients) (int, error) {
	return 0, nil
}
func (f *fakeNotifier) MarkRead(context.Context, string, notificationdomain.Recipients) error {
	return nil
}
func (f *fakeNotifier) MarkAllRead(context.Context, notificationdomain.Recipients) (int, error) {
	return 0, nil
}

var _ notificationdomain.Repository = (*fakeNotifier)(nil)
