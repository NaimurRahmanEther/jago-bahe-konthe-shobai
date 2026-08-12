package http

import (
	"time"

	"jago-bahe-backend/internal/resolution/application"
	"jago-bahe-backend/internal/resolution/domain"
)

// observedCaseDTO is one row of a monitor's observation list (B19): a case below
// them on the accountability ladder, and whether its official has gone quiet.
//
// It is deliberately NOT caseDTO. A monitor watches a case; they do not work it,
// so this carries none of the assignee's working internals — no disputeReason, no
// monitorOfficialId — and offers no action beyond a note. The case is never moved
// off the official: silence raises visibility, not responsibility (Concept §7).
type observedCaseDTO struct {
	CaseID       string `json:"caseId"`
	ProblemID    string `json:"problemId"`
	ProblemTitle string `json:"problemTitle"`
	OfficialID   string `json:"officialId"`
	OfficialName string `json:"officialName"`
	Status       string `json:"status"`
	Deadline     string `json:"deadline"`
	// LastActivityAt is null when the official has never acted on the case at all —
	// the state this whole feature exists to make visible.
	LastActivityAt *string `json:"lastActivityAt"`
	// SilentDays counts days waiting for an answer past the deadline. 0 means the
	// case is on time or has been answered.
	SilentDays int `json:"silentDays"`
	// Direct distinguishes continuous supervision (the caller is this case's own
	// monitor, watching since assignment) from a case that only climbed to them.
	Direct       bool             `json:"direct"`
	Observations []observationDTO `json:"observations"`
}

// observationDTO is one rung of the ladder watching one case. resolvedAt is
// present-and-null while the official is still silent, never absent: both are
// falsy in JS, so a field that came and went would surface only as a wrongly
// enabled button (the A.5.2 lesson at the field level).
type observationDTO struct {
	ID         string               `json:"id"`
	Level      int                  `json:"level"`
	OpenedAt   string               `json:"openedAt"`
	ResolvedAt *string              `json:"resolvedAt"`
	Notes      []observationNoteDTO `json:"notes"`
}

// observationNoteDTO is one thing a monitor recorded doing about the silence.
type observationNoteDTO struct {
	ID        string `json:"id"`
	Text      string `json:"text"`
	CreatedAt string `json:"createdAt"`
}

func toObservedCaseDTOs(rows []application.ObservedCase) []observedCaseDTO {
	out := make([]observedCaseDTO, 0, len(rows))
	for _, r := range rows {
		out = append(out, observedCaseDTO{
			CaseID:         r.CaseID,
			ProblemID:      r.ProblemID,
			ProblemTitle:   r.ProblemTitle,
			OfficialID:     r.OfficialID,
			OfficialName:   r.OfficialName,
			Status:         r.Status,
			Deadline:       r.Deadline.Format(time.RFC3339),
			LastActivityAt: optionalTime(r.LastActivityAt),
			SilentDays:     r.SilentDays,
			Direct:         r.Direct,
			Observations:   toObservationDTOs(r.Observations),
		})
	}
	return out
}

func toObservationDTOs(obs []domain.Observation) []observationDTO {
	out := make([]observationDTO, 0, len(obs))
	for i := range obs {
		out = append(out, toObservationDTO(&obs[i]))
	}
	return out
}

func toObservationDTO(o *domain.Observation) observationDTO {
	notes := make([]observationNoteDTO, 0, len(o.Notes))
	for _, n := range o.Notes {
		notes = append(notes, toObservationNoteDTO(n))
	}
	var resolved *string
	if o.ResolvedAt != nil {
		s := o.ResolvedAt.Format(time.RFC3339)
		resolved = &s
	}
	return observationDTO{
		ID:         o.ID,
		Level:      o.Level,
		OpenedAt:   o.OpenedAt.Format(time.RFC3339),
		ResolvedAt: resolved,
		Notes:      notes,
	}
}

func toObservationNoteDTO(n domain.ObservationNote) observationNoteDTO {
	return observationNoteDTO{ID: n.ID, Text: n.Text, CreatedAt: n.CreatedAt.Format(time.RFC3339)}
}

// optionalTime renders a zero time as JSON null rather than a fake timestamp.
func optionalTime(t time.Time) *string {
	if t.IsZero() {
		return nil
	}
	s := t.Format(time.RFC3339)
	return &s
}
