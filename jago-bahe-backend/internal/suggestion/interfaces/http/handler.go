package http

import (
	"errors"
	"net/http"

	"jago-bahe-backend/internal/suggestion/application"
	"jago-bahe-backend/internal/suggestion/domain"
	"jago-bahe-backend/pkg/auth"
	"jago-bahe-backend/pkg/httpx"
)

// Handler serves the suggestion endpoints.
type Handler struct {
	list      *application.ListSuggestions
	propose   *application.ProposeSuggestion
	upvote    *application.UpvoteSuggestion
	myUpvotes *application.ListMyUpvotes
}

// NewHandler wires the suggestion handler.
func NewHandler(
	list *application.ListSuggestions,
	propose *application.ProposeSuggestion,
	upvote *application.UpvoteSuggestion,
	myUpvotes *application.ListMyUpvotes,
) *Handler {
	return &Handler{list: list, propose: propose, upvote: upvote, myUpvotes: myUpvotes}
}

// Routes registers the suggestion endpoints. Reads are public; write routes are
// gated to residents (server-side role guard, not just the UI).
func (h *Handler) Routes(mux *http.ServeMux, requirePublic httpx.Middleware) {
	mux.HandleFunc("GET /api/problems/{id}/suggestions", h.List)
	mux.Handle("POST /api/problems/{id}/suggestions", requirePublic(http.HandlerFunc(h.Propose)))
	mux.Handle("POST /api/suggestions/{id}/upvote", requirePublic(http.HandlerFunc(h.Upvote)))
}

// List returns a problem's ranked suggestions, each decorated with the reader's
// own upvote.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	sugs, err := h.list.Execute(r.Context(), r.PathValue("id"))
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toSuggestionDTOs(sugs, h.upvoteLookup(r, sugs)))
}

// upvoteLookup resolves the calling account's own upvotes across the given
// suggestions and returns a per-id accessor for the DTO. One query serves a whole
// list; an anonymous reader costs none.
//
// The route is mounted UNGATED, so claims are read best-effort — a reader with no
// token still gets the ranked list, they just get null for every myUpvote.
//
// A lookup failure degrades to "no upvote known" rather than failing the read,
// mirroring the problem handler's voteLookup: the decoration is not worth a 500 on
// a public list. The cost of degrading is that the button is offered again and the
// toggle then withdraws an upvote the reader still holds — recoverable by
// pressing it once more, unlike an unreadable page.
func (h *Handler) upvoteLookup(r *http.Request, sugs []domain.Suggestion) func(string) *bool {
	claims, _ := auth.ClaimsFrom(r.Context())
	if claims.UserID == "" || len(sugs) == 0 {
		return func(string) *bool { return nil }
	}
	ids := make([]string, 0, len(sugs))
	for i := range sugs {
		ids = append(ids, sugs[i].ID)
	}
	upvotes, err := h.myUpvotes.Execute(r.Context(), claims.UserID, ids)
	if err != nil {
		upvotes = nil
	}
	return func(suggestionID string) *bool {
		held := upvotes[suggestionID]
		return &held
	}
}

// Propose records the authenticated resident's suggestion for a problem.
func (h *Handler) Propose(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	var req proposeRequest
	if !httpx.Decode(w, r, &req) {
		return
	}
	s, err := h.propose.Execute(r.Context(), r.PathValue("id"), claims.UserID, req.Text)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusCreated, toSuggestionDTO(s))
}

// Upvote toggles the authenticated resident's upvote on a suggestion.
func (h *Handler) Upvote(w http.ResponseWriter, r *http.Request) {
	claims, _ := auth.ClaimsFrom(r.Context())
	s, upvoted, err := h.upvote.Execute(r.Context(), r.PathValue("id"), claims.UserID)
	if err != nil {
		writeError(w, err)
		return
	}
	dto := toSuggestionDTO(s)
	// Which way the toggle went, so the caller never has to infer it from a count
	// other people are also moving.
	dto.MyUpvote = &upvoted
	httpx.JSON(w, http.StatusOK, dto)
}

// writeError maps suggestion sentinel errors to HTTP status codes in one place.
func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrSuggestionNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "Suggestion not found.")
	case errors.Is(err, domain.ErrProblemNotFound):
		httpx.Error(w, http.StatusNotFound, "not_found", "Problem not found.")
	case errors.Is(err, domain.ErrEmptyText):
		httpx.Error(w, http.StatusBadRequest, "empty_text", "Suggestion text is required.")
	case errors.Is(err, domain.ErrNotVerified):
		httpx.Error(w, http.StatusForbidden, "not_verified", "Only verified residents can suggest or upvote.")
	case errors.Is(err, domain.ErrNotAreaResident):
		httpx.Error(w, http.StatusForbidden, "not_area_resident", "Only residents of this area can suggest or upvote.")
	case errors.Is(err, domain.ErrAreaNotFound):
		httpx.Error(w, http.StatusBadRequest, "invalid_area", "The problem's area does not exist.")
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal", "Something went wrong.")
	}
}
