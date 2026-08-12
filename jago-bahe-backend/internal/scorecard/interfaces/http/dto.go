// Package http exposes the scorecard read model over REST. The DTOs mirror the
// frontend's OfficialStats / OfficialRecord (lib/types/models.js) — the domain
// read models are never serialized directly.
package http

import (
	"time"

	"jago-bahe-backend/internal/scorecard/domain"
)

const timeLayout = "2006-01-02T15:04:05.000Z07:00"

// scorecardResponse mirrors the frontend OfficialStats.
type scorecardResponse struct {
	OfficialID      string  `json:"officialId"`
	Resolved        int     `json:"resolved"`
	Pending         int     `json:"pending"`
	Blocked         int     `json:"blocked"`
	AvgResponseDays float64 `json:"avgResponseDays"`
}

func toScorecardDTO(s domain.OfficialStats) scorecardResponse {
	return scorecardResponse{
		OfficialID:      s.OfficialID,
		Resolved:        s.Resolved,
		Pending:         s.Pending,
		Blocked:         s.Blocked,
		AvgResponseDays: s.AvgResponseDays,
	}
}

// seatOverviewResponse mirrors the frontend seat overview.
type seatOverviewResponse struct {
	Problems        int     `json:"problems"`
	Cases           int     `json:"cases"`
	Resolved        int     `json:"resolved"`
	Pending         int     `json:"pending"`
	Blocked         int     `json:"blocked"`
	AvgResponseDays float64 `json:"avgResponseDays"`
}

func toSeatOverviewDTO(o domain.SeatOverview) seatOverviewResponse {
	return seatOverviewResponse{
		Problems:        o.Problems,
		Cases:           o.Cases,
		Resolved:        o.Resolved,
		Pending:         o.Pending,
		Blocked:         o.Blocked,
		AvgResponseDays: o.AvgResponseDays,
	}
}

// caseRecordDTO is one entry in an official's public record.
//
// answeredSuggestion is the plan's snapshot of the community's top suggestion as
// it stood when the plan was written — not today's top. The UI must render it as
// given and never re-rank; pairing a response with a question the official was
// never asked would misrepresent them (Concept §10).
type caseRecordDTO struct {
	CaseID    string `json:"caseId"`
	ProblemID string `json:"problemId"`
	Title     string `json:"title"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	Deadline  string `json:"deadline"`

	Acknowledged   bool   `json:"acknowledged"`
	AcknowledgedAt string `json:"acknowledgedAt,omitempty"`

	HasPlan            bool   `json:"hasPlan"`
	Strategy           string `json:"strategy,omitempty"`
	TimelineWeeks      int    `json:"timelineWeeks,omitempty"`
	Obstacles          string `json:"obstacles,omitempty"`
	SuggestionResponse string `json:"suggestionResponse,omitempty"`
	AnsweredSuggestion string `json:"answeredSuggestion,omitempty"`
	PlanCreatedAt      string `json:"planCreatedAt,omitempty"`

	// BlockedOnHigherAuthority carries the fairness rule to the UI so a case
	// waiting on someone above this official is shown as waiting (amber), not as
	// neglect. Same rule as the scorecard's blocked bucket.
	BlockedOnHigherAuthority bool `json:"blockedOnHigherAuthority"`
}

// officialRecordResponse is the public record: a list of what an official did.
// It carries no score, rank, or grade by design.
type officialRecordResponse struct {
	OfficialID string          `json:"officialId"`
	Cases      []caseRecordDTO `json:"cases"`
}

func toOfficialRecordDTO(rec domain.OfficialRecord) officialRecordResponse {
	cases := make([]caseRecordDTO, 0, len(rec.Cases))
	for _, c := range rec.Cases {
		cases = append(cases, caseRecordDTO{
			CaseID:                   c.CaseID,
			ProblemID:                c.ProblemID,
			Title:                    c.Title,
			Status:                   c.Status,
			CreatedAt:                c.CreatedAt.Format(timeLayout),
			Deadline:                 c.Deadline.Format(timeLayout),
			Acknowledged:             c.Acknowledged,
			AcknowledgedAt:           formatTime(c.AcknowledgedAt),
			HasPlan:                  c.HasPlan,
			Strategy:                 c.Strategy,
			TimelineWeeks:            c.TimelineWeeks,
			Obstacles:                c.Obstacles,
			SuggestionResponse:       c.SuggestionResponse,
			AnsweredSuggestion:       c.AnsweredSuggestion,
			PlanCreatedAt:            formatTime(c.PlanCreatedAt),
			BlockedOnHigherAuthority: c.BlockedOnHigherAuthority,
		})
	}
	return officialRecordResponse{OfficialID: rec.OfficialID, Cases: cases}
}

func formatTime(t *time.Time) string {
	if t == nil {
		return ""
	}
	return t.Format(timeLayout)
}
