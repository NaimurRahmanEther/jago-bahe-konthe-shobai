package domain

import (
	"context"
	"errors"
	"time"
)

// Sentinel errors mapped to HTTP status by the http layer.
var (
	ErrProblemNotFound  = errors.New("problem not found")
	ErrAlreadyVoted     = errors.New("already voted on this problem")
	ErrNotVerified      = errors.New("resident is not verified")
	ErrNotAreaResident  = errors.New("resident is not in this problem's area")
	ErrInvalidVote      = errors.New("invalid vote choice")
	ErrOfficialNotFound = errors.New("pointed official not found")
	ErrAreaNotFound     = errors.New("area not found")

	// Moderation. ErrNotRejectable covers both halves of the same fact: the
	// problem is past the takedown window (Status.Rejectable), or it moved out of
	// that window under a racing admin between the load and the write.
	ErrNotRejectable          = errors.New("problem can no longer be rejected")
	ErrInvalidRejectionReason = errors.New("invalid rejection reason")
	ErrNotUnionAdmin          = errors.New("admin does not belong to this problem's union")
	ErrStatusNotPublic        = errors.New("status is not publicly listable")

	// ErrNotPending is returned when an approve is attempted on a problem that is
	// not awaiting screening (it was already approved, rejected, or has moved on).
	ErrNotPending = errors.New("problem is not pending approval")

	// ErrNoCaller guards the reporter's own view (B12): it is derived from the
	// token, so an absent caller is an authentication failure, never an empty
	// listing. See Filter.ReporterID for why failing closed matters here.
	ErrNoCaller = errors.New("no authenticated caller")

	// Reporter self-management: a reporter may edit or withdraw their own problem,
	// within a narrow window, and no one else may.
	//
	// ErrNotReporter is a 403 — the caller is authenticated but is not this
	// problem's reporter. ErrNotEditable and ErrNotWithdrawable are 409s: the
	// caller is the reporter, but the problem has moved past the window (a vote
	// landed, or it was assigned). Both windows are also enforced as a CAS in the
	// write query, so a vote or assignment racing the load surfaces the same error
	// rather than silently overwriting.
	ErrNotReporter     = errors.New("caller is not this problem's reporter")
	ErrNotEditable     = errors.New("problem can no longer be edited")
	ErrNotWithdrawable = errors.New("problem can no longer be withdrawn")
)

// Filter narrows a problem listing. Empty fields are ignored.
type Filter struct {
	AreaID     string
	OfficialID string

	// ReporterID scopes the listing to one person's own reports. Empty means
	// unset, like the fields above. There is deliberately no route that lets a
	// request choose this value (Scaffold §5, "There is no reporter filter") — it
	// would be an index of everything a named person has filed, and in a
	// politically sensitive civic platform reporter identity is a safety matter,
	// not a preference.
	ReporterID string

	// Statuses restricts the listing to these states. Nil means no status
	// restriction. Nothing is hidden from the public feed any more, so nil is no
	// longer a leak — but callers serving the public should still pass
	// PublicStatuses() (or a validated subset) so that a future hidden state has
	// one place to be excluded rather than every caller to be audited.
	Statuses []Status
}

// Repository is the port for problem persistence. It also owns the problem's
// validation votes (part of the aggregate).
type Repository interface {
	Create(ctx context.Context, p *Problem) error
	GetByID(ctx context.Context, id string) (*Problem, error)
	List(ctx context.Context, f Filter) ([]Problem, error)

	// AddVote records a validation vote, returning ErrAlreadyVoted if this voter
	// already voted on this problem (distinctness invariant).
	AddVote(ctx context.Context, v ValidationVote) error
	ListVotes(ctx context.Context, problemID string) ([]ValidationVote, error)

	// VotesByViewer returns one viewer's own votes across the given problems, keyed
	// by problem id. Problems the viewer has not voted on have no entry, and an
	// empty viewerID returns an empty map without touching the database.
	//
	// This answers "have I already voted on this?" for a whole page of problems in
	// one round trip. It is deliberately viewer-scoped rather than a general vote
	// lookup: the counts are public, but who voted which way is not something any
	// caller should be able to ask about anyone else.
	VotesByViewer(ctx context.Context, viewerID string, problemIDs []string) (map[string]VoteChoice, error)

	// TitlesByIDs resolves problem ids to titles for the public activity feed,
	// one query for a whole page (A.3.4).
	//
	// PUBLICLY VISIBLE PROBLEMS ONLY. That is a safety property rather than a
	// filter for tidiness: it makes it structurally impossible for the seat-wide
	// feed to print the title of a report still awaiting its admin's screening,
	// without the caller having to remember to check. An id that does not resolve
	// is absent from the map, and the caller renders the row without a title.
	TitlesByIDs(ctx context.Context, problemIDs []string) (map[string]string, error)

	// UpdateValidation persists a recomputed valid count and (possibly flipped) status.
	UpdateValidation(ctx context.Context, problemID string, validCount int, status Status) error

	// SetStatus moves a problem to a new lifecycle state (used by later contexts,
	// e.g. assignment flips Validated -> Assigned). Returns ErrProblemNotFound if
	// the problem does not exist.
	SetStatus(ctx context.Context, problemID string, status Status) error

	// Update persists a reporter's edit to the problem's content (title,
	// description, address/coordinates, proposed solution). The editable window is
	// part of the statement — status = 'Reported' AND valid_count = 0 — so a
	// validation vote landing between the caller's load and this write causes the
	// update to affect no rows and returns ErrNotEditable, rather than silently
	// rewriting a report someone has already endorsed.
	Update(ctx context.Context, p *Problem) error

	// SetWithdrawn takes a problem down at its reporter's request. Like
	// SetScreening it is a compare-and-set on the source status, so a problem
	// assigned out from under the reporter between load and write keeps its new
	// status and the caller gets ErrNotWithdrawable.
	SetWithdrawn(ctx context.Context, problemID string, from Status) error

	// SetScreening records a moderation decision atomically: the resulting status
	// (Rejected), the ground, and which admin decided when. The status guard is
	// part of the statement so two admins racing on the same problem cannot both
	// win, and so a problem assigned out from under the admin is not taken down.
	SetScreening(ctx context.Context, problemID string, from, to Status, reason RejectionReason, reviewedBy string, reviewedAt time.Time) error

	// Delete permanently removes the problem row. This is not a status change and
	// there is no recovery.
	//
	// Every dependent row goes with it, by ON DELETE CASCADE declared in the
	// migrations: validation votes (000006), suggestions (000007), the assignment
	// and its admin votes (000008), the case and — through it — the plan, progress
	// updates, blockers and evidence (000010), and confirmations (000011). On a
	// problem that reached an official, that is the erasure of work published in
	// public, and it silently changes that official's scorecard, because the
	// scorecard's case facts are computed from the `cases` rows this removes.
	//
	// There is deliberately no status guard: the reporter may delete at any point
	// in the lifecycle. See CLAUDE.md A.3.3 for the decision and its cost.
	Delete(ctx context.Context, problemID string) error
}
