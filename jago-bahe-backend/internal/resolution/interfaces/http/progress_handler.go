package http

import (
	"net/http"

	"jago-bahe-backend/internal/resolution/application"
	"jago-bahe-backend/pkg/httpx"
)

// ProgressHandler serves the public progress of a problem's case (B12).
//
// It lives in the resolution context despite its /api/problems/ URL: handler
// ownership follows the data, not the path. Resolution already serves obstacle,
// obstacle/vote, obstacle/plans and confirm under that prefix. Putting this route
// in the problem context instead would force a problem -> resolution import, and
// resolution's Problems port exists precisely to keep that arrow pointing one way.
type ProgressHandler struct {
	get *application.GetProgress
}

// NewProgressHandler wires the handler.
func NewProgressHandler(get *application.GetProgress) *ProgressHandler {
	return &ProgressHandler{get: get}
}

// Routes registers the progress endpoint. Ungated: the plan is already public per
// official (B10), so the same plan is public per problem, and a gate here would
// lock the public out of the public record.
func (h *ProgressHandler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/problems/{id}/progress", h.Get)
}

// Get returns the case's public progress, or a null case when there is none.
//
// Null rather than 404, always — see GetProgress: an unknown problem, an
// unscreened one, an unassigned one and an unmaterialized case are deliberately
// indistinguishable here, because telling them apart would let anyone enumerate
// what is awaiting screening.
func (h *ProgressHandler) Get(w http.ResponseWriter, r *http.Request) {
	c, ok, err := h.get.Execute(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		httpx.JSON(w, http.StatusOK, nil)
		return
	}
	dto := toProgressDTO(c)
	httpx.JSON(w, http.StatusOK, dto)
}
