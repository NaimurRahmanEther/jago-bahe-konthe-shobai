package httpx_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"jago-bahe-backend/pkg/httpx"
)

// dhaka mirrors the fixed zone the parser uses. The test constructs its
// expectations independently rather than importing the parser's own variable, so
// a change to the offset fails here instead of agreeing with itself.
var dhaka = time.FixedZone("Asia/Dhaka", 6*60*60)

func parse(t *testing.T, query string) (from, to time.Time, ok bool, rec *httptest.ResponseRecorder) {
	t.Helper()
	rec = httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/api/problems?"+query, nil)
	from, to, ok = httpx.ParseDateRange(rec, r)
	return from, to, ok, rec
}

func TestParseDateRange(t *testing.T) {
	tests := []struct {
		name     string
		query    string
		wantOK   bool
		wantFrom time.Time
		wantTo   time.Time
	}{
		{
			name:   "both absent leaves both zero",
			query:  "",
			wantOK: true,
		},
		{
			name:   "empty values are the same as absent",
			query:  "from=&to=",
			wantOK: true,
		},
		{
			name:     "from only",
			query:    "from=2026-07-01",
			wantOK:   true,
			wantFrom: time.Date(2026, 7, 1, 0, 0, 0, 0, dhaka),
		},
		{
			// The whole point of the asymmetry: ?to= names a day the caller wants
			// INCLUDED, and the parser hands back the instant AFTER it so the SQL
			// can compare `created_at < to`.
			name:   "to resolves to midnight of the FOLLOWING day",
			query:  "to=2026-07-27",
			wantOK: true,
			wantTo: time.Date(2026, 7, 28, 0, 0, 0, 0, dhaka),
		},
		{
			name:     "both",
			query:    "from=2026-07-01&to=2026-07-27",
			wantOK:   true,
			wantFrom: time.Date(2026, 7, 1, 0, 0, 0, 0, dhaka),
			wantTo:   time.Date(2026, 7, 28, 0, 0, 0, 0, dhaka),
		},
		{
			// from == to is a legal one-day window, not an inverted range. The
			// "today" preset sends exactly this, so refusing it would break the
			// single most-used control on the feed.
			name:     "from equal to to is a one-day window",
			query:    "from=2026-07-27&to=2026-07-27",
			wantOK:   true,
			wantFrom: time.Date(2026, 7, 27, 0, 0, 0, 0, dhaka),
			wantTo:   time.Date(2026, 7, 28, 0, 0, 0, 0, dhaka),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			from, to, ok, rec := parse(t, tt.query)

			if ok != tt.wantOK {
				t.Fatalf("ok = %v, want %v (body %q)", ok, tt.wantOK, rec.Body.String())
			}
			if !from.Equal(tt.wantFrom) {
				t.Errorf("from = %s, want %s", from, tt.wantFrom)
			}
			if !to.Equal(tt.wantTo) {
				t.Errorf("to = %s, want %s", to, tt.wantTo)
			}
			if rec.Body.Len() != 0 {
				t.Errorf("wrote a body on a valid range: %q", rec.Body.String())
			}
		})
	}
}

// TestParseDateRangeCoversTheWholeToDay is the "inclusive" proof, and it is the
// reason `to` is exclusive-of-the-next-midnight rather than `<= to`.
//
// The last representable instant of 27 July in Dhaka must fall INSIDE a range
// whose ?to= is 2026-07-27. An implementation that resolved `to` to midnight at
// the START of the 27th would exclude the entire day the caller asked for, and
// one that subtracted a microsecond would depend on the column's precision.
func TestParseDateRangeCoversTheWholeToDay(t *testing.T) {
	_, to, ok, _ := parse(t, "to=2026-07-27")
	if !ok {
		t.Fatal("ok = false")
	}

	lastMoment := time.Date(2026, 7, 27, 23, 59, 59, 999999999, dhaka)
	if !lastMoment.Before(to) {
		t.Errorf("%s is not < %s — the last instant of the requested day falls outside the range", lastMoment, to)
	}

	firstOfNextDay := time.Date(2026, 7, 28, 0, 0, 0, 0, dhaka)
	if firstOfNextDay.Before(to) {
		t.Errorf("%s is < %s — the range leaks into the following day", firstOfNextDay, to)
	}
}

// TestParseDateRangeAnchorsOnDhakaNotUTC is the test that fails, deterministically
// and at any hour, if anyone parses these dates in UTC.
//
// A report filed at 05:00 Dhaka on 27 July is 23:00 UTC on 26 July. Anchored on
// the Dhaka civil day it belongs to `from=2026-07-27`; anchored on UTC it falls
// out of it — so the feed would silently drop six hours of every day's reports,
// and only for the people awake in them.
func TestParseDateRangeAnchorsOnDhakaNotUTC(t *testing.T) {
	from, to, ok, _ := parse(t, "from=2026-07-27&to=2026-07-27")
	if !ok {
		t.Fatal("ok = false")
	}

	earlyMorning := time.Date(2026, 7, 27, 5, 0, 0, 0, dhaka)
	if earlyMorning.Before(from) {
		t.Errorf("a report filed at %s falls before the range start %s — this is the UTC-anchoring bug", earlyMorning, from)
	}
	if !earlyMorning.Before(to) {
		t.Errorf("a report filed at %s is not before the range end %s", earlyMorning, to)
	}

	// The mirror: 23:00 UTC on the 26th IS 05:00 Dhaka on the 27th, so the same
	// instant expressed in UTC must land inside the same window.
	if sameInstantUTC := earlyMorning.UTC(); sameInstantUTC.Before(from) {
		t.Errorf("the same instant in UTC (%s) falls outside the Dhaka day", sameInstantUTC)
	}
}

func TestParseDateRangeRefusesMalformedDates(t *testing.T) {
	tests := []struct {
		name  string
		query string
	}{
		{"day-first", "from=27-07-2026"},
		{"unpadded month and day", "from=2026-7-1"},
		{"a full timestamp", "from=2026-07-27T00:00:00Z"},
		{"a slashed date", "from=2026/07/27"},
		{"not a date at all", "from=garbage"},
		{"a month with no day", "from=2026-07"},
		{"malformed to", "to=27-07-2026"},
		{"an impossible day", "from=2026-02-31"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, ok, rec := parse(t, tt.query)

			if ok {
				t.Fatal("ok = true, want false")
			}
			assertErrorBody(t, rec, http.StatusBadRequest, "invalid_date")
		})
	}
}

// TestParseDateRangeRefusesAnInvertedRange pins that from > to is REFUSED rather
// than answered with an empty list.
//
// An empty list is indistinguishable from "nothing happened in that window",
// which is the failure A.3.5 rule 2 records for the activity feed: a caller who
// transposed two fields would read a quiet seat instead of a bad request. Same
// reasoning as ErrStatusNotPublic being a 400 rather than a silent no-op.
func TestParseDateRangeRefusesAnInvertedRange(t *testing.T) {
	_, _, ok, rec := parse(t, "from=2026-07-27&to=2026-07-01")
	if ok {
		t.Fatal("ok = true, want false")
	}
	assertErrorBody(t, rec, http.StatusBadRequest, "invalid_date_range")
}

func assertErrorBody(t *testing.T, rec *httptest.ResponseRecorder, status int, code string) {
	t.Helper()

	if rec.Code != status {
		t.Errorf("status = %d, want %d", rec.Code, status)
	}
	var body httpx.ErrorBody
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("body is not the error envelope: %v (%q)", err, rec.Body.String())
	}
	if body.Code != code {
		t.Errorf("code = %q, want %q", body.Code, code)
	}
	if body.Message == "" {
		t.Error("message is empty")
	}
}
