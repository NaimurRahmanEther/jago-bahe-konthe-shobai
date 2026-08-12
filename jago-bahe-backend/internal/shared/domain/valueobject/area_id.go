package valueobject

// AreaID identifies a node in the seat → upazila → union → ward geography.
type AreaID string

func (a AreaID) String() string { return string(a) }

// IsZero reports whether the id is unset.
func (a AreaID) IsZero() bool { return a == "" }
