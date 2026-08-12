package application

import "jago-bahe-backend/internal/assignment/domain"

// resolveOverride encapsulates the B4 override rule: when a union admin assigns a
// problem to an official other than the one the public pointed to, the change is
// an override and a public reason is mandatory (Concept §5 — "override it with a
// reason that is shown publicly"). Confirming the public's choice carries no
// reason. It returns the reason to persist on the assignment (empty on a confirm).
func resolveOverride(pointedOfficialID, chosenOfficialID, reason string) (string, error) {
	if chosenOfficialID == pointedOfficialID {
		return "", nil // confirming the public's choice; no override reason
	}
	if reason == "" {
		return "", domain.ErrMissingOverrideReason
	}
	return reason, nil
}
