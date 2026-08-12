package domain

import (
	"time"

	"jago-bahe-backend/internal/shared/domain/valueobject"
	"jago-bahe-backend/pkg/idgen"
)

// Problem is the aggregate root: a reported local issue, its pointing, its
// moderation outcome, and its validation state. ValidCount is the denormalized
// number of distinct residents who marked it valid — the authority is still the
// votes, recomputed on each cast.
type Problem struct {
	ID                string
	Title             string
	Description       string
	Location          valueobject.Location
	ReporterID        string
	PointedOfficialID string
	ProposedSolution  string
	ImageURL          string
	Status            Status
	ValidCount        int
	CreatedAt         time.Time

	// Moderation outcome. Zero unless an admin has taken the problem down.
	// RejectionReason is set only when Status is Rejected, and is shown publicly
	// alongside the problem — a rejection is accountable, not a disappearance.
	// ReviewedBy is the deciding admin's account id.
	RejectionReason RejectionReason
	ReviewedBy      string
	ReviewedAt      *time.Time
}

// Editable reports whether the reporter may still edit the problem's content. The
// window is deliberately narrow: only while the problem is still Reported and no
// one has validated it yet. Once a validation vote lands, residents have endorsed
// the text they read, and letting the reporter rewrite it under them would make
// their endorsement meaningless — so the content freezes at the first valid vote.
//
// The lock keys on ValidCount (distinct valid voters), not on any vote at all: an
// "invalid" vote is a resident saying the report is wrong, which is a reason to let
// the reporter fix it, not a reason to freeze it. Only an endorsement is protected.
func (p *Problem) Editable() bool {
	return p.Status == StatusReported && p.ValidCount == 0
}

// NewProblem constructs a freshly reported problem: pending approval, unvotable,
// and with no votes yet. It is held invisible to everyone but its reporter and its
// union's admin until that admin approves it into public view (StatusReported).
// The community (V) is still the filter that decides merit — approval only clears
// the spam gate (see Status.PubliclyVisible and Screening).
func NewProblem(title, description string, loc valueobject.Location, reporterID, pointedOfficialID, proposedSolution, imageURL string) *Problem {
	return &Problem{
		ID:                idgen.New("prob"),
		Title:             title,
		Description:       description,
		Location:          loc,
		ReporterID:        reporterID,
		PointedOfficialID: pointedOfficialID,
		ProposedSolution:  proposedSolution,
		ImageURL:          imageURL,
		Status:            StatusPendingApproval,
		ValidCount:        0,
		CreatedAt:         time.Now().UTC(),
	}
}
