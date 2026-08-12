// Package domain (area) models the seat → upazila → union → ward geography that
// decides who a problem can be pointed to and who monitors whom. Pure logic —
// it imports no database or HTTP packages.
package domain

import "jago-bahe-backend/internal/shared/domain/valueobject"

// Level is a node's rung in the area hierarchy.
type Level string

const (
	LevelSeat    Level = "seat"
	LevelUpazila Level = "upazila"
	LevelUnion   Level = "union"
	LevelWard    Level = "ward"
)

// Valid reports whether l is a known level.
func (l Level) Valid() bool {
	switch l {
	case LevelSeat, LevelUpazila, LevelUnion, LevelWard:
		return true
	}
	return false
}

// Area is an aggregate root: one node in the geography tree.
type Area struct {
	ID       valueobject.AreaID
	Name     string
	Level    Level
	ParentID valueobject.AreaID // zero for the seat (root)
}
