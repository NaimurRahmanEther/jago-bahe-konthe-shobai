// Package application holds the resolution use cases: the official's case
// lifecycle (acknowledge → plan → update → evidence → done, plus dispute and the
// typed-block path) and the advisory obstacle-judgment sub-slice. It orchestrates
// the resolution domain, the shared area geography, the audit log, and three
// ports — Assignments, Problems, and Identity — so the context stays decoupled
// from the assignment, problem, and identity contexts.
package application

import (
	"context"
	"time"
)

// AssignmentView is the subset of an assignment the resolution context needs to
// materialize a case.
type AssignmentView struct {
	ProblemID         string
	OfficialID        string
	MonitorOfficialID string
	Deadline          time.Time
}

// Assignments is the port into the assignment context.
type Assignments interface {
	// Get returns the assignment for a problem, or domain.ErrAssignmentMissing.
	Get(ctx context.Context, problemID string) (AssignmentView, error)
	// ListByOfficial returns every assignment held by the official.
	ListByOfficial(ctx context.Context, officialID string) ([]AssignmentView, error)
}

// Problems is the port into the problem context: it resolves a problem's area
// (for the obstacle-judgment residency check) and mirrors the case's lifecycle
// onto the public problem status.
type Problems interface {
	// AreaID returns the problem's area, or domain.ErrProblemNotFound.
	AreaID(ctx context.Context, problemID string) (string, error)
	// ReporterID returns the account that reported the problem (the confirm
	// guardrail: only the reporting resident may confirm), or ErrProblemNotFound.
	ReporterID(ctx context.Context, problemID string) (string, error)
	// SetStatus mirrors the case state onto the problem (InProgress, Blocked,
	// Done, ...), keeping the public problem record in step with the case.
	SetStatus(ctx context.Context, problemID, status string) error
	// TitlesByIDs resolves problem ids to titles in one query, so a monitor's list
	// of observed cases can say what each one is about (A.3.4). It resolves
	// publicly visible problems only — a report still awaiting screening can never
	// surface a title here.
	TitlesByIDs(ctx context.Context, problemIDs []string) (map[string]string, error)
}

// Officials is the port into the identity directory. It exists so an observation
// list can name the official who has gone silent instead of showing a raw id — one
// batch query for the page, never one per row (A.3.4).
type Officials interface {
	NamesByIDs(ctx context.Context, officialIDs []string) (map[string]string, error)
}

// Ladder is the port onto the accountability ladder: who monitors whom, nearest
// rung first. The walk itself belongs to identity (which owns the officials
// directory and the geography it is read against), so the adapter delegates to
// identityapp.MonitorChain rather than restating the rule — the assignment's
// monitor and the escalation's observers must never be able to disagree about who
// supervises whom.
type Ladder interface {
	Chain(ctx context.Context, officialID string, maxRungs int) ([]string, error)
}

// Voter is the subset of an account the obstacle-judgment sub-slice needs to
// decide who may vote/propose/upvote (verified area residents only).
type Voter struct {
	IsResident bool
	Verified   bool
	UnionID    string
}

// Identity is the port into the identity context.
type Identity interface {
	Voter(ctx context.Context, accountID string) (Voter, error)
}

// TopSuggestion is the community's most-supported fix for a problem, as it stood
// at a moment in time. Found reports whether one exists at all — a problem whose
// suggestions have no upvotes has no top (the B3 ranking rule), and an official
// then has nothing to answer.
type TopSuggestion struct {
	ID    string
	Text  string
	Found bool
}

// Suggestions is the port into the suggestion context. It exists so a plan can
// snapshot the suggestion it answered at submit time.
//
// The ranking rule is NOT reimplemented behind this port: the adapter delegates
// to the suggestion domain's own Top(), which is the single authority (B3:
// "ranking and top selection are backend-owned"). "Top" is derived from live
// upvotes and therefore moves, so reading it later would pair an official's
// answer with whatever question happens to lead today — showing them, on their
// own public record, answering something they were never asked.
type Suggestions interface {
	// Top returns the problem's current top suggestion, if any.
	Top(ctx context.Context, problemID string) (TopSuggestion, error)
}
