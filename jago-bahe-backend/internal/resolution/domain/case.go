// Package domain (resolution) owns the official's public workflow on an assigned
// problem: the Case aggregate and its strict lifecycle state machine, plus the
// typed obstacle and its advisory community judgment. Pure logic — no database,
// HTTP, or other contexts.
package domain

import "time"

// Status is a case's lifecycle state. The happy path is Assigned → Acknowledged →
// Planned → InProgress → Done → Resolved; Blocked and Reopened are side-states,
// and Disputed is the terminal dispute path (re-assignment after a dispute is
// TODO(contract) for a later phase). These mirror the frontend CaseStatus.
type Status string

const (
	StatusAssigned     Status = "Assigned"
	StatusAcknowledged Status = "Acknowledged"
	StatusPlanned      Status = "Planned"
	StatusInProgress   Status = "InProgress"
	StatusBlocked      Status = "Blocked"
	StatusDone         Status = "Done"
	StatusResolved     Status = "Resolved"
	StatusReopened     Status = "Reopened"
	StatusDisputed     Status = "Disputed"
)

// Case is the aggregate root: an assigned problem being worked in public. It is
// loaded and saved whole with its plan, updates, evidence, and obstacles (golden
// rule A.4.3). officialID is the assignee; monitorOfficialID (copied from the
// assignment) is the tier notified on obstacles/escalation.
type Case struct {
	ID                string
	ProblemID         string
	OfficialID        string
	MonitorOfficialID string
	Status            Status
	AcknowledgedAt    *time.Time
	Deadline          time.Time
	DisputeReason     string
	// EscalationLevel is the visibility rung the case has climbed to on the monitor
	// ladder from measured silence (0 = with the official only). The work stays
	// with the official; only visibility climbs (Concept §7). Set by the worker.
	EscalationLevel int
	LastEscalatedAt *time.Time
	CreatedAt       time.Time

	Plan      *Plan
	Updates   []ProgressUpdate
	Evidence  []Evidence
	Obstacles []Obstacle
}

// NewCase materializes a fresh case from an assignment (status Assigned, awaiting
// the official's acknowledgement).
func NewCase(id, problemID, officialID, monitorOfficialID string, deadline time.Time) *Case {
	return &Case{
		ID:                id,
		ProblemID:         problemID,
		OfficialID:        officialID,
		MonitorOfficialID: monitorOfficialID,
		Status:            StatusAssigned,
		Deadline:          deadline,
		CreatedAt:         time.Now().UTC(),
	}
}

// HasEvidence reports whether any before/after evidence is attached (the guard
// the lifecycle enforces before Done).
func (c *Case) HasEvidence() bool { return len(c.Evidence) > 0 }

// CompleteTask marks one of the plan's weekly tasks completed. It is idempotent in
// spirit but rejects an unknown id with ErrTaskNotFound; an already-completed task
// simply refreshes nothing meaningful and is left as-is. Completing a task never
// changes the case status — it is a public milestone, not a lifecycle event
// (whether every week is checked has no bearing on Done, which stays evidence-gated).
func (c *Case) CompleteTask(taskID string) error {
	if c.Plan == nil {
		return ErrTaskNotFound
	}
	for i := range c.Plan.Tasks {
		if c.Plan.Tasks[i].ID == taskID {
			if !c.Plan.Tasks[i].Completed {
				now := time.Now().UTC()
				c.Plan.Tasks[i].Completed = true
				c.Plan.Tasks[i].CompletedAt = &now
			}
			return nil
		}
	}
	return ErrTaskNotFound
}

// ActiveObstacle returns the case's current unresolved blocker (the one under
// public judgment) and whether one exists. The latest unresolved blocker wins.
func (c *Case) ActiveObstacle() (*Obstacle, bool) {
	for i := len(c.Obstacles) - 1; i >= 0; i-- {
		if c.Obstacles[i].ResolvedAt == nil {
			return &c.Obstacles[i], true
		}
	}
	return nil, false
}

// BlockedOnHigherAuthority reports whether this case is waiting on an obstacle a
// named authority judged real. It is the one nuance a public view owes an
// official: without it, a case they are blocked on reads as neglect.
//
// Only a *confirmed* blocker protects them — a pending or denied one does not,
// and the advisory public vote is never consulted (Concept §8; the fairness rule
// must not turn on a popularity vote).
//
// The scorecard read model states this same rule over its own SQL projection
// (scorecard/domain.FairnessFlag). That duplication is deliberate and documented
// there: the read model couples to no other domain package. Since resolution owns
// both Status and Adjudication, it states the rule in its own vocabulary rather
// than importing the reporting context — and a cross-context test pins the two to
// the same answer for every combination, which is what actually prevents drift.
func (c *Case) BlockedOnHigherAuthority() bool {
	b, ok := c.ActiveObstacle()
	return ok && c.Status == StatusBlocked && b.Adjudication == AdjudicationConfirmed
}
