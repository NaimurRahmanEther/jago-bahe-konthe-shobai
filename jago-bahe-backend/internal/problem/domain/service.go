package domain

// Service holds the validation rule that spans a problem and its votes. It is
// the authority for the "distinct verified area validators → Validated at V"
// guardrail; the http/UI layers only display the outcome.
type Service struct{}

// NewService constructs the problem domain service.
func NewService() *Service { return &Service{} }

// DistinctValidVoters counts the unique residents who marked the problem valid.
// Invalid votes and duplicate voters do not add to the count.
func (Service) DistinctValidVoters(votes []ValidationVote) int {
	seen := make(map[string]struct{})
	for _, v := range votes {
		if v.Vote == VoteValid {
			seen[v.VoterID] = struct{}{}
		}
	}
	return len(seen)
}

// Evaluate returns the status a problem should hold given the distinct valid
// count and threshold V, and whether it changed. Only a Reported problem flips
// to Validated (at or above V); every other state is left untouched.
func (s Service) Evaluate(current Status, distinctValid, threshold int) (Status, bool) {
	if current == StatusReported && distinctValid >= threshold {
		return StatusValidated, true
	}
	return current, false
}
