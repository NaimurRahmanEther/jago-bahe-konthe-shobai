package domain

import "errors"

// ErrOfficialNotFound is returned when a scorecard is requested for an official
// id that is not in the directory. The HTTP layer maps it to 404.
var ErrOfficialNotFound = errors.New("scorecard: official not found")
