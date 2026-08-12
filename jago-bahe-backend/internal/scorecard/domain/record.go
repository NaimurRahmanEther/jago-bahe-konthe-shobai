package domain

import "time"

// CaseRecord is one entry in an official's public record: a case they hold, the
// plan they published for it, and the community question that plan answers.
//
// AnsweredSuggestion is the snapshot the plan stored when it was submitted, not
// today's top suggestion. Pairing a response with a question the official was
// never asked would misrepresent a named person on a public page — the thing
// Concept §10's legal-care rule exists to prevent.
type CaseRecord struct {
	CaseID    string
	ProblemID string
	Title     string
	Status    string
	CreatedAt time.Time
	Deadline  time.Time

	// Acknowledged/AcknowledgedAt describe the first response. Silence is itself a
	// fact worth recording, so a case with no acknowledgement is still listed.
	Acknowledged   bool
	AcknowledgedAt *time.Time

	// Plan fields; HasPlan is false when the official has not published one yet.
	HasPlan            bool
	Strategy           string
	TimelineWeeks      int
	Obstacles          string
	SuggestionResponse string
	AnsweredSuggestion string
	PlanCreatedAt      *time.Time

	// BlockedOnHigherAuthority marks a case waiting on a confirmed blocker. The
	// same fairness rule the scorecard applies: this is not the official's failure,
	// and the public view must say so rather than let it read as neglect.
	BlockedOnHigherAuthority bool
}

// OfficialRecord is the public, factual account of one official's work. It is
// deliberately a list and a set of stats — never a grade, a rank, or a
// comparison against other officials (Concept §10).
type OfficialRecord struct {
	OfficialID string
	Cases      []CaseRecord
}

// BuildRecord assembles the public record. It applies no judgement of its own:
// ordering is newest-first so a reader sees current work, and the fairness flag
// is projected from the same adjudication rule the scorecard uses.
func BuildRecord(officialID string, cases []CaseRecord) OfficialRecord {
	return OfficialRecord{OfficialID: officialID, Cases: cases}
}

// FairnessFlag reports whether a case's status and blocker adjudication mean it
// is waiting on a higher authority — the one nuance the public record owes an
// official, so a case they are blocked on does not read as silence.
//
// It mirrors classify()'s blocked branch exactly: only a blocker the named
// authority judged *real* protects them; a pending or denied one does not.
func FairnessFlag(status string, adj Adjudication) bool {
	return status == statusBlocked && adj == AdjConfirmed
}
