// Package http exposes the resolution use cases over REST. DTOs mirror the
// frontend's Case / Plan / Obstacle / UnblockingPlan typedefs; domain aggregates
// are never serialized directly (the DTO boundary, golden rule A.4.5).
package http

import (
	"time"

	"jago-bahe-backend/internal/resolution/domain"
)

const timeLayout = "2006-01-02T15:04:05.000Z07:00"

// --- requests ---

type acknowledgeRequest struct {
	Decision string `json:"decision"` // accept | dispute
	Reason   string `json:"reason"`
}

type planRequest struct {
	Strategy string `json:"strategy"`
	// Tasks is the week-by-week checklist: one task per week, week N = position N.
	// It replaces the old single timelineWeeks number (now derived from the count).
	Tasks              []string `json:"tasks"`
	Obstacles          string   `json:"obstacles"`
	SuggestionResponse string   `json:"suggestionResponse"`
}

// replanRequest is a planRequest plus the required reason the official is
// restarting — the "what changed" that goes on the public timeline and audit.
type replanRequest struct {
	Strategy           string   `json:"strategy"`
	Tasks              []string `json:"tasks"`
	Obstacles          string   `json:"obstacles"`
	SuggestionResponse string   `json:"suggestionResponse"`
	Reason             string   `json:"reason"`
}

type updateRequest struct {
	Kind string `json:"kind"` // progress | obstacle
	Text string `json:"text"`
}

type evidenceRequest struct {
	BeforeImageURL string `json:"beforeImageUrl"`
	AfterImageURL  string `json:"afterImageUrl"`
}

type obstacleRequest struct {
	Category    string `json:"category"`
	WhatBlocks  string `json:"whatBlocks"`
	WhoUnblocks string `json:"whoUnblocks"`
	ProofTried  string `json:"proofTried"`
}

type obstacleVoteRequest struct {
	Vote string `json:"vote"` // real | not_convinced
}

type unblockingPlanRequest struct {
	Text string `json:"text"`
}

type confirmRequest struct {
	Outcome string `json:"outcome"` // solved | not_solved
}

type adjudicateRequest struct {
	Decision string `json:"decision"` // confirm | deny
}

// --- responses ---

func ptrTime(t *time.Time) *string {
	if t == nil {
		return nil
	}
	s := t.Format(timeLayout)
	return &s
}

type caseDTO struct {
	ID                string        `json:"id"`
	ProblemID         string        `json:"problemId"`
	OfficialID        string        `json:"officialId"`
	MonitorOfficialID string        `json:"monitorOfficialId"`
	Status            string        `json:"status"`
	AcknowledgedAt    *string       `json:"acknowledgedAt"`
	Deadline          string        `json:"deadline"`
	DisputeReason     string        `json:"disputeReason,omitempty"`
	CreatedAt         string        `json:"createdAt"`
	EscalationLevel   int           `json:"escalationLevel"`
	LastEscalatedAt   *string       `json:"lastEscalatedAt"`
	Plan              *planDTO      `json:"plan"`
	Updates           []updateDTO   `json:"updates"`
	Evidence          []evidenceDTO `json:"evidence"`
	Obstacles         []obstacleDTO `json:"obstacles"`
}

type planDTO struct {
	ID                 string        `json:"id"`
	CaseID             string        `json:"caseId"`
	Strategy           string        `json:"strategy"`
	TimelineWeeks      int           `json:"timelineWeeks"`
	Obstacles          string        `json:"obstacles"`
	SuggestionResponse string        `json:"suggestionResponse"`
	Tasks              []planTaskDTO `json:"tasks"`
	CreatedAt          string        `json:"createdAt"`
}

type planTaskDTO struct {
	ID          string  `json:"id"`
	WeekNumber  int     `json:"weekNumber"`
	Task        string  `json:"task"`
	Completed   bool    `json:"completed"`
	CompletedAt *string `json:"completedAt"`
}

func toPlanTaskDTOs(tasks []domain.PlanTask) []planTaskDTO {
	out := make([]planTaskDTO, 0, len(tasks))
	for _, tk := range tasks {
		out = append(out, planTaskDTO{
			ID: tk.ID, WeekNumber: tk.WeekNumber, Task: tk.Task,
			Completed: tk.Completed, CompletedAt: ptrTime(tk.CompletedAt),
		})
	}
	return out
}

type updateDTO struct {
	ID        string `json:"id"`
	CaseID    string `json:"caseId"`
	Kind      string `json:"kind"`
	Text      string `json:"text"`
	CreatedAt string `json:"createdAt"`
}

type evidenceDTO struct {
	ID             string `json:"id"`
	CaseID         string `json:"caseId"`
	BeforeImageURL string `json:"beforeImageUrl"`
	AfterImageURL  string `json:"afterImageUrl"`
	CreatedAt      string `json:"createdAt"`
}

type unblockingPlanDTO struct {
	ID          string `json:"id"`
	ObstacleID  string `json:"obstacleId"`
	AuthorID    string `json:"authorId"`
	Text        string `json:"text"`
	UpvoteCount int    `json:"upvoteCount"`
	CreatedAt   string `json:"createdAt"`
}

type obstacleDTO struct {
	ID                string              `json:"id"`
	CaseID            string              `json:"caseId"`
	Category          string              `json:"category"`
	WhatBlocks        string              `json:"whatBlocks"`
	WhoUnblocks       string              `json:"whoUnblocks"`
	ProofTried        string              `json:"proofTried"`
	RealCount         int                 `json:"realCount"`
	NotConvincedCount int                 `json:"notConvincedCount"`
	UnblockingPlans   []unblockingPlanDTO `json:"unblockingPlans"`
	Adjudication      string              `json:"adjudication"`
	AdjudicatedAt     *string             `json:"adjudicatedAt"`
	CreatedAt         string              `json:"createdAt"`
	ResolvedAt        *string             `json:"resolvedAt"`
}

func toCaseDTO(c *domain.Case) caseDTO {
	dto := caseDTO{
		ID:                c.ID,
		ProblemID:         c.ProblemID,
		OfficialID:        c.OfficialID,
		MonitorOfficialID: c.MonitorOfficialID,
		Status:            string(c.Status),
		AcknowledgedAt:    ptrTime(c.AcknowledgedAt),
		Deadline:          c.Deadline.Format(timeLayout),
		DisputeReason:     c.DisputeReason,
		CreatedAt:         c.CreatedAt.Format(timeLayout),
		EscalationLevel:   c.EscalationLevel,
		LastEscalatedAt:   ptrTime(c.LastEscalatedAt),
		Updates:           make([]updateDTO, 0, len(c.Updates)),
		Evidence:          make([]evidenceDTO, 0, len(c.Evidence)),
		Obstacles:         make([]obstacleDTO, 0, len(c.Obstacles)),
	}
	if c.Plan != nil {
		dto.Plan = &planDTO{
			ID: c.Plan.ID, CaseID: c.Plan.CaseID, Strategy: c.Plan.Strategy,
			TimelineWeeks: c.Plan.TimelineWeeks, Obstacles: c.Plan.Obstacles,
			SuggestionResponse: c.Plan.SuggestionResponse, Tasks: toPlanTaskDTOs(c.Plan.Tasks),
			CreatedAt: c.Plan.CreatedAt.Format(timeLayout),
		}
	}
	for _, u := range c.Updates {
		dto.Updates = append(dto.Updates, updateDTO{
			ID: u.ID, CaseID: u.CaseID, Kind: string(u.Kind), Text: u.Text, CreatedAt: u.CreatedAt.Format(timeLayout),
		})
	}
	for _, e := range c.Evidence {
		dto.Evidence = append(dto.Evidence, evidenceDTO{
			ID: e.ID, CaseID: e.CaseID, BeforeImageURL: e.BeforeImageURL, AfterImageURL: e.AfterImageURL, CreatedAt: e.CreatedAt.Format(timeLayout),
		})
	}
	for i := range c.Obstacles {
		dto.Obstacles = append(dto.Obstacles, toObstacleDTO(&c.Obstacles[i]))
	}
	return dto
}

func toCaseDTOs(cases []domain.Case) []caseDTO {
	out := make([]caseDTO, 0, len(cases))
	for i := range cases {
		out = append(out, toCaseDTO(&cases[i]))
	}
	return out
}

func toObstacleDTO(b *domain.Obstacle) obstacleDTO {
	plans := make([]unblockingPlanDTO, 0, len(b.UnblockingPlans))
	for _, p := range b.UnblockingPlans {
		plans = append(plans, toUnblockingPlanDTO(&p))
	}
	return obstacleDTO{
		ID: b.ID, CaseID: b.CaseID, Category: string(b.Category),
		WhatBlocks: b.WhatBlocks, WhoUnblocks: b.WhoUnblocks, ProofTried: b.ProofTried,
		RealCount: b.RealCount, NotConvincedCount: b.NotConvincedCount,
		UnblockingPlans: plans, Adjudication: string(b.Adjudication), AdjudicatedAt: ptrTime(b.AdjudicatedAt),
		CreatedAt: b.CreatedAt.Format(timeLayout), ResolvedAt: ptrTime(b.ResolvedAt),
	}
}

func toUnblockingPlanDTO(p *domain.UnblockingPlan) unblockingPlanDTO {
	return unblockingPlanDTO{
		ID: p.ID, ObstacleID: p.ObstacleID, AuthorID: p.AuthorID, Text: p.Text,
		UpvoteCount: p.UpvoteCount, CreatedAt: p.CreatedAt.Format(timeLayout),
	}
}
