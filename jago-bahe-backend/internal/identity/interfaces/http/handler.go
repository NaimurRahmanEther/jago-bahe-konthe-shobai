package http

import (
	"errors"
	"net/http"
	"strconv"

	"jago-bahe-backend/internal/identity/application"
	"jago-bahe-backend/internal/identity/domain"
	"jago-bahe-backend/internal/shared/domain/valueobject"
	"jago-bahe-backend/pkg/auth"
	"jago-bahe-backend/pkg/httpx"
)

// Handler serves the identity endpoints.
type Handler struct {
	register    *application.RegisterResident
	registerOff *application.RegisterOfficial
	login       *application.Login
	listOff     *application.ListOfficials
	getOff      *application.GetOfficial
	claims      *application.ReviewClaims
	verify      *application.VerifyResident
	oversight   *application.Oversight
}

// NewHandler wires the identity handler.
func NewHandler(
	reg *application.RegisterResident,
	regOff *application.RegisterOfficial,
	login *application.Login,
	list *application.ListOfficials,
	get *application.GetOfficial,
	claims *application.ReviewClaims,
	verify *application.VerifyResident,
	oversight *application.Oversight,
) *Handler {
	return &Handler{
		register: reg, registerOff: regOff, login: login, listOff: list, getOff: get,
		claims: claims, verify: verify, oversight: oversight,
	}
}

// Routes registers the identity endpoints.
//
// The claim routes take both admin kinds through one gate: which of them may
// decide a given claim depends on the office's tier, and that routing is the use
// case's job. Splitting it into /admin/claims and /super/claims would encode a
// domain rule in URL prefixes and let the two drift apart.
func (h *Handler) Routes(mux *http.ServeMux, requireAdmin, requireSuperAdmin, requireClaimReviewer httpx.Middleware) {
	mux.HandleFunc("POST /api/auth/register", h.Register)
	mux.HandleFunc("POST /api/auth/register/official", h.RegisterOfficial)
	mux.HandleFunc("POST /api/auth/login", h.Login)
	mux.HandleFunc("GET /api/officials", h.ListOfficials)
	mux.HandleFunc("GET /api/officials/{id}", h.GetOfficial)

	mux.Handle("GET /api/claims/pending", requireClaimReviewer(http.HandlerFunc(h.PendingClaims)))
	mux.Handle("POST /api/claims/{id}/approve", requireClaimReviewer(http.HandlerFunc(h.ApproveClaim)))
	mux.Handle("POST /api/claims/{id}/reject", requireClaimReviewer(http.HandlerFunc(h.RejectClaim)))

	mux.Handle("GET /api/admin/residents/pending", requireAdmin(http.HandlerFunc(h.PendingResidents)))
	mux.Handle("POST /api/admin/residents/{id}/verify", requireAdmin(http.HandlerFunc(h.VerifyResident)))

	// Oversight is read-only, and deliberately has no counterpart that reverses a
	// decision. Do not add one: see identity/application/oversight.go.
	mux.Handle("GET /api/super/oversight", requireSuperAdmin(http.HandlerFunc(h.Oversight)))
}

// Oversight returns recent moderator decisions across the seat.
func (h *Handler) Oversight(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
	entries, err := h.oversight.Execute(r.Context(), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]oversightEntryDTO, 0, len(entries))
	for _, e := range entries {
		out = append(out, toOversightDTO(e))
	}
	httpx.JSON(w, http.StatusOK, out)
}

// RegisterOfficial creates an unverified official account plus a pending claim to
// an existing directory office, and returns a token.
func (h *Handler) RegisterOfficial(w http.ResponseWriter, r *http.Request) {
	var req registerOfficialRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	if !httpx.RequireFields(w, map[string]string{
		"name":       req.Name,
		"phone":      req.Phone,
		"password":   req.Password,
		"nid":        req.NID,
		"officialId": req.OfficialID,
	}) {
		return
	}
	acc, token, err := h.registerOff.Execute(r.Context(), application.RegisterOfficialInput{
		Name:       req.Name,
		Phone:      req.Phone,
		Password:   req.Password,
		NID:        req.NID,
		OfficialID: req.OfficialID,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, registerOfficialResponse{Token: token, User: toUserDTO(acc)})
}

// PendingClaims returns the claims this reviewer may decide.
func (h *Handler) PendingClaims(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	pending, err := h.claims.ListPending(r.Context(), claims.UserID, domain.Role(claims.Role))
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]pendingClaimDTO, 0, len(pending))
	for _, p := range pending {
		out = append(out, toPendingClaimDTO(p))
	}
	httpx.JSON(w, http.StatusOK, out)
}

// ApproveClaim confirms a claim and binds the account to the office.
func (h *Handler) ApproveClaim(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	c, err := h.claims.Approve(r.Context(), r.PathValue("id"), claims.UserID, domain.Role(claims.Role))
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toClaimDTO(*c))
}

// RejectClaim refuses a claim, with a reason shown to the claimant.
func (h *Handler) RejectClaim(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	var req rejectClaimRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	c, err := h.claims.Reject(r.Context(), r.PathValue("id"), claims.UserID, domain.Role(claims.Role), req.Reason)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toClaimDTO(*c))
}

// PendingResidents returns the admin's own union's unverified residents.
func (h *Handler) PendingResidents(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	residents, err := h.verify.ListPending(r.Context(), claims.UserID)
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]userDTO, 0, len(residents))
	for i := range residents {
		out = append(out, toUserDTO(&residents[i]))
	}
	httpx.JSON(w, http.StatusOK, out)
}

// VerifyResident verifies a resident of the admin's own union — the gate that
// lets them validate, and so the gate that makes V mean anything.
func (h *Handler) VerifyResident(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	if err := h.verify.Execute(r.Context(), r.PathValue("id"), claims.UserID); err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, map[string]any{"verified": true})
}

// Register creates a resident account and returns a token.
func (h *Handler) Register(w http.ResponseWriter, r *http.Request) {
	var req registerRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	if !httpx.RequireFields(w, map[string]string{
		"name":     req.Name,
		"phone":    req.Phone,
		"password": req.Password,
		"nid":      req.NID,
		"unionId":  req.UnionID,
	}) {
		return
	}
	token, err := h.register.Execute(r.Context(), application.RegisterResidentInput{
		Name:     req.Name,
		Phone:    req.Phone,
		Password: req.Password,
		NID:      req.NID,
		UnionID:  req.UnionID,
	})
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, tokenResponse{Token: token})
}

// Login authenticates and returns a token, role, and the user.
func (h *Handler) Login(w http.ResponseWriter, r *http.Request) {
	var req loginRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	res, err := h.login.Execute(r.Context(), application.LoginInput{Phone: req.Phone, Password: req.Password})
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, loginResponse{
		Token: res.Token,
		Role:  string(res.Account.Role),
		User:  toUserDTO(res.Account),
	})
}

// ListOfficials returns the directory.
func (h *Handler) ListOfficials(w http.ResponseWriter, r *http.Request) {
	officials, err := h.listOff.Execute(r.Context())
	if err != nil {
		writeError(w, err)
		return
	}
	out := make([]officialDTO, 0, len(officials))
	for _, o := range officials {
		out = append(out, toOfficialDTO(o))
	}
	httpx.JSON(w, http.StatusOK, out)
}

// GetOfficial returns a single directory entry.
func (h *Handler) GetOfficial(w http.ResponseWriter, r *http.Request) {
	official, err := h.getOff.Execute(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toOfficialDTO(*official))
}

// writeError maps identity sentinel errors to HTTP status codes in one place.
func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrInvalidCredentials):
		httpx.Error(w, http.StatusUnauthorized, "invalid_credentials", "Invalid phone or password.")
	case errors.Is(err, domain.ErrPhoneTaken):
		httpx.Error(w, http.StatusConflict, "phone_taken", "This phone number is already registered.")
	case errors.Is(err, domain.ErrInvalidUnion):
		httpx.Error(w, http.StatusBadRequest, "invalid_union", "The selected union does not exist.")
	case errors.Is(err, valueobject.ErrInvalidPhone):
		httpx.Error(w, http.StatusBadRequest, "invalid_phone", "Enter a valid 11-digit phone number.")
	case errors.Is(err, domain.ErrOfficialNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "Official not found.")
	case errors.Is(err, domain.ErrAccountNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "Account not found.")
	case errors.Is(err, domain.ErrClaimNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "Claim not found.")
	case errors.Is(err, domain.ErrClaimNotPending):
		httpx.Error(w, http.StatusConflict, "claim_decided", "This claim has already been decided.")
	case errors.Is(err, domain.ErrOfficeTaken):
		httpx.Error(w, http.StatusConflict, "office_taken", "Someone else has already been confirmed for this office.")
	case errors.Is(err, domain.ErrAlreadyClaimed):
		httpx.Error(w, http.StatusConflict, "already_claimed", "This account has already claimed an office.")
	case errors.Is(err, domain.ErrNotClaimApprover):
		httpx.Error(w, http.StatusForbidden, "not_approver", "You cannot decide this claim.")
	case errors.Is(err, domain.ErrNotUnionAdmin):
		httpx.Error(w, http.StatusForbidden, "not_union_admin", "You can only act within your own union.")
	case errors.Is(err, domain.ErrNotAResident):
		httpx.Error(w, http.StatusBadRequest, "not_a_resident", "Only resident accounts are verified this way.")
	case errors.Is(err, domain.ErrEmptyReason):
		httpx.Error(w, http.StatusBadRequest, "empty_reason", "A reason is required.")
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal", "Something went wrong.")
	}
}
