// Package http exposes the seat's geography over REST. The DTO mirrors the
// frontend's Area (lib/types/models.js) — the domain aggregate is never
// serialized directly.
package http

import (
	"jago-bahe-backend/internal/shared/area/domain"
)

// areaDTO mirrors the frontend Area: {id, name, level, parentId}.
//
// Flat, not a tree. areadomain.Area is flat with a ParentID, the row is flat, and
// the frontend typedef is already field-for-field this shape. A tree would be a
// shape invented at the DTO boundary with no domain counterpart, and the client
// filters by level anyway.
type areaDTO struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Level string `json:"level"`
	// Pointer, not string: the seat is the root and must serialize as JSON null,
	// never "". Both are falsy to the client, so the difference is invisible in
	// any `if (parentId)` check — which is exactly how the empty-body-vs-null bug
	// survived every test in B12 (CLAUDE.md A.5.2 rule 2), one layer up.
	ParentID *string `json:"parentId"`
}

func toAreaDTO(a domain.Area) areaDTO {
	dto := areaDTO{
		ID:    a.ID.String(),
		Name:  a.Name,
		Level: string(a.Level),
	}
	if !a.ParentID.IsZero() {
		parent := a.ParentID.String()
		dto.ParentID = &parent
	}
	return dto
}

func toAreaDTOs(areas []domain.Area) []areaDTO {
	// Non-nil so an empty geography encodes as [] rather than null: the client
	// maps over this, and a null would be a different failure than "no areas".
	out := make([]areaDTO, 0, len(areas))
	for _, a := range areas {
		out = append(out, toAreaDTO(a))
	}
	return out
}
