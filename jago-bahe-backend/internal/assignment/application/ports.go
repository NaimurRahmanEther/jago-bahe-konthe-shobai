// Package application holds the assignment use cases. It orchestrates the
// assignment domain (routing + vote arithmetic), the shared area geography, the
// audit log, and three small ports — Problems, Officials, and Admins — that
// expose only the facts the assignment context needs from the problem, identity,
// and admin sides, keeping the contexts decoupled (the pattern the problem and
// suggestion contexts established).
package application

import (
	"context"

	"jago-bahe-backend/internal/assignment/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// ProblemView is the subset of a problem the assignment context needs.
type ProblemView struct {
	ID                string
	Title             string
	Status            string
	Address           string
	PointedOfficialID string
	AreaID            string

	// ReporterID is the account that filed the report. It travels so the shared
	// assign path can tell the reporter, in one place, which official their report
	// went to — the reporter has no other surface that says so, and being handed to
	// a named person is the moment their report stops being a queue entry (B21).
	//
	// Like the fields below it is supplied by the adapter, not derived here. B17
	// added ValidCount/Address/Routing for the same kind of reason; that is the
	// precedent for widening this struct rather than reaching into problem.
	ReporterID string

	// ValidCount is how many distinct verified residents have validated the
	// problem. Since B17 it does not gate anything — it is the evidence the admin
	// weighs when deciding whether to forward the report, and it is shown to them
	// in the queue. Display and judgment only; never a precondition here.
	ValidCount int

	// EligibleForAssignment is problemdomain.Status.EligibleForAssignment, computed
	// by the adapter in the composition root. It travels as a bool rather than
	// being re-derived here because the assignment context must not import the
	// problem package (A.4.1) — the status vocabulary has one owner, and this
	// context previously duplicated it as a local string const, which is exactly
	// how the two drifted.
	EligibleForAssignment bool

	// Routing is "union" or "super_admin" — which decision path this problem's pointed
	// tier requires. Unlike the fields above it is NOT supplied by the Problems
	// port; ListQueue fills it per row from domain.Service.Route, because routing
	// is this context's own authority (A.4.6). The client must never re-derive it:
	// before B17 the queue DTO omitted it entirely, so the UI read `undefined` and
	// sent every above-union problem to the union-assign screen, where the backend
	// then refused it with ErrWrongRoute.
	Routing string
}

// Problems is the port into the problem context.
type Problems interface {
	// Get returns the problem, or domain.ErrProblemNotFound if absent.
	Get(ctx context.Context, problemID string) (ProblemView, error)
	// ListAssignable returns the problems an admin may still forward to an
	// official — public, and not yet in a case. Since B17 that is Reported as well
	// as Validated: a report is forwardable as soon as its admin approves it, and
	// the validation count travels alongside so the admin can judge it. Assigning
	// moves a problem out of both statuses, so it drops off the queue either way.
	ListAssignable(ctx context.Context) ([]ProblemView, error)
	// MarkAssigned flips a problem (Reported or Validated) -> Assigned.
	MarkAssigned(ctx context.Context, problemID string) error
}

// OfficialView describes a directory official for routing and monitor decisions.
type OfficialView struct {
	ID     string
	Tier   valueobject.Tier
	AreaID string

	// Name is DISPLAY TEXT for the notification that tells a reporter who their
	// report went to (B21) — never a routing fact. Nothing in this context branches
	// on it, and nothing should: routing is decided from Tier.
	Name string
}

// Officials is the port into the identity officials directory.
type Officials interface {
	// Get returns the official, or domain.ErrOfficialNotFound if absent.
	Get(ctx context.Context, officialID string) (OfficialView, error)
	// MonitorFor returns the official who monitors the given official (the tier up
	// the ladder), or "" when the official is at the top (MP / minister).
	MonitorFor(ctx context.Context, officialID string) (string, error)
}

// Admins is the port into the union admin set.
type Admins interface {
	// UnionOf returns the union an admin belongs to (via their linked official),
	// used to enforce that only that union's admin decides a union-level problem.
	UnionOf(ctx context.Context, adminAccountID string) (string, error)
	// EligibleAdvisers returns the admin account IDs entitled to advise on an
	// above-union report in the given scope, for a problem located in the given
	// area. It was EligibleVoters until B20; the SET is unchanged — who has
	// standing to speak on a problem did not change when the decision moved to the
	// super admin, only what their input does.
	EligibleAdvisers(ctx context.Context, scope domain.AdviceScope, problemAreaID string) ([]string, error)
}
