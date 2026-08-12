package application

import (
	"context"
	"strings"

	"jago-bahe-backend/internal/resolution/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
)

// SubmitPlan records the official's public plan. The plan must explicitly answer
// the community's top suggestion (SuggestionResponse required) and set a strategy.
// Legal only from Acknowledged → Planned.
type SubmitPlan struct {
	repo        domain.Repository
	suggestions Suggestions
	audit       auditdomain.Repository
}

// NewSubmitPlan wires the use case.
func NewSubmitPlan(r domain.Repository, s Suggestions, audit auditdomain.Repository) *SubmitPlan {
	return &SubmitPlan{repo: r, suggestions: s, audit: audit}
}

// Execute validates and stores the plan, snapshotting the top suggestion it
// answers so the pairing stays true after later upvotes move "top" elsewhere. The
// plan is a week-by-week checklist: tasks carries one non-empty task per week, and
// at least one is required (a plan with no weeks is not a plan).
func (uc *SubmitPlan) Execute(ctx context.Context, caseID, officialID, strategy string, tasks []string, obstacles, suggestionResponse string) (*domain.Case, error) {
	c, err := loadOwnedCase(ctx, uc.repo, caseID, officialID)
	if err != nil {
		return nil, err
	}
	if strings.TrimSpace(strategy) == "" || strings.TrimSpace(suggestionResponse) == "" {
		return nil, domain.ErrEmptyText // the plan must answer the top suggestion
	}
	cleaned := make([]string, 0, len(tasks))
	for _, tk := range tasks {
		if strings.TrimSpace(tk) == "" {
			return nil, domain.ErrEmptyText // every listed week must carry a task
		}
		cleaned = append(cleaned, strings.TrimSpace(tk))
	}
	if len(cleaned) == 0 {
		return nil, domain.ErrEmptyText // a plan needs at least one week
	}

	next, err := domain.Transition(c.Status, domain.EventPlan, domain.Guards{})
	if err != nil {
		return nil, err
	}

	// Snapshot what the official is answering, at the moment they answer it. A
	// problem with no upvoted suggestion has no top, and the response then stands
	// on its own — that is not an error, so a missing top never blocks a plan.
	top, err := uc.suggestions.Top(ctx, c.ProblemID)
	if err != nil {
		return nil, err
	}

	c.Status = next
	c.Plan = domain.NewPlan(c.ID, strategy, cleaned, obstacles, suggestionResponse, top.ID, top.Text)

	if err := uc.repo.Save(ctx, c); err != nil {
		return nil, err
	}
	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", c.ProblemID, officialID, "plan_submitted", ""))
	return c, nil
}
