package httpx

import (
	"net/http"
	"time"
)

// dateLayout is the only shape a date may take on the wire: a calendar day,
// zero-padded. Never a timestamp — a client that can send an instant can send one
// in a zone the server did not choose, and then "which day is this in?" has two
// answers. time.Parse with this layout rejects `2026-7-1` and `2026-02-31`
// outright, which is why unpadded and impossible dates are 400s rather than
// silently normalised.
const dateLayout = "2006-01-02"

// dhaka is the seat's civil day, fixed at UTC+6.
//
// NOT time.LoadLocation("Asia/Dhaka"), and that is a correctness matter rather
// than a style one: the production image is alpine with only ca-certificates
// installed (see Dockerfile), so it carries NO tzdata — LoadLocation errors there
// while succeeding on every developer machine, which is the worst possible shape
// for a failure. Bangladesh has never observed DST, so a fixed offset is also the
// honest model of the thing, not a shortcut around the lookup.
//
// It is deliberately not config either. The three tunables in config are numbers
// to be set during the pilot (V, D, R); a seat's civil day is geography.
var dhaka = time.FixedZone("Asia/Dhaka", 6*60*60)

// ParseDateRange reads ?from= and ?to= as INCLUSIVE calendar days in Asia/Dhaka
// and returns them as instants, half-open: [from, to).
//
// Both are optional; an absent or empty value yields the zero time, which every
// Filter treats as "unset". `to` comes back as midnight of the day AFTER the one
// requested, so a caller compares `created_at < to`. That asymmetry is what makes
// "?to= includes the whole day" exact, where `<= to` would exclude the day
// entirely and `<= to + 1 day - 1µs` would depend on the column's precision.
//
// It writes the 400 itself and returns false, following Decode and RequireFields,
// so a handler is three lines and cannot forget the error path. Two failures are
// distinguished because they are different mistakes: invalid_date is a malformed
// value, invalid_date_range is a well-formed but inverted one. An inverted range
// is REFUSED rather than answered with an empty list, for the same reason
// ErrStatusNotPublic is a 400 — an empty list is indistinguishable from "nothing
// happened then", so a transposed pair of fields would read as a quiet seat.
//
// This lives in pkg/httpx rather than in either context because BOTH the problem
// feed and the seat's activity record take a range. Two copies is two places that
// can come to disagree about what a day is, silently.
func ParseDateRange(w http.ResponseWriter, r *http.Request) (from, to time.Time, ok bool) {
	q := r.URL.Query()

	from, ok = parseDay(w, q.Get("from"))
	if !ok {
		return time.Time{}, time.Time{}, false
	}

	to, ok = parseDay(w, q.Get("to"))
	if !ok {
		return time.Time{}, time.Time{}, false
	}
	if !to.IsZero() {
		// The requested day is included, so the exclusive bound is the next
		// midnight. AddDate rather than Add(24h): it is a CALENDAR day, and the
		// two only coincide in a zone without transitions. Dhaka is such a zone
		// today; writing the calendar arithmetic anyway means this stays correct
		// if that ever stops being true.
		to = to.AddDate(0, 0, 1)
	}

	if !from.IsZero() && !to.IsZero() && !from.Before(to) {
		Error(w, http.StatusBadRequest, "invalid_date_range", "The start date must not be after the end date.")
		return time.Time{}, time.Time{}, false
	}

	return from, to, true
}

// parseDay turns one YYYY-MM-DD value into midnight Dhaka on that day. An empty
// value is not an error — it is the absence of a bound — and yields the zero time.
func parseDay(w http.ResponseWriter, value string) (time.Time, bool) {
	if value == "" {
		return time.Time{}, true
	}

	day, err := time.ParseInLocation(dateLayout, value, dhaka)
	if err != nil {
		Error(w, http.StatusBadRequest, "invalid_date", "Dates must be calendar days in the form YYYY-MM-DD.")
		return time.Time{}, false
	}
	return day, true
}
