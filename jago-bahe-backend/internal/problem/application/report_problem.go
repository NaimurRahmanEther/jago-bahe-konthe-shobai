package application

import (
	"context"

	"jago-bahe-backend/internal/problem/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// ReportProblemInput is the report use-case request.
type ReportProblemInput struct {
	Title             string
	Description       string
	AreaID            string
	Address           string
	Lat               *float64
	Lng               *float64
	ReporterID        string
	PointedOfficialID string
	ProposedSolution  string
	ImageURL          string
}

// ReportProblem records a new problem after checking its area and pointed
// official exist, and writes a "reported" audit entry.
type ReportProblem struct {
	problems domain.Repository
	areas    areadomain.Repository
	identity Identity
	audit    auditdomain.Repository
}

// NewReportProblem wires the use case.
func NewReportProblem(p domain.Repository, a areadomain.Repository, id Identity, au auditdomain.Repository) *ReportProblem {
	return &ReportProblem{problems: p, areas: a, identity: id, audit: au}
}

// Execute validates references, persists the problem, and audits.
func (uc *ReportProblem) Execute(ctx context.Context, in ReportProblemInput) (*domain.Problem, error) {
	if _, err := uc.areas.GetByID(ctx, valueobject.AreaID(in.AreaID)); err != nil {
		return nil, domain.ErrAreaNotFound
	}
	switch ok, err := uc.identity.OfficialExists(ctx, in.PointedOfficialID); {
	case err != nil:
		return nil, err
	case !ok:
		return nil, domain.ErrOfficialNotFound
	}

	loc := valueobject.Location{
		AreaID:  valueobject.AreaID(in.AreaID),
		Address: in.Address,
		Lat:     in.Lat,
		Lng:     in.Lng,
	}
	p := domain.NewProblem(in.Title, in.Description, loc, in.ReporterID, in.PointedOfficialID, in.ProposedSolution, in.ImageURL)
	if err := uc.problems.Create(ctx, p); err != nil {
		return nil, err
	}

	_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", p.ID, p.ReporterID, "reported", ""))
	return p, nil
}
