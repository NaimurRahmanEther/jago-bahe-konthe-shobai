package application_test

import (
	"context"
	"errors"
	"testing"

	"jago-bahe-backend/internal/resolution/application"
	"jago-bahe-backend/internal/resolution/domain"
)

// plannedCaseWithTasks is a case being worked, with a two-week plan checklist.
func plannedCaseWithTasks(status domain.Status) *domain.Case {
	plan := domain.NewPlan("case-1", "patch it", []string{"clear inlet", "lay pipe"}, "", "adopting", "", "")
	return &domain.Case{
		ID:         "case-1",
		ProblemID:  "prob-1",
		OfficialID: "off-1",
		Status:     status,
		Plan:       plan,
	}
}

func TestCompleteTaskMarksItDone(t *testing.T) {
	repo := newFakeRepo()
	c := plannedCaseWithTasks(domain.StatusInProgress)
	taskID := c.Plan.Tasks[0].ID
	repo.put(c)
	audit := &fakeAudit{}

	uc := application.NewCompleteTask(repo, audit)
	got, err := uc.Execute(context.Background(), "case-1", "off-1", taskID)
	if err != nil {
		t.Fatalf("complete task: %v", err)
	}
	// Only the named task flips; completing a task changes no case status.
	if !got.Plan.Tasks[0].Completed || got.Plan.Tasks[0].CompletedAt == nil {
		t.Fatalf("task 0 not completed: %+v", got.Plan.Tasks[0])
	}
	if got.Plan.Tasks[1].Completed {
		t.Fatal("task 1 should be untouched")
	}
	if got.Status != domain.StatusInProgress {
		t.Fatalf("status = %s, want unchanged InProgress", got.Status)
	}

	// Persisted, not just returned.
	saved, err := repo.GetByID(context.Background(), "case-1")
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	if !saved.Plan.Tasks[0].Completed {
		t.Fatal("completion not persisted")
	}
	if len(audit.actions) != 1 || audit.actions[0] != "task_completed" {
		t.Fatalf("audit actions = %v, want [task_completed]", audit.actions)
	}
}

func TestCompleteTaskGuards(t *testing.T) {
	valid := plannedCaseWithTasks(domain.StatusInProgress)
	knownTask := valid.Plan.Tasks[0].ID

	tests := []struct {
		name       string
		status     domain.Status
		officialID string
		taskID     string
		wantErr    error
	}{
		{"unknown task id", domain.StatusInProgress, "off-1", "task-nope", domain.ErrTaskNotFound},
		{"another official's case", domain.StatusInProgress, "off-2", knownTask, domain.ErrNotCaseOwner},
		// Done/Resolved are not workable — a checklist tick then makes no sense.
		{"a non-workable status", domain.StatusDone, "off-1", knownTask, domain.ErrIllegalTransition},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := newFakeRepo()
			c := plannedCaseWithTasks(tt.status)
			// Keep task ids stable across the sub-tests that reference knownTask.
			c.Plan.Tasks[0].ID = valid.Plan.Tasks[0].ID
			c.Plan.Tasks[1].ID = valid.Plan.Tasks[1].ID
			repo.put(c)

			uc := application.NewCompleteTask(repo, &fakeAudit{})
			_, err := uc.Execute(context.Background(), "case-1", tt.officialID, tt.taskID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("err = %v, want %v", err, tt.wantErr)
			}
		})
	}
}
