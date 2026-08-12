package http

import (
	"errors"
	"net/http"

	"jago-bahe-backend/internal/problem/application"
	"jago-bahe-backend/internal/problem/domain"
	"jago-bahe-backend/pkg/auth"
	"jago-bahe-backend/pkg/httpx"
)

// Handler serves the problem endpoints, including post-publication moderation and
// the reporter's own edit/withdraw of their report.
type Handler struct {
	report         *application.ReportProblem
	list           *application.ListProblems
	listMine       *application.ListMyProblems
	get            *application.GetProblem
	cast           *application.CastValidationVote
	update         *application.UpdateProblem
	withdraw       *application.WithdrawProblem
	deleteOwn      *application.DeleteProblem
	myVotes        *application.ListMyVotes
	assignments    *application.ListAssignments
	listModeration *application.ListModeration
	approve        *application.ApproveProblem
	reject         *application.RejectProblem
	threshold      int // V, for the "X of V" display in DTOs
}

// NewHandler wires the problem handler with the configured threshold V.
func NewHandler(
	report *application.ReportProblem,
	list *application.ListProblems,
	listMine *application.ListMyProblems,
	get *application.GetProblem,
	cast *application.CastValidationVote,
	update *application.UpdateProblem,
	withdraw *application.WithdrawProblem,
	deleteOwn *application.DeleteProblem,
	myVotes *application.ListMyVotes,
	assignments *application.ListAssignments,
	listModeration *application.ListModeration,
	approve *application.ApproveProblem,
	reject *application.RejectProblem,
	threshold int,
) *Handler {
	return &Handler{
		report: report, list: list, listMine: listMine, get: get, cast: cast,
		update: update, withdraw: withdraw, deleteOwn: deleteOwn, myVotes: myVotes,
		assignments: assignments, listModeration: listModeration, approve: approve,
		reject: reject, threshold: threshold,
	}
}

// voteLookup resolves the calling account's own votes across the given problems
// and returns a per-id accessor for the DTO. One query serves a whole page.
//
// A lookup failure degrades to "no vote known" rather than failing the read:
// myVote is a decoration on a public list, and turning a working feed into a 500
// because a secondary query failed would trade a wrongly-enabled button for an
// unusable page. The cost of degrading is that a re-vote is offered and answered
// 409, which the UI already handles.
func (h *Handler) voteLookup(r *http.Request, problems []domain.Problem) func(string) *string {
	claims, _ := auth.ClaimsFrom(r.Context())
	if claims.UserID == "" || len(problems) == 0 {
		return func(string) *string { return nil }
	}
	ids := make([]string, 0, len(problems))
	for i := range problems {
		ids = append(ids, problems[i].ID)
	}
	votes, err := h.myVotes.Execute(r.Context(), claims.UserID, ids)
	if err != nil {
		votes = nil
	}
	return func(problemID string) *string {
		v, ok := votes[problemID]
		if !ok {
			return nil
		}
		s := string(v)
		return &s
	}
}

// assignmentLookup resolves who is working on each of the given problems and
// returns a per-id accessor for the DTO. One query serves a whole page — the
// per-row alternative is the N+1 that kept this field off the feed until now, so
// if this ever becomes a loop over single reads, the field should come back off.
//
// Like voteLookup, a failure degrades to "not assigned" rather than failing the
// read: the assigned official is a decoration on a public list, and turning a
// working feed into a 500 because a secondary query failed would trade a missing
// name for an unusable page. Unlike voteLookup it runs for anonymous readers too —
// an assignment is public, so there is no caller to have.
func (h *Handler) assignmentLookup(r *http.Request, problems []domain.Problem) func(string) application.AssignmentView {
	if len(problems) == 0 {
		return func(string) application.AssignmentView { return application.AssignmentView{} }
	}
	ids := make([]string, 0, len(problems))
	for i := range problems {
		ids = append(ids, problems[i].ID)
	}
	asgns, err := h.assignments.Execute(r.Context(), ids)
	if err != nil {
		asgns = nil
	}
	return func(problemID string) application.AssignmentView { return asgns[problemID] }
}

// Routes registers the problem endpoints. Reads are public; write routes are
// gated to residents, and the moderation routes to admins (role guards enforced
// server-side, not just in the UI).
//
// GET /api/problems/{id} stays mounted ungated even though a PendingApproval
// problem is now visibility-restricted: the mux's Authenticate middleware is
// best-effort, so a token's claims are present when one is sent, and the use case
// reads them to decide whether the caller may see a pending report. An anonymous
// request simply carries no claims and is treated as a stranger.
func (h *Handler) Routes(mux *http.ServeMux, requirePublic, requireAdmin httpx.Middleware) {
	mux.Handle("POST /api/problems", requirePublic(http.HandlerFunc(h.Report)))
	mux.HandleFunc("GET /api/problems", h.List)
	// The reporter's own view (B12). Resident-gated, unlike the feed: it is derived
	// from the token, so an anonymous caller must get a 401 rather than a silently
	// empty list that reads as "you have filed nothing".
	mux.Handle("GET /api/me/problems", requirePublic(http.HandlerFunc(h.Mine)))
	mux.HandleFunc("GET /api/problems/{id}", h.Get)
	mux.Handle("POST /api/problems/{id}/validate", requirePublic(http.HandlerFunc(h.Validate)))
	mux.HandleFunc("GET /api/problems/{id}/audit", h.Audit)
	// A reporter may edit their own report while it is still unvalidated, and
	// withdraw it any time before it is assigned. Both are resident-gated; the use
	// cases enforce ownership themselves (the middleware only proves a caller).
	mux.Handle("PATCH /api/problems/{id}", requirePublic(http.HandlerFunc(h.Update)))
	mux.Handle("POST /api/problems/{id}/withdraw", requirePublic(http.HandlerFunc(h.Withdraw)))
	// A reporter may also erase their report outright, at any status. This is a
	// different power from withdraw, not a stronger flavour of it — nothing survives
	// it, not even the audit trail. See the use case and CLAUDE.md A.3.3.
	mux.Handle("DELETE /api/problems/{id}", requirePublic(http.HandlerFunc(h.Delete)))

	mux.Handle("GET /api/admin/moderation", requireAdmin(http.HandlerFunc(h.Moderation)))
	mux.Handle("POST /api/admin/problems/{id}/approve", requireAdmin(http.HandlerFunc(h.Approve)))
	mux.Handle("POST /api/admin/problems/{id}/reject", requireAdmin(http.HandlerFunc(h.Reject)))
}

// Report records a new problem for the authenticated resident.
func (h *Handler) Report(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	var req reportRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	if !httpx.RequireFields(w, map[string]string{
		"title":             req.Title,
		"description":       req.Description,
		"areaId":            req.Location.AreaID,
		"pointedOfficialId": req.PointedOfficialID,
	}) {
		return
	}
	p, err := h.report.Execute(r.Context(), application.ReportProblemInput{
		Title:             req.Title,
		Description:       req.Description,
		AreaID:            req.Location.AreaID,
		Address:           req.Location.Address,
		Lat:               req.Location.Lat,
		Lng:               req.Location.Lng,
		ReporterID:        claims.UserID,
		PointedOfficialID: req.PointedOfficialID,
		ProposedSolution:  req.ProposedSolution,
		ImageURL:          req.ImageURL,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toProblemDTO(p, h.threshold, nil))
}

// List returns the feed, filtered by area/status/official query params. The use
// case restricts it to publicly visible statuses.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	problems, err := h.list.Execute(r.Context(), q.Get("area"), q.Get("status"), q.Get("official"))
	if err != nil {
		writeError(w, err)
		return
	}
	// The feed itself is caller-blind — the use case took no caller and chose the
	// same rows for everyone. This only decorates those rows with the reader's own
	// vote, so they are not asked to validate something twice.
	myVote := h.voteLookup(r, problems)
	// Composed here, not in toProblemDTO, for the same reason myVote is: the DTO
	// mapper is shared by every write path, and a parameter there would be an
	// invitation to fill it in per row.
	assigned := h.assignmentLookup(r, problems)
	out := make([]problemDTO, 0, len(problems))
	for i := range problems {
		dto := toProblemDTO(&problems[i], h.threshold, myVote(problems[i].ID))
		applyAssignment(&dto, assigned(problems[i].ID))
		out = append(out, dto)
	}
	httpx.JSON(w, http.StatusOK, out)
}

// Mine returns the caller's own reports, at every status.
//
// The reporter is taken from the token and nothing else: there is no reporter
// query param, and adding one would turn this into an index of what a named
// person filed and an oracle for what they have awaiting screening (Scaffold §5,
// "There is no reporter filter"). The route is resident-gated, so an anonymous
// request is rejected by the middleware before it arrives; the use case's own
// empty-caller guard is the second lock, not the first.
func (h *Handler) Mine(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	problems, err := h.listMine.Execute(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, err)
		return
	}
	// nil myVote: this list is about the progress of the caller's own reports, and
	// a reporter cannot validate their own report anyway. The assignment IS carried,
	// though — "who took up my report" is most of what a reporter comes here to find.
	assigned := h.assignmentLookup(r, problems)
	out := make([]problemDTO, 0, len(problems))
	for i := range problems {
		dto := toProblemDTO(&problems[i], h.threshold, nil)
		applyAssignment(&dto, assigned(problems[i].ID))
		out = append(out, dto)
	}
	httpx.JSON(w, http.StatusOK, out)
}

// Get returns a single problem with its public audit trail. The caller is read
// best-effort from the token (the route is ungated): a PendingApproval problem is
// visible only to its reporter or its union's admin, and everyone else is told it
// does not exist.
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	p, audit, assignment, err := h.get.Execute(r.Context(), r.PathValue("id"), callerFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	dto := toProblemDTO(p, h.threshold, h.voteLookup(r, []domain.Problem{*p})(p.ID))
	dto.Audit = toAuditDTO(audit)
	// The use case resolved this one behind the visibility gate, so it is applied
	// from there rather than through assignmentLookup: a pending report must not
	// pay for an assignment query on its way to a 404.
	applyAssignment(&dto, assignment)
	httpx.JSON(w, http.StatusOK, dto)
}

// Update applies a reporter's edit to their own report. The caller is taken from
// the token; the use case refuses (403) anyone who is not this problem's reporter
// and (409) any problem that has left the editable window.
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	var req updateRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	if !httpx.RequireFields(w, map[string]string{
		"title":       req.Title,
		"description": req.Description,
	}) {
		return
	}
	p, err := h.update.Execute(r.Context(), application.UpdateProblemInput{
		ProblemID:        r.PathValue("id"),
		CallerID:         claims.UserID,
		Title:            req.Title,
		Description:      req.Description,
		Address:          req.Location.Address,
		Lat:              req.Location.Lat,
		Lng:              req.Location.Lng,
		ProposedSolution: req.ProposedSolution,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toProblemDTO(p, h.threshold, nil))
}

// Withdraw takes the reporter's own problem down (a soft delete: it becomes
// Withdrawn and stays on the public record). The use case refuses a non-reporter
// (403) and an already-assigned problem (409).
func (h *Handler) Withdraw(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	var req withdrawRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	p, err := h.withdraw.Execute(r.Context(), r.PathValue("id"), claims.UserID, req.Note)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toProblemDTO(p, h.threshold, nil))
}

// Delete permanently erases the reporter's own problem — the row, everything that
// cascades from it, and its audit trail. The use case refuses a non-reporter (403)
// and an unknown problem (404); there is no window, so there is no 409.
//
// 204 with no body, rather than the deleted problem: there is nothing left to
// serve, and A.5.2 rule 2 is explicit that a bodyless response is a 204 and must
// not go through httpx.JSON (which writes `null`).
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	if err := h.deleteOwn.Execute(r.Context(), r.PathValue("id"), claims.UserID); err != nil {
		writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

// Moderation returns the problems in the acting admin's union that are still
// within the takedown window.
func (h *Handler) Moderation(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	problems, err := h.listModeration.Execute(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]problemDTO, 0, len(problems))
	for i := range problems {
		out = append(out, toProblemDTO(&problems[i], h.threshold, nil))
	}
	httpx.JSON(w, http.StatusOK, out)
}

// Approve publishes a pending report on the acting admin's behalf. The admin must
// belong to the report's own union (the use case enforces it).
func (h *Handler) Approve(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	p, err := h.approve.Execute(r.Context(), r.PathValue("id"), claims.UserID)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toProblemDTO(p, h.threshold, nil))
}

// Reject takes a report down on one of the fixed grounds — either during screening
// (a pending report) or as a pre-assignment takedown of an already-public one.
func (h *Handler) Reject(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	var req rejectRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	if !httpx.RequireFields(w, map[string]string{"reason": req.Reason}) {
		return
	}
	p, err := h.reject.Execute(r.Context(), r.PathValue("id"), claims.UserID, domain.RejectionReason(req.Reason), req.Note)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toProblemDTO(p, h.threshold, nil))
}

// Validate records the authenticated resident's validation vote.
func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	var req validateRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	p, err := h.cast.Execute(r.Context(), r.PathValue("id"), claims.UserID, domain.VoteChoice(req.Vote))
	if err != nil {
		writeError(w, err)
		return
	}
	// Echo the vote just cast. Without this the response to a successful vote says
	// myVote: null, and a client that trusts the response re-enables the button it
	// just used — the one place this field is easiest to get wrong.
	cast := req.Vote
	httpx.JSON(w, http.StatusOK, toProblemDTO(p, h.threshold, &cast))
}

// Audit returns just the public audit trail for a problem. It goes through the
// same use case as Get so the two can never disagree about what a reader may see —
// a pending problem's audit is withheld from a stranger exactly as its detail is.
func (h *Handler) Audit(w http.ResponseWriter, r *http.Request) {
	_, audit, _, err := h.get.Execute(r.Context(), r.PathValue("id"), callerFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toAuditDTO(audit))
}

// callerFrom builds the GetProblem caller from the request's best-effort claims.
// A request with no token yields a zero Caller, which the use case treats as a
// stranger — the right default for the pending-visibility gate.
func callerFrom(r *http.Request) application.Caller {
	claims, _ := auth.ClaimsFrom(r.Context())
	return application.Caller{AccountID: claims.UserID, Role: claims.Role}
}

// writeError maps problem sentinel errors to HTTP status codes in one place.
func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrProblemNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "Problem not found.")
	case errors.Is(err, domain.ErrAlreadyVoted):
		httpx.Error(w, http.StatusConflict, "already_voted", "You have already validated this problem.")
	case errors.Is(err, domain.ErrNotVerified):
		httpx.Error(w, http.StatusForbidden, "not_verified", "Only verified residents can validate.")
	case errors.Is(err, domain.ErrNotAreaResident):
		httpx.Error(w, http.StatusForbidden, "not_area_resident", "Only residents of this area can validate.")
	case errors.Is(err, domain.ErrInvalidVote):
		httpx.Error(w, http.StatusBadRequest, "invalid_vote", "Vote must be 'valid' or 'invalid'.")
	case errors.Is(err, domain.ErrOfficialNotFound):
		httpx.Error(w, http.StatusBadRequest, "invalid_official", "The pointed official does not exist.")
	case errors.Is(err, domain.ErrAreaNotFound):
		httpx.Error(w, http.StatusBadRequest, "invalid_area", "The selected area does not exist.")
	case errors.Is(err, domain.ErrNotPending):
		httpx.Error(w, http.StatusConflict, "not_pending", "This report is not awaiting approval.")
	case errors.Is(err, domain.ErrNotRejectable):
		httpx.Error(w, http.StatusConflict, "not_rejectable", "This problem can no longer be rejected.")
	case errors.Is(err, domain.ErrInvalidRejectionReason):
		httpx.Error(w, http.StatusBadRequest, "invalid_reason", "Reason must be one of: spam, abusive, duplicate, wrong_area.")
	case errors.Is(err, domain.ErrNotUnionAdmin):
		httpx.Error(w, http.StatusForbidden, "not_union_admin", "You can only moderate problems in your own union.")
	case errors.Is(err, domain.ErrStatusNotPublic):
		httpx.Error(w, http.StatusBadRequest, "invalid_status", "That status cannot be listed publicly.")
	case errors.Is(err, domain.ErrNoCaller):
		httpx.Error(w, http.StatusUnauthorized, "unauthorized", "Sign in to see your reports.")
	case errors.Is(err, domain.ErrNotReporter):
		httpx.Error(w, http.StatusForbidden, "not_reporter", "You can only change your own reports.")
	case errors.Is(err, domain.ErrNotEditable):
		httpx.Error(w, http.StatusConflict, "not_editable", "This report can no longer be edited.")
	case errors.Is(err, domain.ErrNotWithdrawable):
		httpx.Error(w, http.StatusConflict, "not_withdrawable", "This report can no longer be withdrawn.")
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal", "Something went wrong.")
	}
}
