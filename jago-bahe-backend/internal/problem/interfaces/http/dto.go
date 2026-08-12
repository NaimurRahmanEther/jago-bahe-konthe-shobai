// Package http exposes the problem use cases over REST. DTOs mirror the
// frontend's Problem/ValidationVote/AuditEntry typedefs; domain aggregates are
// never serialized directly.
package http

import (
	"jago-bahe-backend/internal/problem/application"
	problemdomain "jago-bahe-backend/internal/problem/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
)

type locationDTO struct {
	AreaID  string   `json:"areaId"`
	Address string   `json:"address,omitempty"`
	Lat     *float64 `json:"lat,omitempty"`
	Lng     *float64 `json:"lng,omitempty"`
}

type reportRequest struct {
	Title             string      `json:"title"`
	Description       string      `json:"description"`
	Location          locationDTO `json:"location"`
	PointedOfficialID string      `json:"pointedOfficialId"`
	ProposedSolution  string      `json:"proposedSolution"`
	ImageURL          string      `json:"imageUrl"`
}

type validateRequest struct {
	Vote string `json:"vote"`
}

// updateRequest is a reporter's edit body. Only the content fields a reporter may
// change are accepted; areaId and pointedOfficialId are ignored even if sent,
// because they fix the jurisdiction and the accountable official at filing time.
type updateRequest struct {
	Title            string      `json:"title"`
	Description      string      `json:"description"`
	Location         locationDTO `json:"location"`
	ProposedSolution string      `json:"proposedSolution"`
}

// withdrawRequest is a reporter's retraction body. Note is an optional free-text
// reason recorded on the public audit entry — a reporter owes no fixed ground for
// taking down their own report.
type withdrawRequest struct {
	Note string `json:"note"`
}

// rejectRequest is the screening rejection body. Reason must be one of the four
// fixed grounds — the domain refuses anything else, so free text cannot become a
// ground by going through this field. Note is optional colour, recorded on the
// public audit entry alongside (never instead of) the ground.
type rejectRequest struct {
	Reason string `json:"reason"`
	Note   string `json:"note"`
}

// problemDTO mirrors the frontend Problem. validationThreshold is the configured
// V, sent for the "X of V" display only.
type problemDTO struct {
	ID                string      `json:"id"`
	Title             string      `json:"title"`
	Description       string      `json:"description"`
	Location          locationDTO `json:"location"`
	ReporterID        string      `json:"reporterId"`
	PointedOfficialID string      `json:"pointedOfficialId"`

	// AssignedOfficialID is the official the problem was actually handed to, or
	// absent when it has not been assigned. It is a DIFFERENT fact from
	// PointedOfficialID: pointed is who the public asked for, assigned is who the
	// admin decided, and the gap between them is an override — which the admin
	// must justify with a public reason, and which the public record is meant to
	// show rather than merely store.
	//
	// omitempty is correct here (unlike MyVote): absent and empty both mean "not
	// assigned", so there is no third state for a missing key to be confused with.
	//
	// Served on the FEED as well as the detail read. It was detail-only until the
	// public record was asked to say who is working on each report — the objection
	// then was a per-row assignment lookup, an N+1. It is composed in one batch
	// query instead (handler.assignmentLookup), exactly as MyVote is. That is the
	// difference that makes it affordable: keep it batched. Suggestions and
	// Evidence below stay list-empty because no such batch exists for them.
	AssignedOfficialID string `json:"assignedOfficialId,omitempty"`

	// OverrideReason is the public justification an admin must give when they
	// assign someone OTHER than the official the public pointed at. Empty in the
	// ordinary case, where the admin confirmed the public's choice and so had
	// nothing to justify.
	//
	// It is published because an override the public cannot see is not accountable:
	// the record would show the gap between pointed and assigned while withholding
	// the one field that explains it.
	OverrideReason string `json:"overrideReason,omitempty"`

	ProposedSolution    string `json:"proposedSolution,omitempty"`
	ImageURL            string `json:"imageUrl,omitempty"`
	Status              string `json:"status"`
	ValidCount          int    `json:"validCount"`
	ValidationThreshold int    `json:"validationThreshold"`
	CreatedAt           string `json:"createdAt"`

	// MyVote is the CALLING account's own validation vote on this problem, or null
	// when they have not voted, are not a resident, or are anonymous. It is what
	// lets the UI show a vote already cast instead of re-offering the buttons —
	// votes are immutable, so once non-null it never changes.
	//
	// A pointer with NO omitempty: it must serialize as literal `null`, never
	// vanish. An absent key and a null are both falsy in JavaScript, so a
	// regression to "sometimes missing" would be invisible in the UI and surface
	// only as a wrongly-enabled button (the A.5.2 lesson at the field level).
	MyVote *string `json:"myVote"`

	// Screening outcome (B9). RejectionReason is populated only when Status is
	// Rejected and is shown publicly — a rejection carries its ground with it.
	RejectionReason string `json:"rejectionReason,omitempty"`
	ReviewedBy      string `json:"reviewedBy,omitempty"`
	ReviewedAt      string `json:"reviewedAt,omitempty"`

	Suggestions []any          `json:"suggestions,omitempty"` // served by GET /problems/{id}/suggestions (B3); left empty here by design
	Evidence    []any          `json:"evidence,omitempty"`    // TODO(contract): populated in B5
	Audit       []auditItemDTO `json:"audit,omitempty"`
}

type auditItemDTO struct {
	ID         string `json:"id"`
	TargetType string `json:"targetType"`
	TargetID   string `json:"targetId"`
	Actor      string `json:"actor"`
	Action     string `json:"action"`
	Reason     string `json:"reason,omitempty"`
	CreatedAt  string `json:"createdAt"`
}

func toLocationDTO(p *problemdomain.Problem) locationDTO {
	return locationDTO{
		AreaID:  p.Location.AreaID.String(),
		Address: p.Location.Address,
		Lat:     p.Location.Lat,
		Lng:     p.Location.Lng,
	}
}

// toProblemDTO maps a problem to its wire shape. threshold is the configured V.
// myVote is the calling account's own vote on this problem, or nil — it is a
// viewer-relative decoration composed in the handler, not a fact about the
// problem, which is why it arrives as an argument rather than off p.
func toProblemDTO(p *problemdomain.Problem, threshold int, myVote *string) problemDTO {
	dto := problemDTO{
		ID:                  p.ID,
		Title:               p.Title,
		Description:         p.Description,
		Location:            toLocationDTO(p),
		ReporterID:          p.ReporterID,
		PointedOfficialID:   p.PointedOfficialID,
		ProposedSolution:    p.ProposedSolution,
		ImageURL:            p.ImageURL,
		Status:              string(p.Status),
		ValidCount:          p.ValidCount,
		ValidationThreshold: threshold,
		CreatedAt:           p.CreatedAt.Format(timeLayout),
		RejectionReason:     string(p.RejectionReason),
		ReviewedBy:          p.ReviewedBy,
		MyVote:              myVote,
	}
	if p.ReviewedAt != nil {
		dto.ReviewedAt = p.ReviewedAt.Format(timeLayout)
	}
	return dto
}

// applyAssignment stamps the assignment facts onto a DTO the mapper already built.
//
// Separate from toProblemDTO for the reason its own doc comment gives about
// myVote: the mapper is shared with every write path (report, update, withdraw,
// approve, reject), and none of those has an assignment to hand. A parameter there
// would make eleven call sites pass a zero value to say "not applicable".
//
// A zero view leaves both fields empty, which omitempty then drops — so an
// unassigned problem carries no assignment keys at all.
func applyAssignment(dto *problemDTO, a application.AssignmentView) {
	dto.AssignedOfficialID = a.OfficialID
	dto.OverrideReason = a.OverrideReason
}

func toAuditDTO(entries []auditdomain.AuditEntry) []auditItemDTO {
	out := make([]auditItemDTO, 0, len(entries))
	for _, e := range entries {
		out = append(out, auditItemDTO{
			ID:         e.ID,
			TargetType: e.TargetType,
			TargetID:   e.TargetID,
			Actor:      e.Actor,
			Action:     e.Action,
			Reason:     e.Reason,
			CreatedAt:  e.CreatedAt.Format(timeLayout),
		})
	}
	return out
}

const timeLayout = "2006-01-02T15:04:05.000Z07:00"
