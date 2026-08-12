package http

import (
	"errors"
	"net/http"

	"jago-bahe-backend/internal/assignment/application"
	"jago-bahe-backend/internal/assignment/domain"
	"jago-bahe-backend/pkg/auth"
	"jago-bahe-backend/pkg/httpx"
)

// Handler serves the assignment endpoints. Routing is backend authority — the
// caller supplies the decision inputs, the backend enforces the rules.
//
// It spans two gates, which is why Routes takes both. The union-level decisions
// are the union admin's; the above-union ones are the super admin's, advised by
// the union admins (B20). Roles stay FLAT — a super admin fails RequireAdmin by
// design and an admin fails RequireSuperAdmin — so each route is mounted behind
// exactly the gate its decision belongs to, and neither gate is widened.
type Handler struct {
	queue   *application.ListQueue
	assign  *application.AssignWithinUnion
	suggest *application.SuggestForwarding
	mine    *application.ListMyForwarding
	pending *application.ListForwardingQueue
	forward *application.ForwardProblem
	// threshold is V, echoed on every queue row for the admin's "X of V" display.
	// Display only — it gates nothing here (A.3.1). Mirrors the problem handler,
	// which carries it for exactly the same reason.
	threshold int
}

// NewHandler wires the assignment handler.
func NewHandler(
	queue *application.ListQueue,
	assign *application.AssignWithinUnion,
	suggest *application.SuggestForwarding,
	mine *application.ListMyForwarding,
	pending *application.ListForwardingQueue,
	forward *application.ForwardProblem,
	threshold int,
) *Handler {
	return &Handler{queue: queue, assign: assign, suggest: suggest, mine: mine, pending: pending, forward: forward, threshold: threshold}
}

// Routes registers the assignment endpoints behind their respective gates.
func (h *Handler) Routes(mux *http.ServeMux, requireAdmin, requireSuperAdmin httpx.Middleware) {
	mux.Handle("GET /api/admin/queue", requireAdmin(http.HandlerFunc(h.Queue)))
	mux.Handle("POST /api/admin/problems/{id}/assign", requireAdmin(http.HandlerFunc(h.Assign)))
	mux.Handle("GET /api/admin/forwarding", requireAdmin(http.HandlerFunc(h.MyForwarding)))
	mux.Handle("POST /api/admin/problems/{id}/suggest-forwarding", requireAdmin(http.HandlerFunc(h.Suggest)))

	mux.Handle("GET /api/super/queue", requireSuperAdmin(http.HandlerFunc(h.ForwardingQueue)))
	mux.Handle("POST /api/super/problems/{id}/forward", requireSuperAdmin(http.HandlerFunc(h.Forward)))
}

// Queue lists the problems this admin's union may still forward to an official —
// Reported as well as Validated, each carrying its validation count. Scoped to the
// caller: RequireAdmin is role-only and knows nothing about geography, so the use
// case resolves the acting admin's union itself, as the screening queue does.
func (h *Handler) Queue(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	items, err := h.queue.Execute(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toQueueDTOs(items, h.threshold))
}

// Assign handles a union-level assignment (confirm or override with a reason).
func (h *Handler) Assign(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	var req assignRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	deadline, ok := parseDeadline(req.Deadline)
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "invalid_deadline", "Deadline must be an ISO-8601 timestamp.")
		return
	}
	a, err := h.assign.Execute(r.Context(), r.PathValue("id"), claims.UserID, req.OfficialID, req.Priority, deadline, req.OverrideReason)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toAssignmentDTO(a))
}

// MyForwarding lists the above-union reports this admin may advise on, with the
// public tally and their own advice. Scope comes from the JWT.
func (h *Handler) MyForwarding(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	rows, err := h.mine.Execute(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toForwardingDTOs(rows, h.threshold))
}

// Suggest records this admin's advice on where an above-union report should go.
// It decides nothing — the super admin forwards (A.3.8).
func (h *Handler) Suggest(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	var req suggestRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	s, err := h.suggest.Execute(r.Context(), r.PathValue("id"), claims.UserID, req.OfficialID, req.Reason)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toSuggestionDTO(*s))
}

// ForwardingQueue lists every above-union report awaiting the super admin.
func (h *Handler) ForwardingQueue(w http.ResponseWriter, r *http.Request) {
	rows, err := h.pending.Execute(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toForwardingDTOs(rows, h.threshold))
}

// Forward assigns an above-union report to an official. A reason is required when
// the choice departs from the advisers' top suggestion or from the official the
// reporter pointed the report at; the backend decides that, not the client.
func (h *Handler) Forward(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	var req forwardRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	deadline, ok := parseDeadline(req.Deadline)
	if !ok {
		httpx.Error(w, http.StatusBadRequest, "invalid_deadline", "Deadline must be an ISO-8601 timestamp.")
		return
	}
	a, err := h.forward.Execute(r.Context(), r.PathValue("id"), claims.UserID, req.OfficialID, req.Priority, deadline, req.Reason)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toAssignmentDTO(a))
}

// writeError maps assignment sentinel errors to HTTP status codes in one place.
func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrProblemNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "Problem not found.")
	case errors.Is(err, domain.ErrOfficialNotFound):
		httpx.Error(w, http.StatusBadRequest, "invalid_official", "The selected official does not exist.")
	case errors.Is(err, domain.ErrNotAssignable):
		httpx.Error(w, http.StatusConflict, "not_assignable", "This problem cannot be assigned in its current state.")
	case errors.Is(err, domain.ErrAlreadyAssigned):
		httpx.Error(w, http.StatusConflict, "already_assigned", "This problem is already assigned.")
	case errors.Is(err, domain.ErrMissingOverrideReason):
		httpx.Error(w, http.StatusBadRequest, "override_reason_required", "Overriding the public's choice requires a public reason.")
	case errors.Is(err, domain.ErrReasonRequired):
		httpx.Error(w, http.StatusBadRequest, "reason_required",
			"Forwarding to an official the admins did not suggest, or the reporter did not point to, requires a public reason.")
	case errors.Is(err, domain.ErrWrongRoute):
		httpx.Error(w, http.StatusConflict, "wrong_route", "This tier is decided by a different route (a union admin assigns; the super admin forwards above-union).")
	case errors.Is(err, domain.ErrNotUnionAdmin):
		httpx.Error(w, http.StatusForbidden, "not_union_admin", "Only an admin of this problem's union may decide it.")
	case errors.Is(err, domain.ErrNotEligibleAdviser):
		httpx.Error(w, http.StatusForbidden, "not_eligible_adviser", "You are not entitled to advise on this problem.")
	case errors.Is(err, domain.ErrInvalidScope):
		httpx.Error(w, http.StatusBadRequest, "invalid_scope", "Scope must match the tier ('upazila' or 'seat').")
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal", "Something went wrong.")
	}
}
