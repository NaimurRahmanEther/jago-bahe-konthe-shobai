package application

import (
	"context"
	"time"

	"jago-bahe-backend/internal/resolution/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
	notificationdomain "jago-bahe-backend/internal/shared/notification/domain"
)

// AdjudicateObstacle records the named higher authority's (or a moderator's)
// verdict on a blocker — the decision that sets the scorecard consequence, not
// the advisory public vote. Confirm → responsibility sits with the authority, the
// official is protected, the case stays Blocked. Deny → the blocker closes, the
// case resumes (InProgress), the clock restarts, and it may count against the
// official.
type AdjudicateObstacle struct {
	repo     domain.Repository
	problems Problems
	audit    auditdomain.Repository
	notify   notificationdomain.Repository
}

// NewAdjudicateObstacle wires the use case.
func NewAdjudicateObstacle(r domain.Repository, p Problems, audit auditdomain.Repository, n notificationdomain.Repository) *AdjudicateObstacle {
	return &AdjudicateObstacle{repo: r, problems: p, audit: audit, notify: n}
}

// Execute applies the verdict. confirm=true accepts the obstacle as genuine.
func (uc *AdjudicateObstacle) Execute(ctx context.Context, obstacleID, adjudicatorID string, confirm bool) (*domain.Obstacle, error) {
	c, err := uc.repo.GetCaseByObstacle(ctx, obstacleID)
	if err != nil {
		return nil, err // domain.ErrObstacleNotFound
	}
	blocker := findObstacle(c, obstacleID)
	if blocker == nil {
		return nil, domain.ErrObstacleNotFound
	}
	if blocker.Adjudication != domain.AdjudicationPending {
		return nil, domain.ErrAlreadyAdjudicated
	}

	// B21: one notification type, two outcomes discriminated by Detail. The official
	// has no surface that tells them either happened, and the consequences are
	// opposite — confirm protects them on the scorecard and moves responsibility up,
	// deny bounces the work straight back with the clock restarted.
	//
	// Recipient is the ASSIGNEE (a directory office id), never the adjudicator.
	// Entitlement: the case is theirs and they declared the obstacle.
	notifyVerdict := func(detail string) {
		_ = uc.notify.Append(ctx, notificationdomain.ForOfficial(
			c.OfficialID, notificationdomain.TypeObstacleAdjudicated, c.ProblemID,
			notificationTitle(ctx, uc.problems, c.ProblemID), detail))
	}

	now := time.Now().UTC()
	if confirm {
		if err := uc.repo.UpdateObstacleAdjudication(ctx, obstacleID, domain.AdjudicationConfirmed, now, nil); err != nil {
			return nil, err
		}
		_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", c.ProblemID, adjudicatorID, "blocker_confirmed", blocker.WhoUnblocks))
		notifyVerdict("confirmed")
	} else {
		// Deny: the obstacle is within the official's power — bounce it back. Persist
		// the resumed case status first, then record the verdict last so it is never
		// clobbered by the aggregate save.
		next, err := domain.Transition(c.Status, domain.EventProgress, domain.Guards{})
		if err != nil {
			return nil, err
		}
		c.Status = next
		if err := uc.repo.Save(ctx, c); err != nil {
			return nil, err
		}
		if err := uc.repo.UpdateObstacleAdjudication(ctx, obstacleID, domain.AdjudicationDenied, now, &now); err != nil {
			return nil, err
		}
		_ = uc.problems.SetStatus(ctx, c.ProblemID, string(c.Status))
		_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", c.ProblemID, adjudicatorID, "blocker_denied", ""))
		notifyVerdict("denied")
	}

	// Reload so the returned blocker carries the recorded verdict.
	reloaded, err := uc.repo.GetCaseByObstacle(ctx, obstacleID)
	if err != nil {
		return nil, err
	}
	return findObstacle(reloaded, obstacleID), nil
}

// findObstacle returns the blocker with the given id within a case, or nil.
func findObstacle(c *domain.Case, obstacleID string) *domain.Obstacle {
	for i := range c.Obstacles {
		if c.Obstacles[i].ID == obstacleID {
			return &c.Obstacles[i]
		}
	}
	return nil
}
