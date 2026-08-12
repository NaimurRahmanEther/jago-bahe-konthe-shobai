package application

import "context"

// The audit context owns the trail and nothing else. It knows an actor is some
// id and a target is some id; it does not know what an account or a problem is,
// and it must not import identity or problem to find out (A.4.1 — dependencies
// point inward, and a shared context reaching sideways into two others is how
// that rule dies quietly).
//
// So it declares what it needs as ports and the composition root supplies the
// adapters, mirroring resolution's Suggestions port (resolution/application/
// ports.go), which is how a plan learns the text of the suggestion it answered.
//
// Both ports are BATCH, one query per page, per A.3.4. A per-row lookup on a
// hundred-entry feed would be two hundred queries to render one page.

// Actors resolves audit actor ids to display names.
//
// The actor of an entry is not always an account: an official's action is
// recorded against their DIRECTORY OFFICE id (`off-chair-aranagar`), and a change
// the platform made by rule is `audit.ActorSystem`. An implementation resolves
// what it can and simply omits the rest — a name it cannot find is not an error,
// and the handler falls back to showing the id.
type Actors interface {
	NamesByIDs(ctx context.Context, ids []string) (map[string]string, error)
}

// Problems resolves problem ids to titles, so a row reads as a sentence about a
// report rather than an opaque id.
//
// Only PUBLICLY VISIBLE problems are resolved. That is a safety property, not an
// optimisation: it means the feed can never print the title of a report still
// awaiting screening, even if an action were somehow logged against one. A title
// that does not resolve is omitted and the row still renders.
type Problems interface {
	TitlesByIDs(ctx context.Context, ids []string) (map[string]string, error)
}
