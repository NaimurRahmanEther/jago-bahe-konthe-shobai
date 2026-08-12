package domain

import (
	"errors"

	"jago-bahe-backend/internal/shared/domain/valueobject"
)

// ErrInvalidTier is returned when an official is bound to an unknown tier.
var ErrInvalidTier = errors.New("invalid tier")

// Official is a directory entry for an elected representative: the public list
// of who can be pointed to. Bound to a tier and the area they serve.
type Official struct {
	ID     string
	Name   string
	Phone  valueobject.PhoneNumber
	Tier   valueobject.Tier
	AreaID valueobject.AreaID
}

// NewOfficial validates the tier binding and constructs an Official.
func NewOfficial(id, name string, phone valueobject.PhoneNumber, tier valueobject.Tier, areaID valueobject.AreaID) (*Official, error) {
	if !tier.Valid() {
		return nil, ErrInvalidTier
	}
	return &Official{ID: id, Name: name, Phone: phone, Tier: tier, AreaID: areaID}, nil
}
