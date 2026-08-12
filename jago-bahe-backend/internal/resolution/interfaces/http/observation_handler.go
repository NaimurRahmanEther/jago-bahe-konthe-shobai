package http

import (
	"encoding/json"
	"net/http"
	"time"

	"jago-bahe-backend/internal/resolution/application"
	"jago-bahe-backend/pkg/auth"
	"jago-bahe-backend/pkg/httpx"
)

// ObservationHandler serves a monitor's view of the officials below them (B19):
// the read side of escalation, where a case its official has gone silent on
// surfaces to the tier above.
//
// There are exactly two routes and only one of them writes. A monitor may look and
// may record what they did about a silence; there is deliberately no route here
// that reassigns the case, takes it over or closes it. Silence moves visibility up
// the ladder and leaves responsibility below (Concept §7) — the same shape as the
// super admin's oversight page, where any button that changed a decision would
// break the rule the page exists to embody.
type ObservationHandler struct {
	list *application.ListObservations
	note *application.NoteObservation
}

// NewObservationHandler wires the handler.
func NewObservationHandler(list *application.ListObservations, note *application.NoteObservation) *ObservationHandler {
	return &ObservationHandler{list: list, note: note}
}

// Routes registers the observation endpoints behind the RequireOfficial gate.
func (h *ObservationHandler) Routes(mux *http.ServeMux, requireOfficial httpx.Middleware) {
	mux.Handle("GET /api/official/observations", requireOfficial(http.HandlerFunc(h.List)))
	mux.Handle("POST /api/official/observations/{id}/note", requireOfficial(http.HandlerFunc(h.Note)))
}

// List returns the cases the caller monitors, most urgent first.
//
// It takes no parameter: the monitor is the caller, read from the JWT, exactly as
// the admin queue's union scope is. An officialId in the query string would let any
// official read any other official's supervision list.
func (h *ObservationHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	rows, err := h.list.Execute(r.Context(), claims.OfficialID, time.Now().UTC())
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toObservedCaseDTOs(rows))
}

type noteObservationRequest struct {
	Text string `json:"text"`
}

// Note records what the monitor did about a silence. The note is appended, never
// overwritten, and lands on the problem's public audit trail.
func (h *ObservationHandler) Note(w http.ResponseWriter, r *http.Request) {
	var req noteObservationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httpx.Error(w, http.StatusBadRequest, "invalid_body", "Invalid request body.")
		return
	}
	claims, _ := auth.ClaimsFrom(r.Context())
	note, err := h.note.Execute(r.Context(), r.PathValue("id"), claims.OfficialID, req.Text)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toObservationNoteDTO(*note))
}
