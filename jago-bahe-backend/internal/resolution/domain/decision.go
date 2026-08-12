package domain

// Decision is an official's response to a new assignment.
//
// It is a typed value rather than a bool on purpose. The bool it replaced was
// derived as `decision == "accept"`, which made *dispute* the silent default for
// every unrecognised input — a typo, a missing field, an empty body. Disputing
// removes a case from the official's public scorecard (the fairness rule treats a
// disputed case as not theirs), so the failure mode was a named person's record
// quietly losing a case. With a closed enum, an unrecognised decision is an error
// and neither branch is reachable by accident.
type Decision string

const (
	DecisionAccept  Decision = "accept"
	DecisionDispute Decision = "dispute"
)

// Valid reports whether d is one of the two permitted decisions.
func (d Decision) Valid() bool {
	switch d {
	case DecisionAccept, DecisionDispute:
		return true
	}
	return false
}
