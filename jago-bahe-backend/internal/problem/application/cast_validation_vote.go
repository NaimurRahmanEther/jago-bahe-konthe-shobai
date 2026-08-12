package application

import (
	"context"

	"jago-bahe-backend/internal/problem/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
	auditdomain "jago-bahe-backend/internal/shared/audit/domain"
	"jago-bahe-backend/pkg/idgen"
)

// CastValidationVote records a resident's validation and, when the distinct
// verified area validators reach the threshold V, flips the problem to Validated.
// Eligibility (verified resident of the problem's area) and distinctness are the
// B2 guardrails.
type CastValidationVote struct {
	problems  domain.Repository
	areas     areadomain.Repository
	identity  Identity
	audit     auditdomain.Repository
	svc       *domain.Service
	threshold int // V, from config
}

// NewCastValidationVote wires the use case with the configured threshold V.
func NewCastValidationVote(p domain.Repository, a areadomain.Repository, id Identity, au auditdomain.Repository, svc *domain.Service, threshold int) *CastValidationVote {
	return &CastValidationVote{problems: p, areas: a, identity: id, audit: au, svc: svc, threshold: threshold}
}

// Execute validates eligibility, records the vote, recomputes the distinct valid
// count, and flips the status at V. It returns the updated problem.
func (uc *CastValidationVote) Execute(ctx context.Context, problemID, voterID string, choice domain.VoteChoice) (*domain.Problem, error) {
	if !choice.Valid() {
		return nil, domain.ErrInvalidVote
	}

	p, err := uc.problems.GetByID(ctx, problemID)
	if err != nil {
		return nil, err
	}
	// A report awaiting screening is not votable — and, to a resident, does not
	// exist. Refusing with ErrProblemNotFound (not a distinct "pending" error)
	// keeps the vote endpoint from confirming that a hidden report exists at this
	// id, the same no-oracle rule get_problem enforces. Evaluate would refuse the
	// flip anyway (it only lifts Reported), but a resident must not even learn the
	// report is there.
	if !p.Status.PubliclyVisible() {
		return nil, domain.ErrProblemNotFound
	}
	// Note the absence of any further status guard. Voting deliberately continues
	// after a problem is assigned, and this is not an oversight to tidy away. Since
	// B17 an admin may forward a report before it reaches V, so the count that keeps
	// accruing afterwards is the public's verdict on that judgment: if the community
	// pushes a report the admin forwarded at 2 votes up to 7, the vindication belongs
	// on the record; if it stalls at 2, that stands as evidence too. A guard here
	// would mute the one signal that makes an early assignment judgeable. Evaluate
	// only lifts Reported, so no status is ever moved backwards by a late vote.

	voter, err := uc.identity.Voter(ctx, voterID)
	if err != nil {
		return nil, err
	}
	if !voter.IsResident || !voter.Verified {
		return nil, domain.ErrNotVerified
	}
	union, err := uc.unionOf(ctx, p.Location.AreaID.String())
	if err != nil {
		return nil, err
	}
	if voter.UnionID == "" || voter.UnionID != union {
		return nil, domain.ErrNotAreaResident
	}

	newVote := domain.ValidationVote{
		ID:        idgen.New("vote"),
		ProblemID: problemID,
		VoterID:   voterID,
		Vote:      choice,
	}
	if err := uc.problems.AddVote(ctx, newVote); err != nil {
		return nil, err // ErrAlreadyVoted on a repeat
	}

	votes, err := uc.problems.ListVotes(ctx, problemID)
	if err != nil {
		return nil, err
	}
	count := uc.svc.DistinctValidVoters(votes)
	status, flipped := uc.svc.Evaluate(p.Status, count, uc.threshold)

	if err := uc.problems.UpdateValidation(ctx, problemID, count, status); err != nil {
		return nil, err
	}
	if flipped {
		_ = uc.audit.Append(ctx, auditdomain.NewAuditEntry("problem", problemID, auditdomain.ActorSystem, "validated", ""))
	}

	p.ValidCount = count
	p.Status = status
	return p, nil
}

// unionOf resolves the union an area belongs to, mapping an unknown area onto
// this context's sentinel.
func (uc *CastValidationVote) unionOf(ctx context.Context, areaID string) (string, error) {
	union, err := areadomain.ResolveUnion(ctx, uc.areas, areaID)
	if err != nil {
		return "", domain.ErrAreaNotFound
	}
	return union, nil
}
