package http

import (
	"net/http"

	"jago-bahe-backend/internal/resolution/application"
	"jago-bahe-backend/internal/resolution/domain"
	"jago-bahe-backend/pkg/auth"
	"jago-bahe-backend/pkg/httpx"
)

// ConfirmHandler serves the reporting resident's confirmation of a Done case
// (resident-only). It returns the updated case (the problem status is mirrored).
type ConfirmHandler struct {
	confirm *application.ConfirmResolution
}

// NewConfirmHandler wires the confirm handler.
func NewConfirmHandler(confirm *application.ConfirmResolution) *ConfirmHandler {
	return &ConfirmHandler{confirm: confirm}
}

// Routes registers POST /problems/{id}/confirm behind the resident gate.
func (h *ConfirmHandler) Routes(mux *http.ServeMux, requirePublic httpx.Middleware) {
	mux.Handle("POST /api/problems/{id}/confirm", requirePublic(http.HandlerFunc(h.Confirm)))
}

func (h *ConfirmHandler) Confirm(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	var req confirmRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	c, err := h.confirm.Execute(r.Context(), r.PathValue("id"), claims.UserID, domain.ConfirmationOutcome(req.Outcome))
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toCaseDTO(c))
}

// AdjudicateHandler serves the authority's/moderator's verdict on a blocker.
// TODO(contract): this endpoint (POST /admin/obstacles/{id}/adjudicate) is not yet
// in the API contract (Scaffold Spec §5) — it is defined here in B6. It is gated
// to admins (the neutral moderator role) for the pilot; binding it to the blocker's
// named higher authority specifically is a later refinement.
type AdjudicateHandler struct {
	adjudicate *application.AdjudicateObstacle
}

// NewAdjudicateHandler wires the adjudicate handler.
func NewAdjudicateHandler(adjudicate *application.AdjudicateObstacle) *AdjudicateHandler {
	return &AdjudicateHandler{adjudicate: adjudicate}
}

// Routes registers POST /admin/obstacles/{id}/adjudicate behind the admin gate.
func (h *AdjudicateHandler) Routes(mux *http.ServeMux, requireAdmin httpx.Middleware) {
	mux.Handle("POST /api/admin/obstacles/{id}/adjudicate", requireAdmin(http.HandlerFunc(h.Adjudicate)))
}

func (h *AdjudicateHandler) Adjudicate(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	var req adjudicateRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	if req.Decision != "confirm" && req.Decision != "deny" {
		httpx.Error(w, http.StatusBadRequest, "invalid_decision", "Decision must be 'confirm' or 'deny'.")
		return
	}
	b, err := h.adjudicate.Execute(r.Context(), r.PathValue("id"), claims.UserID, req.Decision == "confirm")
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toObstacleDTO(b))
}
