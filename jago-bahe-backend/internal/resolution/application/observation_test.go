package application_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"jago-bahe-backend/internal/resolution/application"
	"jago-bahe-backend/internal/resolution/domain"
)

// --- B19 fakes: the observation half of fakeRepo (fields declared in b6_test.go) ---

func (r *fakeRepo) ListByMonitor(_ context.Context, officialID string) ([]domain.Case, error) {
	var out []domain.Case
	for _, c := range r.cases {
		if c.MonitorOfficialID == officialID && c.Status != domain.StatusResolved && c.Status != domain.StatusDisputed {
			out = append(out, *cloneCase(c))
		}
	}
	return out, nil
}

func (r *fakeRepo) CasesByIDs(_ context.Context, caseIDs []string) ([]domain.Case, error) {
	var out []domain.Case
	for _, id := range caseIDs {
		if c, ok := r.cases[id]; ok {
			out = append(out, *cloneCase(c))
		}
	}
	return out, nil
}

func (r *fakeRepo) ListObservationsByObserver(_ context.Context, officialID string) ([]domain.Observation, error) {
	var out []domain.Observation
	for _, o := range r.observations {
		if o.ObserverOfficialID == officialID {
			out = append(out, *o)
		}
	}
	return out, nil
}

func (r *fakeRepo) ListObservationsByCases(_ context.Context, caseIDs []string) ([]domain.Observation, error) {
	want := make(map[string]bool, len(caseIDs))
	for _, id := range caseIDs {
		want[id] = true
	}
	var out []domain.Observation
	for _, o := range r.observations {
		if want[o.CaseID] {
			out = append(out, *o)
		}
	}
	return out, nil
}

// OpenObservation mirrors the partial unique index: one OPEN row per (case, level).
func (r *fakeRepo) OpenObservation(_ context.Context, o *domain.Observation) (bool, error) {
	for _, existing := range r.observations {
		if existing.CaseID == o.CaseID && existing.Level == o.Level && existing.IsOpen() {
			return false, nil
		}
	}
	clone := *o
	r.observations = append(r.observations, &clone)
	return true, nil
}

func (r *fakeRepo) ResolveOpenObservations(_ context.Context, caseID string, at time.Time) (int, error) {
	n := 0
	for _, o := range r.observations {
		if o.CaseID == caseID && o.IsOpen() {
			o.Resolve(at)
			n++
		}
	}
	return n, nil
}

func (r *fakeRepo) GetObservation(_ context.Context, id string) (*domain.Observation, error) {
	for _, o := range r.observations {
		if o.ID == id {
			clone := *o
			for _, n := range r.notes {
				if n.ObservationID == o.ID {
					clone.Notes = append(clone.Notes, n)
				}
			}
			return &clone, nil
		}
	}
	return nil, domain.ErrObservationNotFound
}

func (r *fakeRepo) AddObservationNote(_ context.Context, n domain.ObservationNote) error {
	r.notes = append(r.notes, n)
	return nil
}

// LastActivityByCases stands in for the SQL projection. It prefers an explicit
// override so a test can pin a value, and otherwise derives it from the aggregate
// through the domain rule — which is what the agreement between the two must mean.
func (r *fakeRepo) LastActivityByCases(_ context.Context, caseIDs []string) (map[string]time.Time, error) {
	out := make(map[string]time.Time, len(caseIDs))
	for _, id := range caseIDs {
		if at, ok := r.lastActivity[id]; ok {
			out[id] = at
			continue
		}
		if c, ok := r.cases[id]; ok {
			if at := c.LastActivityAt(); !at.IsZero() {
				out[id] = at
			}
		}
	}
	return out, nil
}

// fakeLadder answers "who monitors whom" from a fixed chain.
type fakeLadder struct {
	chain []string
	err   error
}

func (f *fakeLadder) Chain(_ context.Context, _ string, maxRungs int) ([]string, error) {
	if f.err != nil {
		return nil, f.err
	}
	if len(f.chain) > maxRungs {
		return f.chain[:maxRungs], nil
	}
	return f.chain, nil
}

type fakeOfficialNames struct {
	names map[string]string
	err   error
}

func (f *fakeOfficialNames) NamesByIDs(context.Context, []string) (map[string]string, error) {
	return f.names, f.err
}

// --- helpers ---

const obsD = 7 * 24 * time.Hour
const obsR = 72 * time.Hour

func silentCase(id, problemID string, deadline time.Time) *domain.Case {
	return &domain.Case{
		ID: id, ProblemID: problemID, OfficialID: "off-union", MonitorOfficialID: "off-upazila",
		Status: domain.StatusAssigned, Deadline: deadline,
	}
}

func fullLadder() *fakeLadder { return &fakeLadder{chain: []string{"off-upazila", "off-mp"}} }

func openLevels(repo *fakeRepo, caseID string) []int {
	var out []int
	for _, o := range repo.observations {
		if o.CaseID == caseID && o.IsOpen() {
			out = append(out, o.Level)
		}
	}
	return out
}

// --- the worker's observation half ---

func TestEscalateOverdue_OpensRungsInOrder(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := newFakeRepo()
	// Silent since a deadline two full windows ago → rungs 1 and 2.
	repo.put(silentCase("c-1", "p-1", now.Add(-2*obsD-time.Minute)))

	uc := application.NewEscalateOverdue(repo, &fakeAudit{}, fullLadder(), obsD, obsR)
	res, err := uc.Run(context.Background(), now)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.ObservationsOpened != 2 {
		t.Fatalf("opened %d observations, want 2", res.ObservationsOpened)
	}

	levels := openLevels(repo, "c-1")
	if len(levels) != 2 || levels[0] != 1 || levels[1] != 2 {
		t.Fatalf("open rungs = %v, want [1 2] in order", levels)
	}
	if repo.observations[0].ObserverOfficialID != "off-upazila" {
		t.Fatalf("rung 1 observer = %q, want the direct monitor", repo.observations[0].ObserverOfficialID)
	}
	if repo.observations[1].ObserverOfficialID != "off-mp" {
		t.Fatalf("rung 2 observer = %q, want the tier above", repo.observations[1].ObserverOfficialID)
	}
}

func TestEscalateOverdue_ObservationsAreIdempotent(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := newFakeRepo()
	repo.put(silentCase("c-1", "p-1", now.Add(-obsD-time.Minute)))
	uc := application.NewEscalateOverdue(repo, &fakeAudit{}, fullLadder(), obsD, obsR)

	if _, err := uc.Run(context.Background(), now); err != nil {
		t.Fatalf("run: %v", err)
	}
	before := len(repo.observations)
	res, err := uc.Run(context.Background(), now)
	if err != nil {
		t.Fatalf("second run: %v", err)
	}
	if res.ObservationsOpened != 0 || len(repo.observations) != before {
		t.Fatalf("second run opened %d (total %d), want 0 (total %d)", res.ObservationsOpened, len(repo.observations), before)
	}
}

// A case whose official has never acted, but whose deadline has not passed, has
// nobody waiting on it — monitoring it is continuous, escalating it is not.
func TestEscalateOverdue_OnTimeCaseGetsNoObserver(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := newFakeRepo()
	repo.put(silentCase("c-1", "p-1", now.Add(obsD)))

	uc := application.NewEscalateOverdue(repo, &fakeAudit{}, fullLadder(), obsD, obsR)
	res, _ := uc.Run(context.Background(), now)
	if res.ObservationsOpened != 0 {
		t.Fatalf("opened %d observations on an on-time case, want 0", res.ObservationsOpened)
	}
}

func TestEscalateOverdue_ResolvesWhenTheOfficialResponds(t *testing.T) {
	opened := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := newFakeRepo()
	// An hour past the deadline: inside the first window, so exactly rung 1 opens.
	repo.put(silentCase("c-1", "p-1", opened.Add(-time.Hour)))
	uc := application.NewEscalateOverdue(repo, &fakeAudit{}, fullLadder(), obsD, obsR)
	if _, err := uc.Run(context.Background(), opened); err != nil {
		t.Fatalf("run: %v", err)
	}
	if len(openLevels(repo, "c-1")) == 0 {
		t.Fatal("setup: expected an open observation")
	}

	// The official acts a day later; the next scan releases the monitor.
	responded := opened.Add(24 * time.Hour)
	repo.lastActivity["c-1"] = responded
	res, err := uc.Run(context.Background(), responded.Add(time.Hour))
	if err != nil {
		t.Fatalf("run after response: %v", err)
	}
	if res.ObservationsClosed != 1 {
		t.Fatalf("closed %d observations, want 1", res.ObservationsClosed)
	}
	if got := openLevels(repo, "c-1"); len(got) != 0 {
		t.Fatalf("open rungs after the response = %v, want none", got)
	}
	// Kept as history, not deleted: that the official was silent is a fact.
	if len(repo.observations) != 1 || repo.observations[0].ResolvedAt == nil {
		t.Fatalf("the resolved observation must survive as history, got %+v", repo.observations)
	}
}

// The regression that pins the two-clock design. cases.escalation_level is a
// high-water mark and can never fall, so if observations were driven off it a
// SECOND silence would open nothing and the monitor would never be called back.
func TestEscalateOverdue_ReopensAfterASecondSilence(t *testing.T) {
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := newFakeRepo()
	repo.put(silentCase("c-1", "p-1", start.Add(-3*obsD)))
	uc := application.NewEscalateOverdue(repo, &fakeAudit{}, fullLadder(), obsD, obsR)

	if _, err := uc.Run(context.Background(), start); err != nil {
		t.Fatalf("first scan: %v", err)
	}
	level := repo.escalations["c-1"]
	if level < 2 {
		t.Fatalf("setup: expected the case to have climbed, level = %d", level)
	}

	// The official answers → everything closes.
	responded := start.Add(time.Hour)
	repo.lastActivity["c-1"] = responded
	if _, err := uc.Run(context.Background(), responded.Add(time.Minute)); err != nil {
		t.Fatalf("scan after response: %v", err)
	}
	if got := openLevels(repo, "c-1"); len(got) != 0 {
		t.Fatalf("open rungs after the response = %v, want none", got)
	}

	// ...and then goes quiet again for a full window.
	res, err := uc.Run(context.Background(), responded.Add(obsD+time.Hour))
	if err != nil {
		t.Fatalf("scan after the second silence: %v", err)
	}
	if res.ObservationsOpened != 1 {
		t.Fatalf("second silence opened %d observations, want 1", res.ObservationsOpened)
	}
	if got := openLevels(repo, "c-1"); len(got) != 1 || got[0] != 1 {
		t.Fatalf("open rungs = %v, want rung 1 re-opened", got)
	}
	// The level is a record of how late the case ever got and does not fall back.
	if repo.escalations["c-1"] < level {
		t.Fatalf("escalation level fell from %d to %d", level, repo.escalations["c-1"])
	}
}

// A case with no ladder above it (the MP's own work) must not fail the scan.
func TestEscalateOverdue_NoLadderIsNotAnError(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := newFakeRepo()
	repo.put(silentCase("c-1", "p-1", now.Add(-2*obsD)))

	uc := application.NewEscalateOverdue(repo, &fakeAudit{}, &fakeLadder{}, obsD, obsR)
	res, err := uc.Run(context.Background(), now)
	if err != nil {
		t.Fatalf("run: %v", err)
	}
	if res.ObservationsOpened != 0 {
		t.Fatalf("opened %d observations without a ladder, want 0", res.ObservationsOpened)
	}
	if res.Escalated != 1 {
		t.Fatalf("the level should still have risen, escalated = %d", res.Escalated)
	}
}

// One official with an unresolvable ladder must not stop the rest of the seat from
// escalating: the worker walks every open case on every tick.
func TestEscalateOverdue_LadderFailureDegrades(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := newFakeRepo()
	repo.put(silentCase("c-1", "p-1", now.Add(-2*obsD)))
	repo.put(silentCase("c-2", "p-2", now.Add(-2*obsD)))

	uc := application.NewEscalateOverdue(repo, &fakeAudit{}, &fakeLadder{err: errors.New("directory down")}, obsD, obsR)
	res, err := uc.Run(context.Background(), now)
	if err != nil {
		t.Fatalf("a ladder failure must not fail the scan: %v", err)
	}
	if res.Escalated != 2 {
		t.Fatalf("escalated %d, want both cases", res.Escalated)
	}
}

// The rungs that opened are on the public trail, once each.
func TestEscalateOverdue_AuditsEachRungOnce(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := newFakeRepo()
	// One rung's worth of silence, so the count below is unambiguous.
	repo.put(silentCase("c-1", "p-1", now.Add(-time.Hour)))
	audit := &fakeAudit{}
	uc := application.NewEscalateOverdue(repo, audit, fullLadder(), obsD, obsR)

	_, _ = uc.Run(context.Background(), now)
	_, _ = uc.Run(context.Background(), now)

	opened := 0
	for _, e := range audit.entries {
		if e.Action == "observation_opened" {
			opened++
		}
	}
	if opened != 1 {
		t.Fatalf("wrote %d observation_opened entries across two scans, want 1", opened)
	}
}

// --- the monitor's list ---

func observedList(t *testing.T, repo *fakeRepo, officialID string, now time.Time) []application.ObservedCase {
	t.Helper()
	uc := application.NewListObservations(repo, &fakeProblems{}, &fakeOfficialNames{names: map[string]string{"off-union": "Union Chairman"}})
	rows, err := uc.Execute(context.Background(), officialID, now)
	if err != nil {
		t.Fatalf("list observations: %v", err)
	}
	return rows
}

// Decision 1: monitoring is continuous from assignment, not something that starts
// when a deadline is missed — otherwise a monitor can say they never knew.
func TestListObservations_IncludesHealthyMonitoredCases(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := newFakeRepo()
	repo.put(silentCase("c-ontime", "p-ontime", now.Add(obsD)))

	rows := observedList(t, repo, "off-upazila", now)
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want the healthy monitored case", len(rows))
	}
	if rows[0].HasOpenObservation() {
		t.Fatal("an on-time case must not read as escalated")
	}
	if !rows[0].Direct {
		t.Fatal("the direct monitor's own case must be marked direct")
	}
	if rows[0].SilentDays != 0 {
		t.Fatalf("silentDays = %d on an on-time case, want 0", rows[0].SilentDays)
	}
}

// A tier above the direct monitor sees a case only once silence climbed to them.
func TestListObservations_HigherRungSeesOnlyEscalatedCases(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := newFakeRepo()
	repo.put(silentCase("c-quiet", "p-quiet", now.Add(obsD)))      // healthy
	repo.put(silentCase("c-silent", "p-silent", now.Add(-2*obsD))) // climbing

	uc := application.NewEscalateOverdue(repo, &fakeAudit{}, fullLadder(), obsD, obsR)
	if _, err := uc.Run(context.Background(), now); err != nil {
		t.Fatalf("scan: %v", err)
	}

	rows := observedList(t, repo, "off-mp", now)
	if len(rows) != 1 {
		t.Fatalf("the MP sees %d cases, want only the escalated one", len(rows))
	}
	if rows[0].CaseID != "c-silent" {
		t.Fatalf("the MP sees %q, want c-silent", rows[0].CaseID)
	}
	if rows[0].Direct {
		t.Fatal("a case that only climbed to this official is not their direct supervision")
	}
}

// Decision 2: the history survives the response, so a pattern stays visible.
func TestListObservations_KeepsResolvedHistory(t *testing.T) {
	start := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := newFakeRepo()
	repo.put(silentCase("c-1", "p-1", start.Add(-2*obsD)))
	uc := application.NewEscalateOverdue(repo, &fakeAudit{}, fullLadder(), obsD, obsR)
	if _, err := uc.Run(context.Background(), start); err != nil {
		t.Fatalf("scan: %v", err)
	}
	repo.lastActivity["c-1"] = start.Add(time.Hour)
	if _, err := uc.Run(context.Background(), start.Add(2*time.Hour)); err != nil {
		t.Fatalf("scan after response: %v", err)
	}

	// The MP was called in and released; the record of it stays on their list.
	rows := observedList(t, repo, "off-mp", start.Add(3*time.Hour))
	if len(rows) != 1 {
		t.Fatalf("got %d rows, want the resolved case kept as history", len(rows))
	}
	if rows[0].HasOpenObservation() {
		t.Fatal("the observation should be closed")
	}
	if len(rows[0].Observations) == 0 || rows[0].Observations[0].ResolvedAt == nil {
		t.Fatalf("the resolved observation must be returned, got %+v", rows[0].Observations)
	}
}

// Ordering is the backend's: the page must not decide what is urgent (A.5 rule 7).
func TestListObservations_SortsWaitingFirst(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := newFakeRepo()
	repo.put(silentCase("c-ontime", "p-ontime", now.Add(obsD)))
	repo.put(silentCase("c-late", "p-late", now.Add(-obsD-time.Minute)))
	repo.put(silentCase("c-latest", "p-latest", now.Add(-3*obsD)))

	uc := application.NewEscalateOverdue(repo, &fakeAudit{}, fullLadder(), obsD, obsR)
	if _, err := uc.Run(context.Background(), now); err != nil {
		t.Fatalf("scan: %v", err)
	}

	rows := observedList(t, repo, "off-upazila", now)
	if len(rows) != 3 {
		t.Fatalf("got %d rows, want 3", len(rows))
	}
	if rows[0].CaseID != "c-latest" || rows[1].CaseID != "c-late" || rows[2].CaseID != "c-ontime" {
		t.Fatalf("order = %s, %s, %s; want longest silence first and the healthy case last",
			rows[0].CaseID, rows[1].CaseID, rows[2].CaseID)
	}
}

// Reads are never audited: an entry per view would publish who is watching whom.
func TestListObservations_NeverAudits(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := newFakeRepo()
	repo.put(silentCase("c-1", "p-1", now.Add(-2*obsD)))
	audit := &fakeAudit{}
	scan := application.NewEscalateOverdue(repo, audit, fullLadder(), obsD, obsR)
	if _, err := scan.Run(context.Background(), now); err != nil {
		t.Fatalf("scan: %v", err)
	}
	before := len(audit.entries)

	observedList(t, repo, "off-upazila", now)
	if len(audit.entries) != before {
		t.Fatalf("the list wrote %d audit entries, want none", len(audit.entries)-before)
	}
}

func TestListObservations_RefusesAnEmptyCaller(t *testing.T) {
	uc := application.NewListObservations(newFakeRepo(), &fakeProblems{}, &fakeOfficialNames{})
	if _, err := uc.Execute(context.Background(), "", time.Now()); !errors.Is(err, domain.ErrNotObserver) {
		t.Fatalf("err = %v, want ErrNotObserver", err)
	}
}

// A failed decoration must not take down the list: a monitor who cannot see the
// official's name still has to be able to see the silence (A.3.4 rule 3).
func TestListObservations_DegradesWhenNamesFail(t *testing.T) {
	now := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)
	repo := newFakeRepo()
	repo.put(silentCase("c-1", "p-1", now.Add(-2*obsD)))

	uc := application.NewListObservations(repo, &fakeProblems{}, &fakeOfficialNames{err: errors.New("directory down")})
	rows, err := uc.Execute(context.Background(), "off-upazila", now)
	if err != nil {
		t.Fatalf("a failed name lookup must not fail the list: %v", err)
	}
	if len(rows) != 1 || rows[0].OfficialName != "" {
		t.Fatalf("want one row with an unknown name, got %+v", rows)
	}
}

// --- the monitor's one action ---

func openObservation(repo *fakeRepo, caseID, observer string) *domain.Observation {
	o := domain.NewObservation(caseID, observer, 1, time.Now().UTC())
	repo.observations = append(repo.observations, o)
	return o
}

func TestNoteObservation_AppendsAndAudits(t *testing.T) {
	repo := newFakeRepo()
	repo.put(silentCase("c-1", "p-1", time.Now().Add(-2*obsD)))
	o := openObservation(repo, "c-1", "off-upazila")
	audit := &fakeAudit{}
	uc := application.NewNoteObservation(repo, audit)

	if _, err := uc.Execute(context.Background(), o.ID, "off-upazila", "চেয়ারম্যানকে মনে করিয়ে দিয়েছি"); err != nil {
		t.Fatalf("note: %v", err)
	}
	if _, err := uc.Execute(context.Background(), o.ID, "off-upazila", "দ্বিতীয়বার তাগাদা দিয়েছি"); err != nil {
		t.Fatalf("second note: %v", err)
	}
	// Append-only: the first note must survive the second.
	if len(repo.notes) != 2 {
		t.Fatalf("stored %d notes, want 2 — notes are never overwritten", len(repo.notes))
	}

	noted := 0
	for _, e := range audit.entries {
		if e.Action == "observation_noted" {
			noted++
			if e.Actor != "off-upazila" {
				t.Fatalf("audit actor = %q, want the monitor", e.Actor)
			}
			if e.TargetID != "p-1" {
				t.Fatalf("audit target = %q, want the problem so it lands on the public trail", e.TargetID)
			}
		}
	}
	if noted != 2 {
		t.Fatalf("wrote %d observation_noted entries, want 2", noted)
	}
}

func TestNoteObservation_OnlyTheObserver(t *testing.T) {
	repo := newFakeRepo()
	repo.put(silentCase("c-1", "p-1", time.Now().Add(-2*obsD)))
	o := openObservation(repo, "c-1", "off-upazila")
	uc := application.NewNoteObservation(repo, &fakeAudit{})

	if _, err := uc.Execute(context.Background(), o.ID, "off-someone-else", "hello"); !errors.Is(err, domain.ErrNotObserver) {
		t.Fatalf("err = %v, want ErrNotObserver", err)
	}
	if len(repo.notes) != 0 {
		t.Fatal("a stranger's note must not be stored")
	}
}

// History is not editable after the fact.
func TestNoteObservation_RefusesAResolvedObservation(t *testing.T) {
	repo := newFakeRepo()
	repo.put(silentCase("c-1", "p-1", time.Now().Add(-2*obsD)))
	o := openObservation(repo, "c-1", "off-upazila")
	o.Resolve(time.Now().UTC())
	uc := application.NewNoteObservation(repo, &fakeAudit{})

	if _, err := uc.Execute(context.Background(), o.ID, "off-upazila", "late note"); !errors.Is(err, domain.ErrObservationResolved) {
		t.Fatalf("err = %v, want ErrObservationResolved", err)
	}
}

func TestNoteObservation_RequiresText(t *testing.T) {
	repo := newFakeRepo()
	repo.put(silentCase("c-1", "p-1", time.Now().Add(-2*obsD)))
	o := openObservation(repo, "c-1", "off-upazila")
	uc := application.NewNoteObservation(repo, &fakeAudit{})

	if _, err := uc.Execute(context.Background(), o.ID, "off-upazila", "   "); !errors.Is(err, domain.ErrEmptyText) {
		t.Fatalf("err = %v, want ErrEmptyText", err)
	}
}

func TestNoteObservation_UnknownObservation(t *testing.T) {
	uc := application.NewNoteObservation(newFakeRepo(), &fakeAudit{})
	if _, err := uc.Execute(context.Background(), "obs-nope", "off-upazila", "hi"); !errors.Is(err, domain.ErrObservationNotFound) {
		t.Fatalf("err = %v, want ErrObservationNotFound", err)
	}
}
