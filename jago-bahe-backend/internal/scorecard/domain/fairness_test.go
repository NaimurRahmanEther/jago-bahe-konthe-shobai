package domain

import "testing"

// The fairness rule is the heart of B7: a case blocked on a higher authority (a
// blocker adjudicated real) is never the official's failure; silence, missed
// deadlines, and a pattern of blockers judged excuses are.
func TestAggregateOfficial_Fairness(t *testing.T) {
	tests := []struct {
		name  string
		facts []CaseFact
		want  OfficialStats
	}{
		{
			name:  "no cases is all zero",
			facts: nil,
			want:  OfficialStats{OfficialID: "off-1"},
		},
		{
			name:  "resolved counts as credit",
			facts: []CaseFact{{Status: "Resolved"}},
			want:  OfficialStats{OfficialID: "off-1", Resolved: 1},
		},
		{
			name:  "blocker adjudicated real is the fair bucket, not a failure",
			facts: []CaseFact{{Status: "Blocked", ActiveBlocker: AdjConfirmed}},
			want:  OfficialStats{OfficialID: "off-1", Blocked: 1},
		},
		{
			name:  "blocker still pending adjudication is NOT yet protected (pending)",
			facts: []CaseFact{{Status: "Blocked", ActiveBlocker: AdjPending}},
			want:  OfficialStats{OfficialID: "off-1", Pending: 1},
		},
		{
			name:  "excuse-blocker (denied → bounced back to work) counts against as pending",
			facts: []CaseFact{{Status: "InProgress", ActiveBlocker: AdjNone}},
			want:  OfficialStats{OfficialID: "off-1", Pending: 1},
		},
		{
			name: "silence and open work all count as pending",
			facts: []CaseFact{
				{Status: "Assigned"}, {Status: "Acknowledged"}, {Status: "Planned"},
				{Status: "InProgress"}, {Status: "Reopened"}, {Status: "Done"},
			},
			want: OfficialStats{OfficialID: "off-1", Pending: 6},
		},
		{
			name:  "disputed is neither credit nor blame",
			facts: []CaseFact{{Status: "Disputed"}},
			want:  OfficialStats{OfficialID: "off-1"},
		},
		{
			name: "avg response averages acknowledged cases, rounded to one decimal",
			facts: []CaseFact{
				{Status: "Resolved", Acknowledged: true, ResponseDays: 2},
				{Status: "Done", Acknowledged: true, ResponseDays: 3.5},
				{Status: "Assigned"}, // never acknowledged — excluded from the average
			},
			want: OfficialStats{OfficialID: "off-1", Resolved: 1, Pending: 2, AvgResponseDays: 2.8},
		},
		{
			name: "mixed portfolio buckets correctly",
			facts: []CaseFact{
				{Status: "Resolved", Acknowledged: true, ResponseDays: 1},
				{Status: "Resolved", Acknowledged: true, ResponseDays: 4},
				{Status: "Blocked", ActiveBlocker: AdjConfirmed},
				{Status: "Blocked", ActiveBlocker: AdjPending},
				{Status: "InProgress"},
				{Status: "Disputed"},
			},
			want: OfficialStats{OfficialID: "off-1", Resolved: 2, Pending: 2, Blocked: 1, AvgResponseDays: 2.5},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := AggregateOfficial("off-1", tt.facts)
			if got != tt.want {
				t.Errorf("AggregateOfficial() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

func TestAggregateSeat(t *testing.T) {
	facts := []CaseFact{
		{Status: "Resolved", Acknowledged: true, ResponseDays: 2},
		{Status: "Blocked", ActiveBlocker: AdjConfirmed},
		{Status: "InProgress"},
		{Status: "Disputed"},
	}
	got := AggregateSeat(facts)
	want := SeatOverview{Cases: 4, Resolved: 1, Pending: 1, Blocked: 1, AvgResponseDays: 2}
	if got != want {
		t.Errorf("AggregateSeat() = %+v, want %+v", got, want)
	}
}
