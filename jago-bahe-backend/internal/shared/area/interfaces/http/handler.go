package http

import (
	"net/http"

	"jago-bahe-backend/internal/shared/area/application"
	"jago-bahe-backend/pkg/httpx"
)

// Handler serves the seat's public geography (B13).
type Handler struct {
	list *application.ListAreas
}

// NewHandler wires the area handler.
func NewHandler(list *application.ListAreas) *Handler {
	return &Handler{list: list}
}

// Routes registers the geography endpoint. Ungated, taking no role gate at all —
// the public geography of a public seat, read by screens that are themselves
// anonymous (Landing, ProblemFeed, and Register, where a resident picks their
// union before they have an account to gate on).
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/areas", h.List)
}

// List returns every area in the seat.
//
// Not audited: the audit log records state changes, and an entry per read of the
// geography would publish who is reading what while changing nothing.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	areas, err := h.list.Execute(r.Context())
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Could not list areas.")
		return
	}
	httpx.JSON(w, http.StatusOK, toAreaDTOs(areas))
}
