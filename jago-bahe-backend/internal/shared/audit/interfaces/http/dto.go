// Package http exposes the seat's public decision record over REST. The DTO
// mirrors the frontend's ActivityEntry (lib/types/models.js); the domain entry is
// never serialized directly.
package http

import (
	"jago-bahe-backend/internal/shared/audit/domain"
)

const timeLayout = "2006-01-02T15:04:05.000Z07:00"

// activityEntryDTO is one public decision, resolved into something a reader can
// actually read.
//
// It is deliberately NOT the same shape as the per-problem auditItemDTO. That one
// serves a trail already scoped to a problem the reader is looking at, so an
// actor id and an action are enough. This one is seat-wide, so a row has to say
// on its own WHO acted and WHICH report it was about — otherwise the aggregate
// that makes a pattern visible is a page of opaque ids.
type activityEntryDTO struct {
	ID     string `json:"id"`
	Action string `json:"action"`

	// ActorID is always present; ActorName only when the id resolved to an
	// account or a directory office. `system` resolves to neither, and the client
	// renders its own label for it — the platform acting by rule is a different
	// kind of fact from a person deciding, and the audit domain keeps that
	// distinction in one spelling (domain.ActorSystem).
	ActorID   string `json:"actorId"`
	ActorName string `json:"actorName,omitempty"`

	TargetType string `json:"targetType"`
	TargetID   string `json:"targetId"`

	// ProblemTitle is present only for a target that is a publicly visible
	// problem. Absent is normal, not an error: the repository resolves public
	// problems only, so a title that does not appear is one the reader was never
	// entitled to see.
	ProblemTitle string `json:"problemTitle,omitempty"`

	Reason    string `json:"reason,omitempty"`
	CreatedAt string `json:"createdAt"`
}

func toActivityDTOs(entries []domain.AuditEntry, actorNames, problemTitles map[string]string) []activityEntryDTO {
	// Non-nil so an empty feed encodes as [] rather than null: the client maps
	// over this, and null would be a different failure than "nothing yet".
	out := make([]activityEntryDTO, 0, len(entries))
	for _, e := range entries {
		dto := activityEntryDTO{
			ID:         e.ID,
			Action:     e.Action,
			ActorID:    e.Actor,
			ActorName:  actorNames[e.Actor],
			TargetType: e.TargetType,
			TargetID:   e.TargetID,
			Reason:     e.Reason,
			CreatedAt:  e.CreatedAt.UTC().Format(timeLayout),
		}
		if e.TargetType == "problem" {
			dto.ProblemTitle = problemTitles[e.TargetID]
		}
		out = append(out, dto)
	}
	return out
}
