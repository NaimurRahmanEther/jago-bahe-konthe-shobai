package domain_test

import (
	"testing"
	"time"

	"jago-bahe-backend/internal/suggestion/domain"
)

// sug builds a suggestion with a deterministic CreatedAt (n minutes after a base
// instant) so tie-breaking by "oldest first" is testable.
func sug(id string, upvotes, ageMin int) domain.Suggestion {
	base := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return domain.Suggestion{
		ID:          id,
		UpvoteCount: upvotes,
		CreatedAt:   base.Add(time.Duration(ageMin) * time.Minute),
	}
}

func ids(sugs []domain.Suggestion) []string {
	out := make([]string, len(sugs))
	for i, s := range sugs {
		out[i] = s.ID
	}
	return out
}

func topIDs(sugs []domain.Suggestion) []string {
	var out []string
	for _, s := range sugs {
		if s.IsTop {
			out = append(out, s.ID)
		}
	}
	return out
}

func eq(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if a[i] != b[i] {
			return false
		}
	}
	return true
}

func TestRank(t *testing.T) {
	svc := domain.NewService()
	tests := []struct {
		name      string
		in        []domain.Suggestion
		wantOrder []string
		wantTops  []string
	}{
		{"empty", nil, []string{}, nil},
		{
			"single with upvotes is top",
			[]domain.Suggestion{sug("a", 1, 0)},
			[]string{"a"}, []string{"a"},
		},
		{
			"zero upvotes never top",
			[]domain.Suggestion{sug("a", 0, 0), sug("b", 0, 1)},
			[]string{"a", "b"}, nil,
		},
		{
			"clear winner ordered first and only top",
			[]domain.Suggestion{sug("a", 1, 0), sug("b", 4, 1), sug("c", 2, 2)},
			[]string{"b", "c", "a"}, []string{"b"},
		},
		{
			"tie at max marks both top, oldest first",
			[]domain.Suggestion{sug("new", 3, 5), sug("old", 3, 1), sug("low", 1, 2)},
			[]string{"old", "new", "low"}, []string{"old", "new"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := svc.Rank(tt.in)
			if order := ids(got); !eq(order, tt.wantOrder) {
				t.Fatalf("Rank order = %v, want %v", order, tt.wantOrder)
			}
			if tops := topIDs(got); !eq(tops, tt.wantTops) {
				t.Fatalf("Rank tops = %v, want %v", tops, tt.wantTops)
			}
		})
	}
}

func TestTop(t *testing.T) {
	svc := domain.NewService()
	tests := []struct {
		name   string
		in     []domain.Suggestion
		wantID string
		wantOK bool
	}{
		{"empty has no top", nil, "", false},
		{"all zero upvotes has no top", []domain.Suggestion{sug("a", 0, 0)}, "", false},
		{"highest wins", []domain.Suggestion{sug("a", 2, 0), sug("b", 5, 1)}, "b", true},
		{"tie resolves to oldest", []domain.Suggestion{sug("new", 3, 5), sug("old", 3, 1)}, "old", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := svc.Top(tt.in)
			if ok != tt.wantOK || got.ID != tt.wantID {
				t.Fatalf("Top() = (%q, %v), want (%q, %v)", got.ID, ok, tt.wantID, tt.wantOK)
			}
		})
	}
}
