package domain_test

import (
	"testing"
	"time"

	"jago-bahe-backend/internal/resolution/domain"
)

func TestObservationOpenAndResolve(t *testing.T) {
	now := time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC)

	o := domain.NewObservation("case-1", "off-upazila", 1, now)
	if !o.IsOpen() {
		t.Fatal("a fresh observation must be open")
	}
	if o.Level != 1 || o.CaseID != "case-1" || o.ObserverOfficialID != "off-upazila" {
		t.Fatalf("unexpected observation %+v", o)
	}
	if !o.OpenedAt.Equal(now) {
		t.Fatalf("OpenedAt = %v, want %v", o.OpenedAt, now)
	}

	resolved := now.Add(48 * time.Hour)
	o.Resolve(resolved)
	if o.IsOpen() {
		t.Fatal("a resolved observation must not read as open")
	}
	if o.ResolvedAt == nil || !o.ResolvedAt.Equal(resolved) {
		t.Fatalf("ResolvedAt = %v, want %v", o.ResolvedAt, resolved)
	}

	// Idempotent: resolving again must not rewrite when the silence ended. The
	// first response is the fact worth keeping.
	o.Resolve(resolved.Add(72 * time.Hour))
	if !o.ResolvedAt.Equal(resolved) {
		t.Fatalf("re-resolving moved ResolvedAt to %v", o.ResolvedAt)
	}
}

// TestMissingRungs pins the ladder walk: rungs open in order, so a case can never
// be visible to the MP without being visible to the tier below first.
func TestMissingRungs(t *testing.T) {
	open := func(levels ...int) []domain.Observation {
		out := make([]domain.Observation, 0, len(levels))
		for _, l := range levels {
			out = append(out, domain.Observation{Level: l})
		}
		return out
	}
	resolved := func(level int) domain.Observation {
		at := time.Now()
		return domain.Observation{Level: level, ResolvedAt: &at}
	}

	tests := []struct {
		name   string
		open   []domain.Observation
		target int
		want   []int
	}{
		{"target 0 opens nothing", nil, 0, nil},
		{"nothing open, target 1", nil, 1, []int{1}},
		{"nothing open, target 3 opens 1,2,3 in order", nil, 3, []int{1, 2, 3}},
		{"rung 1 already open, target 2 opens only 2", open(1), 2, []int{2}},
		{"every rung open, target 3 opens nothing", open(1, 2, 3), 3, nil},
		{"a resolved rung does not count as open", []domain.Observation{resolved(1)}, 1, []int{1}},
		{"target below what is open opens nothing", open(1, 2, 3), 1, nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.MissingRungs(tt.open, tt.target)
			if len(got) != len(tt.want) {
				t.Fatalf("MissingRungs = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("MissingRungs = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// TestSilenceLevel pins the second clock. It is deliberately NOT the escalation
// clock: escalation measures lateness from a fixed deadline, this measures silence
// from the official's last act, so an official who is working does not accumulate
// observers (CLAUDE.md A.3.7).
func TestSilenceLevel(t *testing.T) {
	deadline := time.Date(2026, 3, 8, 0, 0, 0, 0, time.UTC)
	const window = 7 * 24 * time.Hour

	tests := []struct {
		name         string
		lastActivity time.Time
		now          time.Time
		want         int
	}{
		{"never acted, before the deadline", time.Time{}, deadline.Add(-time.Hour), 0},
		{"never acted, exactly at the deadline", time.Time{}, deadline, 0},
		{"never acted, one hour past", time.Time{}, deadline.Add(time.Hour), 1},
		{"never acted, one full window past", time.Time{}, deadline.Add(window), 2},
		{"never acted, three windows past is capped", time.Time{}, deadline.Add(3 * window), 3},
		{
			name:         "acting after the deadline restarts the clock",
			lastActivity: deadline.Add(3 * window),
			now:          deadline.Add(3*window + time.Hour),
			want:         0,
		},
		{
			name:         "going silent again after acting climbs from the act, not the deadline",
			lastActivity: deadline.Add(3 * window),
			now:          deadline.Add(4*window + time.Hour),
			want:         1,
		},
		{
			name:         "activity before the deadline does not delay the clock",
			lastActivity: deadline.Add(-2 * window),
			now:          deadline.Add(time.Hour),
			want:         1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := domain.SilenceLevel(deadline, tt.lastActivity, tt.now, window, domain.MaxEscalationLevel)
			if got != tt.want {
				t.Fatalf("SilenceLevel = %d, want %d", got, tt.want)
			}
		})
	}
}
