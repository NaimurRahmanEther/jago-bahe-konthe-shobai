package domain_test

import (
	"testing"

	"jago-bahe-backend/internal/assignment/domain"
)

func suggestions(officialIDs ...string) []domain.Suggestion {
	out := make([]domain.Suggestion, 0, len(officialIDs))
	for i, id := range officialIDs {
		out = append(out, domain.Suggestion{
			ID:                  "sug-" + id,
			ProblemID:           "prob-1",
			AdminAccountID:      "admin-" + string(rune('a'+i)),
			SuggestedOfficialID: id,
		})
	}
	return out
}

// TestTopSuggestion pins the advisory tally. The tie case is the load-bearing
// one: an arbitrary winner would make the reason requirement turn on a coin
// flip, so a tie reports NO top rather than picking one.
func TestTopSuggestion(t *testing.T) {
	tests := []struct {
		name      string
		given     []domain.Suggestion
		wantID    string
		wantCount int
	}{
		{"nobody has advised", nil, "", 0},
		{"one adviser", suggestions("off-mp"), "off-mp", 1},
		{"a clear winner", suggestions("off-mp", "off-mp", "off-upz"), "off-mp", 2},
		{"a two-way tie has no top", suggestions("off-mp", "off-upz"), "", 0},
		{"a tie at the top has no top even with a trailing third",
			suggestions("off-mp", "off-mp", "off-upz", "off-upz", "off-vice"), "", 0},
		{"a winner over a tied pair below it", suggestions("off-mp", "off-mp", "off-mp", "off-upz", "off-vice"), "off-mp", 3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotID, gotCount := domain.TopSuggestion(tt.given)
			if gotID != tt.wantID || gotCount != tt.wantCount {
				t.Fatalf("TopSuggestion = (%q, %d), want (%q, %d)", gotID, gotCount, tt.wantID, tt.wantCount)
			}
		})
	}
}

// TestRequiresReason pins when the super admin owes the public an explanation.
//
// Two triggers, independent: departing from the admins' most-suggested official,
// or from the official the reporter pointed the report at. The consequence worth
// seeing is the third case — when those two disagree, NO choice satisfies both,
// so a reason is always required. That is the intent: disagreement is exactly
// when the public deserves the rationale.
func TestRequiresReason(t *testing.T) {
	tests := []struct {
		name      string
		forwarded string
		top       string // "" when nobody advised, or when the advisers tied
		pointed   string
		want      bool
	}{
		{"agrees with the advisers and the reporter", "off-mp", "off-mp", "off-mp", false},
		{"departs from the reporter, matches the advisers", "off-mp", "off-mp", "off-upz", true},
		{"departs from the advisers, matches the reporter", "off-mp", "off-upz", "off-mp", true},
		{"departs from both", "off-vice", "off-upz", "off-mp", true},
		{"advisers and reporter disagree: any pick needs a reason (a)", "off-mp", "off-upz", "off-mp", true},
		{"advisers and reporter disagree: any pick needs a reason (b)", "off-upz", "off-upz", "off-mp", true},

		// With no top there is nothing to depart from on the advisers' side, so
		// only the reporter's choice can trigger the requirement. This covers BOTH
		// "nobody advised" and "the advisers tied" — TopSuggestion reports the two
		// identically, on purpose.
		{"nobody advised, matches the reporter", "off-mp", "", "off-mp", false},
		{"nobody advised, departs from the reporter", "off-upz", "", "off-mp", true},
		{"the advisers tied, and the reporter's choice is taken", "off-mp", "", "off-mp", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := domain.RequiresReason(tt.forwarded, tt.top, tt.pointed); got != tt.want {
				t.Fatalf("RequiresReason(%q, top=%q, pointed=%q) = %v, want %v",
					tt.forwarded, tt.top, tt.pointed, got, tt.want)
			}
		})
	}
}
