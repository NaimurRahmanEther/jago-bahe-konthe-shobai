package domain_test

import (
	"testing"

	"jago-bahe-backend/internal/problem/domain"
)

func vote(voter string, choice domain.VoteChoice) domain.ValidationVote {
	return domain.ValidationVote{VoterID: voter, Vote: choice}
}

func TestDistinctValidVoters(t *testing.T) {
	svc := domain.NewService()
	tests := []struct {
		name  string
		votes []domain.ValidationVote
		want  int
	}{
		{"none", nil, 0},
		{"one valid", []domain.ValidationVote{vote("a", domain.VoteValid)}, 1},
		{"invalid does not count", []domain.ValidationVote{vote("a", domain.VoteInvalid)}, 0},
		{
			"duplicate voter counted once",
			[]domain.ValidationVote{vote("a", domain.VoteValid), vote("a", domain.VoteValid)},
			1,
		},
		{
			"mixed",
			[]domain.ValidationVote{
				vote("a", domain.VoteValid),
				vote("b", domain.VoteValid),
				vote("c", domain.VoteInvalid),
				vote("b", domain.VoteValid),
			},
			2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := svc.DistinctValidVoters(tt.votes); got != tt.want {
				t.Fatalf("DistinctValidVoters() = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestEvaluate(t *testing.T) {
	svc := domain.NewService()
	const v = 5
	tests := []struct {
		name        string
		current     domain.Status
		distinct    int
		wantStatus  domain.Status
		wantFlipped bool
	}{
		{"below threshold stays reported", domain.StatusReported, 4, domain.StatusReported, false},
		{"at threshold flips", domain.StatusReported, 5, domain.StatusValidated, true},
		{"above threshold flips", domain.StatusReported, 7, domain.StatusValidated, true},
		{"already validated is untouched", domain.StatusValidated, 9, domain.StatusValidated, false},
		{"assigned is untouched", domain.StatusAssigned, 9, domain.StatusAssigned, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, flipped := svc.Evaluate(tt.current, tt.distinct, v)
			if got != tt.wantStatus || flipped != tt.wantFlipped {
				t.Fatalf("Evaluate() = (%s, %v), want (%s, %v)", got, flipped, tt.wantStatus, tt.wantFlipped)
			}
		})
	}
}
