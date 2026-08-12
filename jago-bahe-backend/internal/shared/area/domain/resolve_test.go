package domain_test

import (
	"context"
	"errors"
	"testing"

	"jago-bahe-backend/internal/shared/area/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// fakeAreas is an in-memory area Repository for the resolution tests.
type fakeAreas map[string]*domain.Area

func (f fakeAreas) GetByID(_ context.Context, id valueobject.AreaID) (*domain.Area, error) {
	a, ok := f[id.String()]
	if !ok {
		return nil, domain.ErrAreaNotFound
	}
	return a, nil
}

func (f fakeAreas) List(context.Context) ([]domain.Area, error) { return nil, nil }

func (f fakeAreas) Children(context.Context, valueobject.AreaID) ([]domain.Area, error) {
	return nil, nil
}

func area(id string, level domain.Level, parent string) *domain.Area {
	return &domain.Area{
		ID:       valueobject.AreaID(id),
		Level:    level,
		ParentID: valueobject.AreaID(parent),
	}
}

func TestUnionOfArea(t *testing.T) {
	tests := []struct {
		name string
		area *domain.Area
		want string
	}{
		{"a ward resolves to its parent union", area("ward-1", domain.LevelWard, "union-1"), "union-1"},
		{"a union is its own union", area("union-1", domain.LevelUnion, "upazila-1"), "union-1"},
		{"an upazila is returned as-is", area("upazila-1", domain.LevelUpazila, "seat-1"), "upazila-1"},
		{"a seat is returned as-is", area("seat-1", domain.LevelSeat, ""), "seat-1"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := domain.UnionOfArea(tt.area); got != tt.want {
				t.Fatalf("UnionOfArea(%s) = %q, want %q", tt.area.ID, got, tt.want)
			}
		})
	}
}

func TestResolveUnion(t *testing.T) {
	areas := fakeAreas{
		"ward-1":    area("ward-1", domain.LevelWard, "union-1"),
		"union-1":   area("union-1", domain.LevelUnion, "upazila-1"),
		"upazila-1": area("upazila-1", domain.LevelUpazila, "seat-1"),
	}

	tests := []struct {
		name    string
		areaID  string
		want    string
		wantErr error
	}{
		{"ward walks up to its union", "ward-1", "union-1", nil},
		{"union resolves to itself", "union-1", "union-1", nil},
		{"upazila resolves to itself", "upazila-1", "upazila-1", nil},
		{"unknown area errors", "nope", "", domain.ErrAreaNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := domain.ResolveUnion(context.Background(), areas, tt.areaID)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ResolveUnion(%q) err = %v, want %v", tt.areaID, err, tt.wantErr)
			}
			if got != tt.want {
				t.Fatalf("ResolveUnion(%q) = %q, want %q", tt.areaID, got, tt.want)
			}
		})
	}
}
