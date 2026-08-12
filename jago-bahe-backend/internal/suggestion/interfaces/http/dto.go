// Package http exposes the suggestion use cases over REST. DTOs mirror the
// frontend's Suggestion typedef; domain aggregates are never serialized directly.
package http

import (
	suggestiondomain "jago-bahe-backend/internal/suggestion/domain"
)

type proposeRequest struct {
	Text string `json:"text"`
}

// suggestionDTO mirrors the frontend Suggestion. upvoteCount and isTop are
// backend-derived (live count + ranking service).
type suggestionDTO struct {
	ID          string `json:"id"`
	ProblemID   string `json:"problemId"`
	AuthorID    string `json:"authorId"`
	Text        string `json:"text"`
	UpvoteCount int    `json:"upvoteCount"`
	IsTop       bool   `json:"isTop"`
	CreatedAt   string `json:"createdAt"`

	// MyUpvote is the CALLING account's own upvote on this suggestion, or null when
	// they have not upvoted, are not a resident, or are anonymous. It is what lets
	// the UI show an upvote already cast instead of re-offering the button — the
	// state used to be component-local, so a reload re-offered it and the backend
	// answered a request the UI had invited.
	//
	// A pointer with NO omitempty: it must serialize as literal `null`, never
	// vanish. An absent key and a null are both falsy in JavaScript, so a regression
	// to "sometimes missing" would be invisible in the UI and surface only as a
	// wrongly-enabled button (problemDTO.MyVote carries the same reasoning).
	//
	// The counts above are public; THIS is the caller's own row and nobody else's.
	MyUpvote *bool `json:"myUpvote"`
}

func toSuggestionDTO(s *suggestiondomain.Suggestion) suggestionDTO {
	return suggestionDTO{
		ID:          s.ID,
		ProblemID:   s.ProblemID,
		AuthorID:    s.AuthorID,
		Text:        s.Text,
		UpvoteCount: s.UpvoteCount,
		IsTop:       s.IsTop,
		CreatedAt:   s.CreatedAt.Format(timeLayout),
	}
}

// toSuggestionDTOs maps a ranked list, decorating each row with the viewer's own
// upvote. myUpvote is a per-id accessor composed in the handler — the mapper stays
// caller-free, exactly as the problem context's does, so the write paths that reuse
// toSuggestionDTO have no viewer to supply.
func toSuggestionDTOs(sugs []suggestiondomain.Suggestion, myUpvote func(string) *bool) []suggestionDTO {
	out := make([]suggestionDTO, 0, len(sugs))
	for i := range sugs {
		dto := toSuggestionDTO(&sugs[i])
		dto.MyUpvote = myUpvote(sugs[i].ID)
		out = append(out, dto)
	}
	return out
}

const timeLayout = "2006-01-02T15:04:05.000Z07:00"
