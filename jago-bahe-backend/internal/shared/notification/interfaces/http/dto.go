// Package http exposes the caller's own in-app notifications over REST. The DTO
// mirrors the frontend's Notification (lib/types/models.js); the domain type is
// never serialized directly (A.4.5).
package http

import (
	"jago-bahe-backend/internal/shared/notification/domain"
)

const timeLayout = "2006-01-02T15:04:05.000Z07:00"

// notificationDTO is one in-app message as the recipient sees it.
//
// RecipientKind and RecipientID are deliberately ABSENT. They are always the
// caller — the route has no way to return anyone else's rows — so echoing them
// would serve nothing but a reader looking for an id space to point at.
type notificationDTO struct {
	ID           string `json:"id"`
	Type         string `json:"type"`
	ProblemID    string `json:"problemId"`
	ProblemTitle string `json:"problemTitle"`

	// Detail is TYPE-SPECIFIC and is only ever read inside a switch on Type: the
	// rejection ground enum, the assigned official's name, an ISO deadline,
	// "confirmed"/"denied", the obstacle's free-text whoUnblocks, or "".
	Detail string `json:"detail"`

	// A pointer with NO omitempty: it must serialize as literal `null` while
	// unread, never be absent. Both are falsy in JavaScript, so a regression to
	// "sometimes missing" would surface only as a wrongly-styled row — the same
	// reasoning as problemDTO.MyVote and observationDTO.ResolvedAt (A.5.2 at the
	// field level).
	ReadAt *string `json:"readAt"`

	CreatedAt string `json:"createdAt"`
}

// unreadCountDTO is the bell's badge.
type unreadCountDTO struct {
	Count int `json:"count"`
}

// markAllReadDTO reports how many messages the caller just cleared.
type markAllReadDTO struct {
	Marked int `json:"marked"`
}

func toNotificationDTO(n domain.Notification) notificationDTO {
	dto := notificationDTO{
		ID:           n.ID,
		Type:         string(n.Type),
		ProblemID:    n.ProblemID,
		ProblemTitle: n.ProblemTitle,
		Detail:       n.Detail,
		CreatedAt:    n.CreatedAt.Format(timeLayout),
	}
	if n.ReadAt != nil {
		read := n.ReadAt.Format(timeLayout)
		dto.ReadAt = &read
	}
	return dto
}

// toNotificationDTOs maps a page. make(...,0,n) so an empty result encodes as `[]`
// rather than `null` — the client renders an empty state, not an error.
func toNotificationDTOs(ns []domain.Notification) []notificationDTO {
	out := make([]notificationDTO, 0, len(ns))
	for _, n := range ns {
		out = append(out, toNotificationDTO(n))
	}
	return out
}
