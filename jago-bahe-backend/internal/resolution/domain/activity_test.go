package domain_test

import (
	"testing"
	"time"

	"jago-bahe-backend/internal/resolution/domain"
)

// TestCaseLastActivityAt pins the silence clock's one rule: it reads only what THE
// ASSIGNEE did. Every "not counted" case below is a way the clock could be reset by
// somebody other than the official it is measuring — which would let a case look
// answered when nobody answered it (B19, CLAUDE.md A.3.7).
func TestCaseLastActivityAt(t *testing.T) {
	base := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)
	at := func(d time.Duration) time.Time { return base.Add(d) }
	ptr := func(t time.Time) *time.Time { return &t }

	tests := []struct {
		name string
		kase domain.Case
		want time.Time
	}{
		{
			name: "a bare assigned case has no assignee activity at all",
			kase: domain.Case{CreatedAt: base, Deadline: at(7 * 24 * time.Hour)},
			want: time.Time{},
		},
		{
			name: "acknowledgement is activity",
			kase: domain.Case{CreatedAt: base, AcknowledgedAt: ptr(at(time.Hour))},
			want: at(time.Hour),
		},
		{
			name: "a plan submitted after the acknowledgement wins",
			kase: domain.Case{
				CreatedAt:      base,
				AcknowledgedAt: ptr(at(time.Hour)),
				Plan:           &domain.Plan{CreatedAt: at(5 * time.Hour)},
			},
			want: at(5 * time.Hour),
		},
		{
			name: "a completed weekly task later than every update wins",
			kase: domain.Case{
				CreatedAt: base,
				Plan: &domain.Plan{
					CreatedAt: at(time.Hour),
					Tasks: []domain.PlanTask{
						{WeekNumber: 1, Completed: true, CompletedAt: ptr(at(20 * time.Hour))},
						{WeekNumber: 2},
					},
				},
				Updates: []domain.ProgressUpdate{{CreatedAt: at(3 * time.Hour)}},
			},
			want: at(20 * time.Hour),
		},
		{
			name: "a progress update is activity",
			kase: domain.Case{
				CreatedAt: base,
				Updates: []domain.ProgressUpdate{
					{CreatedAt: at(2 * time.Hour)},
					{CreatedAt: at(9 * time.Hour)},
				},
			},
			want: at(9 * time.Hour),
		},
		{
			name: "uploading evidence is activity",
			kase: domain.Case{CreatedAt: base, Evidence: []domain.Evidence{{CreatedAt: at(4 * time.Hour)}}},
			want: at(4 * time.Hour),
		},
		{
			name: "declaring an obstacle is activity — the official answered",
			kase: domain.Case{CreatedAt: base, Obstacles: []domain.Obstacle{{CreatedAt: at(6 * time.Hour)}}},
			want: at(6 * time.Hour),
		},
		{
			// The admin ruling on an obstacle is the ADMIN's act. Counting it would
			// restart the official's silence clock for work they did not do.
			name: "an adjudication later than everything is NOT counted",
			kase: domain.Case{
				CreatedAt: base,
				Obstacles: []domain.Obstacle{{
					CreatedAt:     at(6 * time.Hour),
					Adjudication:  domain.AdjudicationConfirmed,
					AdjudicatedAt: ptr(at(30 * time.Hour)),
				}},
			},
			want: at(6 * time.Hour),
		},
		{
			name: "an obstacle's resolution is NOT counted either",
			kase: domain.Case{
				CreatedAt: base,
				Obstacles: []domain.Obstacle{{
					CreatedAt:  at(6 * time.Hour),
					ResolvedAt: ptr(at(40 * time.Hour)),
				}},
			},
			want: at(6 * time.Hour),
		},
		{
			// The public judging an obstacle is pressure, not an answer (Concept §8).
			name: "community obstacle votes and unblocking plans are NOT counted",
			kase: domain.Case{
				CreatedAt: base,
				Obstacles: []domain.Obstacle{{
					CreatedAt:         at(6 * time.Hour),
					RealCount:         12,
					NotConvincedCount: 3,
					UnblockingPlans:   []domain.UnblockingPlan{{CreatedAt: at(50 * time.Hour)}},
				}},
			},
			want: at(6 * time.Hour),
		},
		{
			name: "nil plan, empty slices and nil optionals do not panic",
			kase: domain.Case{CreatedAt: base},
			want: time.Time{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.kase.LastActivityAt(); !got.Equal(tt.want) {
				t.Fatalf("LastActivityAt() = %v, want %v", got, tt.want)
			}
		})
	}
}
