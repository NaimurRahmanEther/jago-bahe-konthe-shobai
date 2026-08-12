package application

import (
	"context"
	"strings"

	areadomain "jago-bahe-backend/internal/shared/area/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
	"jago-bahe-backend/internal/suggestion/domain"
)

// ProposeSuggestion records a resident's proposed fix for a problem and writes a
// "suggestion_proposed" audit entry on the problem's public trail. Only a
// verified resident of the problem's area may propose (the B3 guardrail).
type ProposeSuggestion struct {
	suggestions domain.Repository
	areas       areadomain.Repository
	problems    Problems
	identity    Identity
	audit       auditdomain.Repository
}

// NewProposeSuggestion wires the use case.
func NewProposeSuggestion(s domain.Repository, a areadomain.Repository, pr Problems, id Identity, au auditdomain.Repository) *ProposeSuggestion {
	return &ProposeSuggestion{suggestions: s, areas: a, problems: pr, identity: id, audit: au}
}

// Execute validates eligibility and text, persists the suggestion, and audits.
func (uc *ProposeSuggestion) Execute(ctx context.Context, problemID, authorID, text string) (*domain.Suggestion, error) {
	text = strings.TrimSpace(text)
	if text == "" {
		return nil, domain.ErrEmptyText
	}
	if err := ensureAreaResident(ctx, uc.identity, uc.areas, uc.problems, problemID, authorID); err != nil {
		return nil, err
	}

	s := domain.NewSuggestion(problemID, authorID, text)
	if err := uc.suggestions.Create(ctx, s); err != nil {
		return nil, err
	}

	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", problemID, authorID, "suggestion_proposed", ""))
	return s, nil
}
