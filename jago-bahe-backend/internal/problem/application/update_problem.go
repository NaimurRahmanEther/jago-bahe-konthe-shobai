package application

import (
	"context"

	"jago-bahe-backend/internal/problem/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
)

// UpdateProblemInput is the reporter's edit request. Only the content fields a
// reporter may change are here: area and pointed official are fixed at filing (they
// set jurisdiction and the accountable official), so they are absent by design.
type UpdateProblemInput struct {
	ProblemID        string
	CallerID         string
	Title            string
	Description      string
	Address          string
	Lat              *float64
	Lng              *float64
	ProposedSolution string
}

// UpdateProblem lets a reporter edit their own problem while it is still Reported
// and unvalidated. Two guards stand in front of the write: the caller must be the
// reporter, and the problem must still be Editable. The domain owns both windows,
// and the repository enforces the editable window a second time as a CAS so a vote
// racing the edit cannot slip a rewrite past an endorsement.
type UpdateProblem struct {
	problems domain.Repository
	audit    auditdomain.Repository
}

// NewUpdateProblem wires the use case.
func NewUpdateProblem(p domain.Repository, au auditdomain.Repository) *UpdateProblem {
	return &UpdateProblem{problems: p, audit: au}
}

// Execute applies the edit and writes an "edited" audit entry. It returns
// ErrNotReporter if the caller does not own the problem and ErrNotEditable if the
// problem has left the editable window.
func (uc *UpdateProblem) Execute(ctx context.Context, in UpdateProblemInput) (*domain.Problem, error) {
	p, err := uc.problems.GetByID(ctx, in.ProblemID)
	if err != nil {
		return nil, err
	}
	// Ownership before window: a stranger is told they are not the reporter (403),
	// not shown the window state of a problem that was never theirs to manage.
	if in.CallerID == "" || p.ReporterID != in.CallerID {
		return nil, domain.ErrNotReporter
	}
	if !p.Editable() {
		return nil, domain.ErrNotEditable
	}

	p.Title = in.Title
	p.Description = in.Description
	p.Location.Address = in.Address
	p.Location.Lat = in.Lat
	p.Location.Lng = in.Lng
	p.ProposedSolution = in.ProposedSolution

	if err := uc.problems.Update(ctx, p); err != nil {
		return nil, err
	}
	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", p.ID, in.CallerID, "edited", ""))
	return p, nil
}
