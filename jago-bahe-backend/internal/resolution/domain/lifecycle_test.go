package domain_test

import (
	"errors"
	"testing"

	"jago-bahe-backend/internal/resolution/domain"
)

func TestTransition(t *testing.T) {
	withEvidence := domain.Guards{HasEvidence: true}
	noEvidence := domain.Guards{HasEvidence: false}

	tests := []struct {
		name    string
		from    domain.Status
		event   domain.Event
		guards  domain.Guards
		want    domain.Status
		wantErr error
	}{
		// --- the happy path ---
		{"assigned -> acknowledged", domain.StatusAssigned, domain.EventAcknowledge, noEvidence, domain.StatusAcknowledged, nil},
		{"acknowledged -> planned", domain.StatusAcknowledged, domain.EventPlan, noEvidence, domain.StatusPlanned, nil},
		{"planned -> in progress", domain.StatusPlanned, domain.EventProgress, noEvidence, domain.StatusInProgress, nil},
		{"in progress -> in progress (more updates)", domain.StatusInProgress, domain.EventProgress, noEvidence, domain.StatusInProgress, nil},
		{"in progress -> done with evidence", domain.StatusInProgress, domain.EventMarkDone, withEvidence, domain.StatusDone, nil},
		{"done -> resolved on confirm", domain.StatusDone, domain.EventConfirmResolved, noEvidence, domain.StatusResolved, nil},

		// --- dispute path ---
		{"assigned -> disputed", domain.StatusAssigned, domain.EventDispute, noEvidence, domain.StatusDisputed, nil},

		// --- block / resume / reopen ---
		{"planned -> blocked", domain.StatusPlanned, domain.EventBlock, noEvidence, domain.StatusBlocked, nil},
		{"in progress -> blocked", domain.StatusInProgress, domain.EventBlock, noEvidence, domain.StatusBlocked, nil},
		{"blocked resumes to in progress", domain.StatusBlocked, domain.EventProgress, noEvidence, domain.StatusInProgress, nil},
		{"reopened resumes to in progress", domain.StatusReopened, domain.EventProgress, noEvidence, domain.StatusInProgress, nil},
		{"done -> reopened on reject", domain.StatusDone, domain.EventConfirmReopened, noEvidence, domain.StatusReopened, nil},

		// --- replan / restart: a fresh plan resolves to InProgress (active work) ---
		{"in progress can replan", domain.StatusInProgress, domain.EventReplan, noEvidence, domain.StatusInProgress, nil},
		{"reopened can replan", domain.StatusReopened, domain.EventReplan, noEvidence, domain.StatusInProgress, nil},
		{"cannot replan from acknowledged", domain.StatusAcknowledged, domain.EventReplan, noEvidence, domain.StatusAcknowledged, domain.ErrIllegalTransition},
		{"cannot replan from planned", domain.StatusPlanned, domain.EventReplan, noEvidence, domain.StatusPlanned, domain.ErrIllegalTransition},
		{"cannot replan from blocked (must unblock first)", domain.StatusBlocked, domain.EventReplan, noEvidence, domain.StatusBlocked, domain.ErrIllegalTransition},
		{"cannot replan from done", domain.StatusDone, domain.EventReplan, withEvidence, domain.StatusDone, domain.ErrIllegalTransition},
		{"cannot replan from resolved", domain.StatusResolved, domain.EventReplan, noEvidence, domain.StatusResolved, domain.ErrIllegalTransition},
		{"cannot replan from assigned", domain.StatusAssigned, domain.EventReplan, noEvidence, domain.StatusAssigned, domain.ErrIllegalTransition},
		{"cannot replan from disputed", domain.StatusDisputed, domain.EventReplan, noEvidence, domain.StatusDisputed, domain.ErrIllegalTransition},

		// --- evidence guard ---
		{"cannot mark done without evidence", domain.StatusInProgress, domain.EventMarkDone, noEvidence, domain.StatusInProgress, domain.ErrEvidenceRequired},

		// --- illegal moves ---
		{"cannot plan before acknowledging", domain.StatusAssigned, domain.EventPlan, noEvidence, domain.StatusAssigned, domain.ErrIllegalTransition},
		{"cannot mark done from planned", domain.StatusPlanned, domain.EventMarkDone, withEvidence, domain.StatusPlanned, domain.ErrIllegalTransition},
		{"cannot block from acknowledged", domain.StatusAcknowledged, domain.EventBlock, noEvidence, domain.StatusAcknowledged, domain.ErrIllegalTransition},
		{"cannot block from blocked", domain.StatusBlocked, domain.EventBlock, noEvidence, domain.StatusBlocked, domain.ErrIllegalTransition},
		{"cannot acknowledge twice", domain.StatusAcknowledged, domain.EventAcknowledge, noEvidence, domain.StatusAcknowledged, domain.ErrIllegalTransition},
		{"cannot progress from assigned", domain.StatusAssigned, domain.EventProgress, noEvidence, domain.StatusAssigned, domain.ErrIllegalTransition},
		{"official cannot resolve (no confirm from in progress)", domain.StatusInProgress, domain.EventConfirmResolved, noEvidence, domain.StatusInProgress, domain.ErrIllegalTransition},
		{"resolved is terminal", domain.StatusResolved, domain.EventProgress, noEvidence, domain.StatusResolved, domain.ErrIllegalTransition},
		{"disputed is terminal", domain.StatusDisputed, domain.EventPlan, noEvidence, domain.StatusDisputed, domain.ErrIllegalTransition},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.Transition(tt.from, tt.event, tt.guards)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("Transition(%s, %s) err = %v, want %v", tt.from, tt.event, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("Transition(%s, %s) = %s, want %s", tt.from, tt.event, got, tt.want)
			}
		})
	}
}

func TestIsWorkable(t *testing.T) {
	workable := []domain.Status{domain.StatusPlanned, domain.StatusInProgress, domain.StatusBlocked, domain.StatusReopened}
	notWorkable := []domain.Status{domain.StatusAssigned, domain.StatusAcknowledged, domain.StatusDone, domain.StatusResolved, domain.StatusDisputed}

	for _, s := range workable {
		if !domain.IsWorkable(s) {
			t.Errorf("IsWorkable(%s) = false, want true", s)
		}
	}
	for _, s := range notWorkable {
		if domain.IsWorkable(s) {
			t.Errorf("IsWorkable(%s) = true, want false", s)
		}
	}
}
