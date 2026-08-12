package domain

import (
	"context"
	"errors"

	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// ErrAreaNotFound is the sentinel returned when an area id does not exist.
var ErrAreaNotFound = errors.New("area not found")

// Repository is the port for area persistence. The Postgres struct in
// infrastructure/postgres implements it; services depend only on this interface.
type Repository interface {
	GetByID(ctx context.Context, id valueobject.AreaID) (*Area, error)
	List(ctx context.Context) ([]Area, error)
	Children(ctx context.Context, parentID valueobject.AreaID) ([]Area, error)
}
