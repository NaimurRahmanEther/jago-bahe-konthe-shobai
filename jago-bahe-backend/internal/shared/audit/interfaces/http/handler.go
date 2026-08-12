package http

import (
	"net/http"
	"strconv"

	"jago-bahe-backend/internal/shared/audit/application"
	"jago-bahe-backend/internal/shared/audit/domain"
	"jago-bahe-backend/pkg/httpx"
)

// Handler serves the seat's public decision record.
type Handler struct {
	list    *application.ListActivity
	actors  application.Actors
	problem application.Problems
}

// NewHandler wires the activity handler.
func NewHandler(list *application.ListActivity, actors application.Actors, problems application.Problems) *Handler {
	return &Handler{list: list, actors: actors, problem: problems}
}

// Routes registers the public activity endpoint. Ungated, taking no role gate at
// all — this is the record of public power exercised over public reports, and the
// people it exists to inform are residents.
//
// Publishing it does NOT soften "oversight, not override" (Scaffold §2); it widens
// who does the overseeing. There is still no endpoint anywhere that reverses one
// of these decisions, and none may be added — the remedy for an admin abusing
// their position is removing them in public, which is what this feed makes
// possible.
func (h *Handler) Routes(mux *http.ServeMux) {
	mux.HandleFunc("GET /api/seat/activity", h.List)
}

// List returns recent public decisions across the seat, newest first.
//
// Not audited: reads never are (A.4.4). An entry per view of the accountability
// feed would publish who is watching whom — surveillance wearing the costume of
// transparency.
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	entries, err := h.list.Execute(r.Context(), limit)
	if err != nil {
		httpx.Error(w, http.StatusInternalServerError, "internal_error", "Could not load the activity record.")
		return
	}

	actorNames, problemTitles := h.resolve(r, entries)
	httpx.JSON(w, http.StatusOK, toActivityDTOs(entries, actorNames, problemTitles))
}

// resolve turns the ids on a page of entries into names, in two batch queries —
// one per lookup, never one per row (A.3.4).
//
// Both lookups DEGRADE rather than fail. A name that will not resolve costs the
// reader an id where they wanted a name; a 500 costs them the entire public
// accountability record because a secondary query had a bad day. The first is a
// blemish, the second defeats the purpose of the page.
func (h *Handler) resolve(r *http.Request, entries []domain.AuditEntry) (actorNames, problemTitles map[string]string) {
	actorNames, problemTitles = map[string]string{}, map[string]string{}
	if len(entries) == 0 {
		return actorNames, problemTitles
	}

	actorIDs := make([]string, 0, len(entries))
	problemIDs := make([]string, 0, len(entries))
	seenActor, seenProblem := map[string]bool{}, map[string]bool{}
	for _, e := range entries {
		// The platform acting by rule has no name to look up, and asking for one
		// would put a junk id in every batch.
		if e.Actor != domain.ActorSystem && !seenActor[e.Actor] {
			seenActor[e.Actor] = true
			actorIDs = append(actorIDs, e.Actor)
		}
		if e.TargetType == "problem" && !seenProblem[e.TargetID] {
			seenProblem[e.TargetID] = true
			problemIDs = append(problemIDs, e.TargetID)
		}
	}

	if names, err := h.actors.NamesByIDs(r.Context(), actorIDs); err == nil {
		actorNames = names
	}
	if titles, err := h.problem.TitlesByIDs(r.Context(), problemIDs); err == nil {
		problemTitles = titles
	}
	return actorNames, problemTitles
}
