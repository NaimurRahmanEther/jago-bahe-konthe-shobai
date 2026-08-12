package application

import (
	"context"
	"errors"
	"testing"

	"jago-bahe-backend/internal/scorecard/domain"
)

// fakeQuery is an in-memory Query for testing the use cases without Postgres.
type fakeQuery struct {
	officials  map[string]bool
	byOff      map[string][]domain.CaseFact
	seat       []domain.CaseFact
	problems   int
	recordsOff map[string][]domain.CaseRecord
}

func (q fakeQuery) OfficialExists(_ context.Context, id string) (bool, error) {
	return q.officials[id], nil
}
func (q fakeQuery) CaseFactsByOfficial(_ context.Context, id string) ([]domain.CaseFact, error) {
	return q.byOff[id], nil
}
func (q fakeQuery) SeatCaseFacts(context.Context) ([]domain.CaseFact, error) { return q.seat, nil }
func (q fakeQuery) SeatProblemCount(context.Context) (int, error)            { return q.problems, nil }
func (q fakeQuery) RecordByOfficial(_ context.Context, id string) ([]domain.CaseRecord, error) {
	return q.recordsOff[id], nil
}

func TestOfficialScorecard_UnknownOfficial(t *testing.T) {
	uc := NewOfficialScorecard(fakeQuery{officials: map[string]bool{}})
	_, err := uc.Execute(context.Background(), "ghost")
	if !errors.Is(err, domain.ErrOfficialNotFound) {
		t.Fatalf("want ErrOfficialNotFound, got %v", err)
	}
}

func TestOfficialScorecard_Aggregates(t *testing.T) {
	q := fakeQuery{
		officials: map[string]bool{"off-1": true},
		byOff: map[string][]domain.CaseFact{"off-1": {
			{Status: "Resolved", Acknowledged: true, ResponseDays: 3},
			{Status: "Blocked", ActiveBlocker: domain.AdjConfirmed},
			{Status: "InProgress"},
		}},
	}
	got, err := NewOfficialScorecard(q).Execute(context.Background(), "off-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := domain.OfficialStats{OfficialID: "off-1", Resolved: 1, Pending: 1, Blocked: 1, AvgResponseDays: 3}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestOfficialRecord_UnknownOfficial(t *testing.T) {
	uc := NewOfficialRecord(fakeQuery{officials: map[string]bool{}})
	_, err := uc.Execute(context.Background(), "ghost")
	if !errors.Is(err, domain.ErrOfficialNotFound) {
		t.Fatalf("want ErrOfficialNotFound, got %v", err)
	}
}

// TestOfficialRecord_ReportsWhatTheyWrote covers the point of the record: an
// official's own words, paired with the question they were actually answering.
func TestOfficialRecord_ReportsWhatTheyWrote(t *testing.T) {
	q := fakeQuery{
		officials: map[string]bool{"off-1": true},
		recordsOff: map[string][]domain.CaseRecord{"off-1": {
			{
				CaseID: "case-1", ProblemID: "prob-1", Title: "pothole", Status: "InProgress",
				HasPlan: true, Strategy: "patch it", SuggestionResponse: "adopting this",
				AnsweredSuggestion: "resurface the lane",
			},
			{CaseID: "case-2", ProblemID: "prob-2", Title: "drain", Status: "Blocked", BlockedOnHigherAuthority: true},
		}},
	}
	got, err := NewOfficialRecord(q).Execute(context.Background(), "off-1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.OfficialID != "off-1" || len(got.Cases) != 2 {
		t.Fatalf("got %+v", got)
	}
	if got.Cases[0].AnsweredSuggestion != "resurface the lane" || got.Cases[0].SuggestionResponse != "adopting this" {
		t.Errorf("the Q&A pairing must survive to the record, got %+v", got.Cases[0])
	}
	// The fairness rule reaches the public record, not just the numbers: a case
	// waiting on a higher authority must not read as this official's neglect.
	if !got.Cases[1].BlockedOnHigherAuthority {
		t.Error("blocked-on-higher-authority flag lost")
	}
}

// TestOfficialRecord_EmptyForNewOfficial: an official with no cases has an empty
// record, not an error — a newly elected member simply has nothing on file yet.
func TestOfficialRecord_EmptyForNewOfficial(t *testing.T) {
	q := fakeQuery{officials: map[string]bool{"off-9": true}}
	got, err := NewOfficialRecord(q).Execute(context.Background(), "off-9")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got.Cases) != 0 {
		t.Fatalf("expected an empty record, got %d cases", len(got.Cases))
	}
}

func TestSeatOverview_Aggregates(t *testing.T) {
	q := fakeQuery{
		seat: []domain.CaseFact{
			{Status: "Resolved"},
			{Status: "Blocked", ActiveBlocker: domain.AdjConfirmed},
			{Status: "InProgress"},
		},
		problems: 9,
	}
	got, err := NewSeatOverview(q).Execute(context.Background())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := domain.SeatOverview{Problems: 9, Cases: 3, Resolved: 1, Pending: 1, Blocked: 1}
	if got != want {
		t.Errorf("got %+v, want %+v", got, want)
	}
}
