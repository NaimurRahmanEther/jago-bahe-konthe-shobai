package valueobject

// Location is where a problem is, resolving to an area (Ward/Union) plus an
// optional human address and coordinates.
type Location struct {
	AreaID  AreaID
	Address string
	Lat     *float64
	Lng     *float64
}
