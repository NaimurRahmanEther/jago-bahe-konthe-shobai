package application

import (
	"context"
	"sort"
	"time"

	"jago-bahe-backend/internal/resolution/domain"
)

// ObservedCase is one row of a monitor's observation list: a case below them,
// what it is about, whether its official has gone quiet, and for how long.
//
// It is deliberately not the case DTO. A monitor watches; they do not work the
// case, so nothing here is an internal of the assignee's (no dispute reason, no
// monitor id) and nothing here is an action.
type ObservedCase struct {
	CaseID         string
	ProblemID      string
	ProblemTitle   string
	OfficialID     string
	OfficialName   string
	Status         string
	Deadline       time.Time
	LastActivityAt time.Time // zero when the official has never acted
	SilentDays     int       // days past the response deadline with no answer; 0 when answered or on time
	Direct         bool      // the caller is this case's own monitor, watching from assignment
	Observations   []domain.Observation
}

// HasOpenObservation reports whether a rung is currently watching this case.
func (o ObservedCase) HasOpenObservation() bool {
	for i := range o.Observations {
		if o.Observations[i].IsOpen() {
			return true
		}
	}
	return false
}

// ListObservations is the monitor's view of the officials below them.
//
// Two things reach it, and the difference is the whole design (Concept §7):
//
//   - Every case the caller is the DIRECT monitor of, from the moment it was
//     assigned and whether or not anything has gone wrong. Monitoring is
//     continuous so that a higher authority can never later claim they did not know.
//   - Cases from FURTHER DOWN the ladder, but only once silence has climbed to the
//     caller. A tier does not get to read everything beneath it as a matter of
//     course; it is called in.
//
// It takes no parameter. The caller comes from the JWT, exactly as the admin
// queue's union scope does — a monitor id in the query string would let any
// official read any other official's supervision list.
type ListObservations struct {
	repo      domain.Repository
	problems  Problems
	officials Officials
}

// NewListObservations wires the use case.
func NewListObservations(r domain.Repository, problems Problems, officials Officials) *ListObservations {
	return &ListObservations{repo: r, problems: problems, officials: officials}
}

// Execute returns the caller's observed cases, most urgent first.
func (uc *ListObservations) Execute(ctx context.Context, officialID string, now time.Time) ([]ObservedCase, error) {
	if officialID == "" {
		return nil, domain.ErrNotObserver
	}

	direct, err := uc.repo.ListByMonitor(ctx, officialID)
	if err != nil {
		return nil, err
	}
	mine, err := uc.repo.ListObservationsByObserver(ctx, officialID)
	if err != nil {
		return nil, err
	}

	seen := make(map[string]bool, len(direct))
	cases := make([]domain.Case, 0, len(direct)+len(mine))
	for _, c := range direct {
		seen[c.ID] = true
		cases = append(cases, c)
	}

	// The rungs above the direct monitor: cases they can see only because silence
	// climbed to them. ListByMonitor would never return these.
	var escalatedIDs []string
	for _, o := range mine {
		if !seen[o.CaseID] {
			seen[o.CaseID] = true
			escalatedIDs = append(escalatedIDs, o.CaseID)
		}
	}
	if len(escalatedIDs) > 0 {
		extra, err := uc.repo.CasesByIDs(ctx, escalatedIDs)
		if err != nil {
			return nil, err
		}
		cases = append(cases, extra...)
	}
	if len(cases) == 0 {
		return nil, nil
	}

	caseIDs := make([]string, 0, len(cases))
	problemIDs := make([]string, 0, len(cases))
	officialIDs := make([]string, 0, len(cases))
	for i := range cases {
		caseIDs = append(caseIDs, cases[i].ID)
		problemIDs = append(problemIDs, cases[i].ProblemID)
		officialIDs = append(officialIDs, cases[i].OfficialID)
	}

	// Three batch lookups for the page, never one per row (A.3.4). Each degrades to
	// "not known" rather than failing the list: a monitor who cannot see the names
	// still has to be able to see the silences.
	observations, err := uc.repo.ListObservationsByCases(ctx, caseIDs)
	if err != nil {
		observations = nil
	}
	byCase := make(map[string][]domain.Observation, len(cases))
	for _, o := range observations {
		byCase[o.CaseID] = append(byCase[o.CaseID], o)
	}
	titles, err := uc.problems.TitlesByIDs(ctx, problemIDs)
	if err != nil {
		titles = nil
	}
	names, err := uc.officials.NamesByIDs(ctx, officialIDs)
	if err != nil {
		names = nil
	}

	out := make([]ObservedCase, 0, len(cases))
	for i := range cases {
		c := &cases[i]
		row := ObservedCase{
			CaseID:         c.ID,
			ProblemID:      c.ProblemID,
			ProblemTitle:   titles[c.ProblemID],
			OfficialID:     c.OfficialID,
			OfficialName:   names[c.OfficialID],
			Status:         string(c.Status),
			Deadline:       c.Deadline,
			LastActivityAt: c.LastActivityAt(),
			Direct:         c.MonitorOfficialID == officialID,
			Observations:   byCase[c.ID],
		}
		row.SilentDays = silentDays(c, row.LastActivityAt, now)
		out = append(out, row)
	}

	// Ordering is the backend's (A.5 rule 7): cases with somebody waiting come
	// first, longest silence first, so the page never has to decide what is urgent.
	sort.SliceStable(out, func(i, j int) bool {
		oi, oj := out[i].HasOpenObservation(), out[j].HasOpenObservation()
		if oi != oj {
			return oi
		}
		if out[i].SilentDays != out[j].SilentDays {
			return out[i].SilentDays > out[j].SilentDays
		}
		return out[i].Deadline.Before(out[j].Deadline)
	})
	return out, nil
}

// silentDays reports how long the case has been waiting for an answer, counted
// from whichever came last: the response deadline, or the official's last act. It
// is 0 for a case that is on time or has been answered since the deadline passed.
func silentDays(c *domain.Case, lastActivity, now time.Time) int {
	from := c.Deadline
	if lastActivity.After(from) {
		from = lastActivity
	}
	if !now.After(from) {
		return 0
	}
	return int(now.Sub(from).Hours() / 24)
}
