package http

import (
	"errors"
	"net/http"

	"jago-bahe-backend/internal/resolution/application"
	"jago-bahe-backend/internal/resolution/domain"
	"jago-bahe-backend/pkg/auth"
	"jago-bahe-backend/pkg/httpx"
)

// CaseHandler serves the official's case-lifecycle endpoints (official-only). The
// caller's officialId comes from the JWT; every action is guarded to the case's
// owner in the use case.
type CaseHandler struct {
	list     *application.ListCases
	get      *application.GetCase
	ack      *application.AcknowledgeCase
	plan     *application.SubmitPlan
	replan   *application.RevisePlan
	update   *application.PostUpdate
	evidence *application.UploadEvidence
	markDone *application.MarkDone
	obstacle *application.ReportObstacle
	task     *application.CompleteTask
}

// NewCaseHandler wires the official case handler.
func NewCaseHandler(list *application.ListCases, get *application.GetCase, ack *application.AcknowledgeCase, plan *application.SubmitPlan, replan *application.RevisePlan, update *application.PostUpdate, evidence *application.UploadEvidence, markDone *application.MarkDone, obstacle *application.ReportObstacle, task *application.CompleteTask) *CaseHandler {
	return &CaseHandler{list: list, get: get, ack: ack, plan: plan, replan: replan, update: update, evidence: evidence, markDone: markDone, obstacle: obstacle, task: task}
}

// Routes registers the official case endpoints behind the RequireOfficial gate.
func (h *CaseHandler) Routes(mux *http.ServeMux, requireOfficial httpx.Middleware) {
	mux.Handle("GET /api/official/cases", requireOfficial(http.HandlerFunc(h.List)))
	mux.Handle("GET /api/official/cases/{id}", requireOfficial(http.HandlerFunc(h.Get)))
	mux.Handle("POST /api/official/cases/{id}/acknowledge", requireOfficial(http.HandlerFunc(h.Acknowledge)))
	mux.Handle("POST /api/official/cases/{id}/plan", requireOfficial(http.HandlerFunc(h.Plan)))
	mux.Handle("POST /api/official/cases/{id}/replan", requireOfficial(http.HandlerFunc(h.Replan)))
	mux.Handle("POST /api/official/cases/{id}/updates", requireOfficial(http.HandlerFunc(h.Update)))
	mux.Handle("POST /api/official/cases/{id}/evidence", requireOfficial(http.HandlerFunc(h.Evidence)))
	mux.Handle("POST /api/official/cases/{id}/done", requireOfficial(http.HandlerFunc(h.Done)))
	mux.Handle("POST /api/official/cases/{id}/report-obstacle", requireOfficial(http.HandlerFunc(h.ReportObstacle)))
	mux.Handle("POST /api/official/cases/{id}/tasks/{taskId}/complete", requireOfficial(http.HandlerFunc(h.CompleteTask)))
}

func (h *CaseHandler) List(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	cases, err := h.list.Execute(r.Context(), claims.OfficialID)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toCaseDTOs(cases))
}

func (h *CaseHandler) Get(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	c, err := h.get.Execute(r.Context(), r.PathValue("id"), claims.OfficialID)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toCaseDTO(c))
}

func (h *CaseHandler) Acknowledge(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	var req acknowledgeRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	// The raw decision goes to the use case rather than being collapsed to a bool
	// here: comparing to "accept" at the boundary is what silently turned every
	// malformed request into a dispute.
	c, err := h.ack.Execute(r.Context(), r.PathValue("id"), claims.OfficialID, domain.Decision(req.Decision), req.Reason)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toCaseDTO(c))
}

func (h *CaseHandler) Plan(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	var req planRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	c, err := h.plan.Execute(r.Context(), r.PathValue("id"), claims.OfficialID, req.Strategy, req.Tasks, req.Obstacles, req.SuggestionResponse)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toCaseDTO(c))
}

// Replan records a replacement plan to restart a stalled case (Reopened, or
// InProgress after an obstacle was resolved). Legal source states are enforced in
// the use case via the lifecycle; an illegal one maps to 409.
func (h *CaseHandler) Replan(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	var req replanRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	c, err := h.replan.Execute(r.Context(), r.PathValue("id"), claims.OfficialID, req.Strategy, req.Tasks, req.Obstacles, req.SuggestionResponse, req.Reason)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toCaseDTO(c))
}

// CompleteTask marks one weekly task of the case's plan completed.
func (h *CaseHandler) CompleteTask(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	c, err := h.task.Execute(r.Context(), r.PathValue("id"), claims.OfficialID, r.PathValue("taskId"))
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toCaseDTO(c))
}

func (h *CaseHandler) Update(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	var req updateRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	c, err := h.update.Execute(r.Context(), r.PathValue("id"), claims.OfficialID, domain.UpdateKind(req.Kind), req.Text)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toCaseDTO(c))
}

func (h *CaseHandler) Evidence(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	var req evidenceRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	c, err := h.evidence.Execute(r.Context(), r.PathValue("id"), claims.OfficialID, req.BeforeImageURL, req.AfterImageURL)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toCaseDTO(c))
}

func (h *CaseHandler) Done(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	c, err := h.markDone.Execute(r.Context(), r.PathValue("id"), claims.OfficialID)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toCaseDTO(c))
}

func (h *CaseHandler) ReportObstacle(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	var req obstacleRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	c, err := h.obstacle.Execute(r.Context(), r.PathValue("id"), claims.OfficialID, domain.ObstacleCategory(req.Category), req.WhatBlocks, req.WhoUnblocks, req.ProofTried)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toCaseDTO(c))
}

// ObstacleHandler serves the advisory obstacle-judgment endpoints: a public read
// plus resident-only vote/propose/upvote writes.
type ObstacleHandler struct {
	get     *application.GetObstacle
	vote    *application.VoteObstacle
	propose *application.ProposeUnblockingPlan
	upvote  *application.UpvoteUnblockingPlan
}

// NewObstacleHandler wires the obstacle handler.
func NewObstacleHandler(get *application.GetObstacle, vote *application.VoteObstacle, propose *application.ProposeUnblockingPlan, upvote *application.UpvoteUnblockingPlan) *ObstacleHandler {
	return &ObstacleHandler{get: get, vote: vote, propose: propose, upvote: upvote}
}

// Routes registers the obstacle endpoints. The read is public; writes are gated
// to residents (server-side role guard).
func (h *ObstacleHandler) Routes(mux *http.ServeMux, requirePublic httpx.Middleware) {
	mux.HandleFunc("GET /api/problems/{id}/obstacle", h.Get)
	mux.Handle("POST /api/problems/{id}/obstacle/vote", requirePublic(http.HandlerFunc(h.Vote)))
	mux.Handle("POST /api/problems/{id}/obstacle/plans", requirePublic(http.HandlerFunc(h.Propose)))
	mux.Handle("POST /api/obstacle/plans/{planId}/upvote", requirePublic(http.HandlerFunc(h.Upvote)))
}

// Get returns the active blocker under public judgment, or null if none.
func (h *ObstacleHandler) Get(w http.ResponseWriter, r *http.Request) {
	b, ok, err := h.get.Execute(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	if !ok {
		httpx.JSON(w, http.StatusOK, nil)
		return
	}
	dto := toObstacleDTO(b)
	httpx.JSON(w, http.StatusOK, dto)
}

func (h *ObstacleHandler) Vote(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	var req obstacleVoteRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	b, err := h.vote.Execute(r.Context(), r.PathValue("id"), claims.UserID, domain.ObstacleVoteChoice(req.Vote))
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toObstacleDTO(b))
}

func (h *ObstacleHandler) Propose(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	var req unblockingPlanRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	p, err := h.propose.Execute(r.Context(), r.PathValue("id"), claims.UserID, req.Text)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toUnblockingPlanDTO(p))
}

func (h *ObstacleHandler) Upvote(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	p, err := h.upvote.Execute(r.Context(), r.PathValue("planId"), claims.UserID)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toUnblockingPlanDTO(p))
}

// writeError maps resolution sentinel errors to HTTP status codes in one place.
func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrCaseNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "Case not found.")
	case errors.Is(err, domain.ErrProblemNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "Problem not found.")
	case errors.Is(err, domain.ErrObstacleNotFound):
		httpx.Error(w, http.StatusNotFound, "no_obstacle", "This problem has no active obstacle to judge.")
	case errors.Is(err, domain.ErrPlanNotFound):
		httpx.Error(w, http.StatusNotFound, "plan_not_found", "Unblocking plan not found.")
	case errors.Is(err, domain.ErrAssignmentMissing):
		httpx.Error(w, http.StatusConflict, "no_assignment", "This problem has not been assigned yet.")
	case errors.Is(err, domain.ErrNotCaseOwner):
		httpx.Error(w, http.StatusForbidden, "not_case_owner", "This case is assigned to another official.")
	case errors.Is(err, domain.ErrTaskNotFound):
		httpx.Error(w, http.StatusNotFound, "task_not_found", "No such weekly task on this plan.")
	case errors.Is(err, domain.ErrIllegalTransition):
		httpx.Error(w, http.StatusConflict, "illegal_transition", "That action is not allowed from the case's current state.")
	case errors.Is(err, domain.ErrEvidenceRequired):
		httpx.Error(w, http.StatusConflict, "evidence_required", "Attach before/after evidence before marking the case done.")
	case errors.Is(err, domain.ErrEmptyText):
		httpx.Error(w, http.StatusBadRequest, "empty_text", "Required text is missing.")
	case errors.Is(err, domain.ErrInvalidDecision):
		httpx.Error(w, http.StatusBadRequest, "invalid_decision", "Decision must be 'accept' or 'dispute'.")
	case errors.Is(err, domain.ErrInvalidKind):
		httpx.Error(w, http.StatusBadRequest, "invalid_kind", "Update kind must be 'progress' or 'obstacle'.")
	case errors.Is(err, domain.ErrInvalidCategory):
		httpx.Error(w, http.StatusBadRequest, "invalid_category", "Invalid obstacle category.")
	case errors.Is(err, domain.ErrInvalidChoice):
		httpx.Error(w, http.StatusBadRequest, "invalid_choice", "Vote must be 'real' or 'not_convinced'.")
	case errors.Is(err, domain.ErrAlreadyVotedObstacle):
		httpx.Error(w, http.StatusConflict, "already_voted", "You have already judged this obstacle.")
	case errors.Is(err, domain.ErrNotReporter):
		httpx.Error(w, http.StatusForbidden, "not_reporter", "Only the resident who reported this problem can confirm it.")
	case errors.Is(err, domain.ErrInvalidOutcome):
		httpx.Error(w, http.StatusBadRequest, "invalid_outcome", "Outcome must be 'solved' or 'not_solved'.")
	case errors.Is(err, domain.ErrAlreadyAdjudicated):
		httpx.Error(w, http.StatusConflict, "already_adjudicated", "This obstacle has already been adjudicated.")
	case errors.Is(err, domain.ErrNotVerified):
		httpx.Error(w, http.StatusForbidden, "not_verified", "Only verified residents can take part.")
	case errors.Is(err, domain.ErrNotAreaResident):
		httpx.Error(w, http.StatusForbidden, "not_area_resident", "Only residents of this area can take part.")
	case errors.Is(err, domain.ErrAreaNotFound):
		httpx.Error(w, http.StatusBadRequest, "invalid_area", "The problem's area does not exist.")
	case errors.Is(err, domain.ErrObservationNotFound):
		httpx.Error(w, http.StatusNotFound, "observation_not_found", "Observation not found.")
	case errors.Is(err, domain.ErrNotObserver):
		httpx.Error(w, http.StatusForbidden, "not_observer", "You are not the monitor watching this case.")
	case errors.Is(err, domain.ErrObservationResolved):
		httpx.Error(w, http.StatusConflict, "observation_resolved", "This observation closed when the official responded.")
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal", "Something went wrong.")
	}
}
