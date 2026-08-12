package http

import (
	"errors"
	"net/http"

	"jago-bahe-backend/internal/scorecard/application"
	"jago-bahe-backend/internal/scorecard/domain"
	"jago-bahe-backend/pkg/httpx"
)

// Handler serves the public accountability endpoints: an official's scorecard,
// their public record, and the seat overview.
type Handler struct {
	scorecard *application.OfficialScorecard
	record    *application.OfficialRecord
	seat      *application.SeatOverview
}

// NewHandler wires the scorecard handler.
func NewHandler(scorecard *application.OfficialScorecard, record *application.OfficialRecord, seat *application.SeatOverview) *Handler {
	return &Handler{scorecard: scorecard, record: record, seat: seat}
}

// Routes registers the public accountability endpoints. All are public (any) per
// the contract — this is the public record, and gating it would defeat the point
// — so none takes a role gate. Nothing here is scoped to a caller, so there is no
// per-caller data to protect; the only confidentiality the record owes is to
// unscreened problems, which the query excludes.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/officials/{id}/scorecard", h.OfficialScorecard)
	mux.HandleFunc("GET /api/officials/{id}/record", h.OfficialRecord)
	mux.HandleFunc("GET /api/seat/overview", h.SeatOverview)
}

// OfficialScorecard returns one official's public accountability numbers.
func (h *Handler) OfficialScorecard(w http.ResponseWriter, r *http.Request) {
	stats, err := h.scorecard.Execute(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toScorecardDTO(stats))
}

// OfficialRecord returns one official's public record: their cases, the plans
// they published, and their answers to the community's top suggestions.
func (h *Handler) OfficialRecord(w http.ResponseWriter, r *http.Request) {
	rec, err := h.record.Execute(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toOfficialRecordDTO(rec))
}

// SeatOverview returns the seat-wide accountability roll-up.
func (h *Handler) SeatOverview(w http.ResponseWriter, r *http.Request) {
	overview, err := h.seat.Execute(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toSeatOverviewDTO(overview))
}

// writeError maps scorecard sentinel errors to HTTP status codes in one place.
func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrOfficialNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "Official not found.")
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal", "Something went wrong.")
	}
}
