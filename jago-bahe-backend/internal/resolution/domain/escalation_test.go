package domain_test

import (
	"testing"
	"time"

	"jago-bahe-backend/internal/resolution/domain"
)

func TestEscalationTarget(t *testing.T) {
	anchor := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	const window = 24 * time.Hour
	const max = 3

	tests := []struct {
		name string
		now  time.Time
		want int
	}{
		{"before the anchor is level 0", anchor.Add(-time.Hour), 0},
		{"exactly at the anchor is level 0", anchor, 0},
		{"just past the anchor climbs to 1", anchor.Add(time.Minute), 1},
		{"within the first window stays at 1", anchor.Add(23 * time.Hour), 1},
		{"one full window past climbs to 2", anchor.Add(window), 2},
		{"two full windows past climbs to 3", anchor.Add(2 * window), 3},
		{"capped at maxLevel", anchor.Add(10 * window), 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := domain.EscalationTarget(anchor, tt.now, window, max); got != tt.want {
				t.Fatalf("EscalationTarget(now=%v) = %d, want %d", tt.now.Sub(anchor), got, tt.want)
			}
		})
	}
}

func TestEscalationTarget_ZeroWindow(t *testing.T) {
	anchor := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	if got := domain.EscalationTarget(anchor, anchor.Add(time.Hour), 0, 3); got != 0 {
		t.Fatalf("zero window should not escalate, got %d", got)
	}
}

func TestIsEscalatable(t *testing.T) {
	yes := []domain.Status{domain.StatusAssigned, domain.StatusAcknowledged, domain.StatusPlanned,
		domain.StatusInProgress, domain.StatusReopened, domain.StatusBlocked}
	no := []domain.Status{domain.StatusDone, domain.StatusResolved, domain.StatusDisputed}
	for _, s := range yes {
		if !domain.IsEscalatable(s) {
			t.Errorf("IsEscalatable(%s) = false, want true", s)
		}
	}
	for _, s := range no {
		if domain.IsEscalatable(s) {
			t.Errorf("IsEscalatable(%s) = true, want false", s)
		}
	}
}
