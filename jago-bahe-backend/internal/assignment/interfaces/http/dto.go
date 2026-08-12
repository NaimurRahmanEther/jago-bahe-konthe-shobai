// Package http exposes the assignment use cases over REST. DTOs mirror the
// frontend's Assignment / ForwardingItem typedefs; domain aggregates are never
// serialized directly (the DTO boundary, golden rule A.4.5).
package http

import (
	"time"

	"jago-bahe-backend/internal/assignment/application"
	"jago-bahe-backend/internal/assignment/domain"
)

const timeLayout = "2006-01-02T15:04:05.000Z07:00"

// --- requests ---

type assignRequest struct {
	OfficialID     string `json:"officialId"`
	Priority       string `json:"priority"`
	Deadline       string `json:"deadline"`       // optional ISO-8601; empty → now + D
	OverrideReason string `json:"overrideReason"` // required when overriding the public's choice
}

type suggestRequest struct {
	OfficialID string `json:"officialId"`
	Reason     string `json:"reason"` // optional; the adviser's own public note
}

type forwardRequest struct {
	OfficialID string `json:"officialId"`
	Priority   string `json:"priority"`
	Deadline   string `json:"deadline"` // optional ISO-8601; empty → now + D
	// Reason is required when the choice departs from the advisers' top suggestion
	// or from the official the reporter pointed the report at. The BACKEND decides
	// that (domain.RequiresReason); the client may mirror the rule to mark the
	// field, but it never gets to decide it (A.5.7).
	Reason string `json:"reason"`
}

// parseDeadline converts an optional ISO-8601 deadline to *time.Time (nil when
// empty), and reports whether the value was well-formed.
func parseDeadline(s string) (*time.Time, bool) {
	if s == "" {
		return nil, true
	}
	t, err := time.Parse(time.RFC3339, s)
	if err != nil {
		return nil, false
	}
	return &t, true
}

// --- responses ---

// queueItemDTO is one row of the admin's assignment queue.
//
// ValidCount and ValidationThreshold mirror problemDTO's fields of the same names
// byte-for-byte, deliberately: they are the same two numbers rendered by the same
// frontend component, and two admin-facing shapes that disagree about the
// community's validation count would be worse than either alone. Both are display
// only — since B17 the count gates nothing (A.3.1).
//
// Address and Routing exist because the UI needs them and cannot derive them:
// address is free text on the problem row (not recoverable from AreaID), and
// routing is backend authority. Both were absent before B17, and the UI read them
// off the row anyway — see the note on ProblemView.Routing.
type queueItemDTO struct {
	ProblemID           string `json:"problemId"`
	Title               string `json:"title"`
	Status              string `json:"status"`
	Address             string `json:"address"`
	PointedOfficialID   string `json:"pointedOfficialId"`
	AreaID              string `json:"areaId"`
	Routing             string `json:"routing"`
	ValidCount          int    `json:"validCount"`
	ValidationThreshold int    `json:"validationThreshold"`
}

func toQueueDTOs(items []application.ProblemView, threshold int) []queueItemDTO {
	out := make([]queueItemDTO, 0, len(items))
	for _, p := range items {
		out = append(out, queueItemDTO{
			ProblemID:           p.ID,
			Title:               p.Title,
			Status:              p.Status,
			Address:             p.Address,
			PointedOfficialID:   p.PointedOfficialID,
			AreaID:              p.AreaID,
			Routing:             p.Routing,
			ValidCount:          p.ValidCount,
			ValidationThreshold: threshold,
		})
	}
	return out
}

type assignmentDTO struct {
	ID                string `json:"id"`
	ProblemID         string `json:"problemId"`
	OfficialID        string `json:"officialId"`
	MonitorOfficialID string `json:"monitorOfficialId"`
	Priority          string `json:"priority"`
	Deadline          string `json:"deadline"`
	OverrideReason    string `json:"overrideReason,omitempty"`
	CreatedAt         string `json:"createdAt"`
}

func toAssignmentDTO(a *domain.Assignment) assignmentDTO {
	return assignmentDTO{
		ID:                a.ID,
		ProblemID:         a.ProblemID,
		OfficialID:        a.OfficialID,
		MonitorOfficialID: a.MonitorOfficialID,
		Priority:          a.Priority,
		Deadline:          a.Deadline.Format(timeLayout),
		OverrideReason:    a.OverrideReason,
		CreatedAt:         a.CreatedAt.Format(timeLayout),
	}
}

// suggestionDTO is one union admin's advice on an above-union forward. The
// adviser is named: the tally is public, and an anonymous tally would let an admin
// influence the seat's assignments without standing behind it (the opposite of the
// vote's secret-ballot rule, which protected a BINDING choice — advice that binds
// nobody has no such claim).
type suggestionDTO struct {
	ID             string `json:"id"`
	AdminAccountID string `json:"adminAccountId"`
	OfficialID     string `json:"officialId"`
	Reason         string `json:"reason,omitempty"`
	CreatedAt      string `json:"createdAt"`
}

func toSuggestionDTO(s domain.Suggestion) suggestionDTO {
	return suggestionDTO{
		ID:             s.ID,
		AdminAccountID: s.AdminAccountID,
		OfficialID:     s.SuggestedOfficialID,
		Reason:         s.Reason,
		CreatedAt:      s.CreatedAt.Format(timeLayout),
	}
}

// forwardingItemDTO is one above-union report awaiting the super admin, on both
// the advisers' list and the super admin's queue — one shape, so the two surfaces
// can never disagree about what the seat has been advised.
type forwardingItemDTO struct {
	ProblemID           string `json:"problemId"`
	Title               string `json:"title"`
	Status              string `json:"status"`
	Address             string `json:"address"`
	AreaID              string `json:"areaId"`
	PointedOfficialID   string `json:"pointedOfficialId"`
	Scope               string `json:"scope"`
	ValidCount          int    `json:"validCount"`
	ValidationThreshold int    `json:"validationThreshold"`

	Suggestions []suggestionDTO `json:"suggestions"`
	// TopOfficialID is empty when nobody advised AND when the advisers tied — a tie
	// is genuine disagreement, and inventing a winner would make the reason
	// requirement turn on a coin flip (domain.TopSuggestion).
	TopOfficialID string `json:"topOfficialId"`
	TopCount      int    `json:"topCount"`
	// MySuggestion is the calling admin's own advice, present-and-null when they
	// have not advised, never absent — the myVote lesson at the field level
	// (A.3.2.1 rule 2): both are falsy in JS, so a field that came and went would
	// surface only as a wrongly pre-filled select. Always null for the super admin,
	// who advises on nothing.
	MySuggestion *suggestionDTO `json:"mySuggestion"`
}

func toForwardingDTOs(rows []application.ForwardingCandidate, threshold int) []forwardingItemDTO {
	out := make([]forwardingItemDTO, 0, len(rows))
	for _, r := range rows {
		suggestions := make([]suggestionDTO, 0, len(r.Suggestions))
		for _, s := range r.Suggestions {
			suggestions = append(suggestions, toSuggestionDTO(s))
		}
		var mine *suggestionDTO
		if r.MySuggestion != nil {
			d := toSuggestionDTO(*r.MySuggestion)
			mine = &d
		}
		out = append(out, forwardingItemDTO{
			ProblemID:           r.Problem.ID,
			Title:               r.Problem.Title,
			Status:              r.Problem.Status,
			Address:             r.Problem.Address,
			AreaID:              r.Problem.AreaID,
			PointedOfficialID:   r.Problem.PointedOfficialID,
			Scope:               string(r.Scope),
			ValidCount:          r.Problem.ValidCount,
			ValidationThreshold: threshold,
			Suggestions:         suggestions,
			TopOfficialID:       r.TopOfficialID,
			TopCount:            r.TopCount,
			MySuggestion:        mine,
		})
	}
	return out
}
