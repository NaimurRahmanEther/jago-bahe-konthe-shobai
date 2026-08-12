package application_test

import (
	"context"
	"errors"
	"testing"

	"jago-bahe-backend/internal/identity/application"
	"jago-bahe-backend/internal/identity/domain"
	areadomain "jago-bahe-backend/internal/shared/area/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// The pilot geography, cut down to one branch of the ladder.
func ladderAreas() map[string]areadomain.Area {
	return map[string]areadomain.Area{
		"seat-1":    {ID: "seat-1", Name: "Naogaon-2", Level: areadomain.LevelSeat},
		"upazila-1": {ID: "upazila-1", Name: "Dhamoirhat", Level: areadomain.LevelUpazila, ParentID: "seat-1"},
		"union-1":   {ID: "union-1", Name: "Agrodwigun", Level: areadomain.LevelUnion, ParentID: "upazila-1"},
		"ward-1":    {ID: "ward-1", Name: "Ward 1", Level: areadomain.LevelWard, ParentID: "union-1"},
	}
}

func ladderOfficials() []domain.Official {
	return []domain.Official{
		{ID: "off-ward", Name: "Ward Member", Tier: valueobject.TierWardMember, AreaID: "ward-1"},
		{ID: "off-union", Name: "Union Chairman", Tier: valueobject.TierUnionChairman, AreaID: "union-1"},
		{ID: "off-upazila", Name: "Upazila Chairman", Tier: valueobject.TierUpazilaChairman, AreaID: "upazila-1"},
		{ID: "off-mp", Name: "MP", Tier: valueobject.TierMP, AreaID: "seat-1"},
	}
}

type fakeLadderOfficials struct {
	officials []domain.Official
	listErr   error
	listCalls int
}

func (f *fakeLadderOfficials) List(context.Context) ([]domain.Official, error) {
	f.listCalls++
	if f.listErr != nil {
		return nil, f.listErr
	}
	return f.officials, nil
}

func (f *fakeLadderOfficials) GetByID(_ context.Context, id string) (*domain.Official, error) {
	for i := range f.officials {
		if f.officials[i].ID == id {
			return &f.officials[i], nil
		}
	}
	return nil, domain.ErrOfficialNotFound
}

func (f *fakeLadderOfficials) NamesByIDs(context.Context, []string) (map[string]string, error) {
	return nil, nil
}

type fakeLadderAreas struct {
	areas map[string]areadomain.Area
}

func (f fakeLadderAreas) GetByID(_ context.Context, id valueobject.AreaID) (*areadomain.Area, error) {
	a, ok := f.areas[id.String()]
	if !ok {
		return nil, areadomain.ErrAreaNotFound
	}
	return &a, nil
}

func (f fakeLadderAreas) List(context.Context) ([]areadomain.Area, error) { return nil, nil }
func (f fakeLadderAreas) Children(context.Context, valueobject.AreaID) ([]areadomain.Area, error) {
	return nil, nil
}

func TestMonitorChain(t *testing.T) {
	tests := []struct {
		name      string
		officials []domain.Official
		areas     map[string]areadomain.Area
		start     string
		maxRungs  int
		want      []string
	}{
		{
			name:      "a ward official climbs to the upazila chairman, then the MP",
			officials: ladderOfficials(),
			areas:     ladderAreas(),
			start:     "off-ward",
			maxRungs:  3,
			want:      []string{"off-upazila", "off-mp"},
		},
		{
			name:      "a union official climbs the same two rungs",
			officials: ladderOfficials(),
			areas:     ladderAreas(),
			start:     "off-union",
			maxRungs:  3,
			want:      []string{"off-upazila", "off-mp"},
		},
		{
			name:      "an upazila official is monitored by the MP alone",
			officials: ladderOfficials(),
			areas:     ladderAreas(),
			start:     "off-upazila",
			maxRungs:  3,
			want:      []string{"off-mp"},
		},
		{
			name:      "the MP is the top of the ladder and has no monitor",
			officials: ladderOfficials(),
			areas:     ladderAreas(),
			start:     "off-mp",
			maxRungs:  3,
			want:      nil,
		},
		{
			name:      "maxRungs truncates the chain",
			officials: ladderOfficials(),
			areas:     ladderAreas(),
			start:     "off-union",
			maxRungs:  1,
			want:      []string{"off-upazila"},
		},
		{
			name:      "maxRungs of zero asks for nothing",
			officials: ladderOfficials(),
			areas:     ladderAreas(),
			start:     "off-union",
			maxRungs:  0,
			want:      nil,
		},
		{
			// Degrade, never abort: the worker walks this for every open case.
			name:      "an unresolvable area ends the chain without an error",
			officials: []domain.Official{{ID: "off-orphan", Tier: valueobject.TierUnionChairman, AreaID: "union-gone"}},
			areas:     ladderAreas(),
			start:     "off-orphan",
			maxRungs:  3,
			want:      nil,
		},
		{
			name:      "an unknown official yields an empty chain",
			officials: ladderOfficials(),
			areas:     ladderAreas(),
			start:     "off-nobody",
			maxRungs:  3,
			want:      nil,
		},
		{
			// A vacant office must NOT promote the tier above into rung 1 — that
			// would put the case in front of the MP a full window early.
			name: "a vacant upazila chairmanship ends the chain, it does not promote the MP",
			officials: []domain.Official{
				{ID: "off-union", Tier: valueobject.TierUnionChairman, AreaID: "union-1"},
				{ID: "off-mp", Tier: valueobject.TierMP, AreaID: "seat-1"},
			},
			areas:    ladderAreas(),
			start:    "off-union",
			maxRungs: 3,
			want:     nil,
		},
		{
			// One person holding two rungs must appear once: two observations on
			// one official for one case is the same person asked to watch twice.
			name: "an official holding two rungs is deduped",
			officials: []domain.Official{
				{ID: "off-union", Tier: valueobject.TierUnionChairman, AreaID: "union-1"},
				{ID: "off-both", Tier: valueobject.TierUpazilaChairman, AreaID: "upazila-1"},
				{ID: "off-both", Tier: valueobject.TierMP, AreaID: "seat-1"},
			},
			areas:    ladderAreas(),
			start:    "off-union",
			maxRungs: 3,
			want:     []string{"off-both"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			officials := &fakeLadderOfficials{officials: tt.officials}
			got, err := application.MonitorChain(context.Background(), officials, fakeLadderAreas{areas: tt.areas}, tt.start, tt.maxRungs)
			if err != nil {
				t.Fatalf("MonitorChain: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("MonitorChain = %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("MonitorChain = %v, want %v", got, tt.want)
				}
			}
		})
	}
}

// The directory is read once per call, not once per rung: the worker walks this
// for every open case on every tick.
func TestMonitorChainReadsTheDirectoryOnce(t *testing.T) {
	officials := &fakeLadderOfficials{officials: ladderOfficials()}
	if _, err := application.MonitorChain(context.Background(), officials, fakeLadderAreas{areas: ladderAreas()}, "off-ward", 3); err != nil {
		t.Fatalf("MonitorChain: %v", err)
	}
	if officials.listCalls != 1 {
		t.Fatalf("directory listed %d times, want 1", officials.listCalls)
	}
}

func TestMonitorChainPropagatesDirectoryFailure(t *testing.T) {
	boom := errors.New("directory unavailable")
	officials := &fakeLadderOfficials{listErr: boom}
	if _, err := application.MonitorChain(context.Background(), officials, fakeLadderAreas{areas: ladderAreas()}, "off-ward", 3); !errors.Is(err, boom) {
		t.Fatalf("err = %v, want %v", err, boom)
	}
}
