package http

import (
	"errors"
	"net/http"
	"strconv"

	"jago-bahe-backend/internal/shared/notification/application"
	"jago-bahe-backend/internal/shared/notification/domain"
	"jago-bahe-backend/pkg/auth"
	"jago-bahe-backend/pkg/httpx"
)

// Handler serves the caller's own in-app notifications.
//
// There is no resolve() helper here and no ports on this context, which a reader
// coming from audithttp.Handler will look for. It needs none: a notification
// snapshots its problem title and detail at write time, so a page of them is
// already readable and the read path never touches `problems` at all — which is
// also what stops it becoming an oracle over problem ids.
type Handler struct {
	list     *application.ListNotifications
	unread   *application.UnreadCount
	markRead *application.MarkRead
	markAll  *application.MarkAllRead
}

// NewHandler wires the notification handler.
func NewHandler(
	list *application.ListNotifications,
	unread *application.UnreadCount,
	markRead *application.MarkRead,
	markAll *application.MarkAllRead,
) *Handler {
	return &Handler{list: list, unread: unread, markRead: markRead, markAll: markAll}
}

// Routes registers the four notification endpoints behind one gate.
//
// requireAnyRole admits all four roles, because a notification is about the CALLER
// rather than about their office — a resident, an official, an admin and the super
// admin each have their own. No existing gate fits: RequirePublic is residents
// only, and the admin gates are flat and exclusive by design.
//
// Every route is self-derived from the token. There is no recipient parameter on
// any of them and none may be added — the problems are public, the authorship graph
// is not (A.3.2 rule 1, A.3.9 constraint 1).
func (h *Handler) Routes(mux *http.ServeMux, requireAnyRole httpx.Middleware) {
	mux.Handle("GET /api/me/notifications", requireAnyRole(http.HandlerFunc(h.List)))
	mux.Handle("GET /api/me/notifications/unread-count", requireAnyRole(http.HandlerFunc(h.UnreadCount)))
	mux.Handle("POST /api/me/notifications/{id}/read", requireAnyRole(http.HandlerFunc(h.MarkRead)))
	mux.Handle("POST /api/me/notifications/read-all", requireAnyRole(http.HandlerFunc(h.MarkAllRead)))
}

// List returns the caller's own messages, newest first.
//
// Not audited: reads never are (A.4.4), and the log is public, so an entry per view
// would publish who is reading what.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	items, err := h.list.Execute(r.Context(), recipientsFrom(r), limit)
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, toNotificationDTOs(items))
}

// UnreadCount returns the bell's badge.
func (h *Handler) UnreadCount(w http.ResponseWriter, r *http.Request) {
	n, err := h.unread.Execute(r.Context(), recipientsFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, unreadCountDTO{Count: n})
}

// MarkRead marks one message read. 204, and idempotent.
//
// NOT AUDITED, and that is the one deliberate exception to A.4.4 in this codebase —
// see mark_read.go. A 404 here covers both an unknown id and someone else's, so the
// route is no oracle over notification ids.
func (h *Handler) MarkRead(w http.ResponseWriter, r *http.Request) {
	if err := h.markRead.Execute(r.Context(), r.PathValue("id"), recipientsFrom(r)); err != nil {
		writeError(w, err)
		return
	}
	// Bare WriteHeader, not httpx.JSON(w, 204, nil) — that would write the literal
	// `null` into a body a 204 must not have (A.5.2 rule 2).
	w.WriteHeader(http.StatusNoContent)
}

// MarkAllRead clears the caller's badge and reports how many it cleared.
func (h *Handler) MarkAllRead(w http.ResponseWriter, r *http.Request) {
	n, err := h.markAll.Execute(r.Context(), recipientsFrom(r))
	if err != nil {
		writeError(w, err)
		return
	}
	httpx.JSON(w, http.StatusOK, markAllReadDTO{Marked: n})
}

// recipientsFrom builds the caller's own two identities from their token, and is
// the ONLY way a Recipients value is constructed on the read path. Both halves come
// from the JWT: an official's work is addressed to their DIRECTORY OFFICE id, not
// their account, so a caller who is both needs both.
//
// OfficialID is "" for residents, admins and the super admin, which is exactly why
// the repository's predicate refuses to match an empty id.
func recipientsFrom(r *http.Request) domain.Recipients {
	claims, ok := auth.ClaimsFrom(r.Context())
	if !ok {
		return domain.Recipients{}
	}
	return domain.Recipients{AccountID: claims.UserID, OfficialID: claims.OfficialID}
}

// writeError maps the context's sentinels to status codes, in one place (A.4.8).
func writeError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, domain.ErrNoCaller):
		// 401, never an empty list: "you have no messages" and "we do not know who
		// you are" are different claims and must not look the same.
		httpx.Error(w, http.StatusUnauthorized, "unauthenticated", "Authentication required.")
	case errors.Is(err, domain.ErrNotificationNotFound):
		// 404, NEVER 403. An id that exists and belongs to someone else must be
		// indistinguishable from one that does not exist, or this route enumerates
		// notification ids.
		httpx.Error(w, http.StatusNotFound, "not_found", "Notification not found.")
	default:
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Could not load notifications.")
	}
}
