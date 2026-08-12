package domain

import (
	"time"

	"jago-bahe-backend/pkg/idgen"
)

// PlanTask is one week's task in an official's plan — the plan read as a
// week-by-week checklist rather than a single duration. WeekNumber is 1-based and
// matches the task's position in the plan. Completed/CompletedAt are the only
// fields that change after creation: the official checks a week off in public as
// the work is finished (a milestone, not a lifecycle transition — completing every
// task changes no status and does not itself resolve the case).
type PlanTask struct {
	ID          string
	PlanID      string
	WeekNumber  int
	Task        string
	Completed   bool
	CompletedAt *time.Time
	CreatedAt   time.Time
}

// NewPlanTask constructs a week's task (not yet completed).
func NewPlanTask(planID string, week int, task string) PlanTask {
	return PlanTask{
		ID:         idgen.New("task"),
		PlanID:     planID,
		WeekNumber: week,
		Task:       task,
		CreatedAt:  time.Now().UTC(),
	}
}
