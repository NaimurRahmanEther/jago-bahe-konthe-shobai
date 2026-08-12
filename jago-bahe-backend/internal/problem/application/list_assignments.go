package application

import "context"

// ListAssignments answers "who is working on each of these problems?" for a whole
// page, in one round trip.
//
// It exists so the public feed can name the official actually handling each report
// — the platform's central promise is that officials work in public, and a feed
// that shows only what was reported, never who took it up, keeps the accountable
// half of that private. The field used to be detail-only precisely because the
// per-row alternative is an N+1; this is the batch that makes it affordable, the
// same shape ListMyVotes uses for myVote.
//
// Like ListMyVotes it decorates rows the feed already selected. It never adds,
// removes or reorders one, and it takes no caller: an assignment is public, so
// there is nothing here that varies by who is reading.
type ListAssignments struct {
	assignments Assignments
}

// NewListAssignments wires the use case.
func NewListAssignments(a Assignments) *ListAssignments {
	return &ListAssignments{assignments: a}
}

// Execute returns the assignment per problem id, omitting problems that have none.
// An empty id list yields an empty map and no query.
func (uc *ListAssignments) Execute(ctx context.Context, problemIDs []string) (map[string]AssignmentView, error) {
	if len(problemIDs) == 0 {
		return map[string]AssignmentView{}, nil
	}
	return uc.assignments.ForMany(ctx, problemIDs)
}
