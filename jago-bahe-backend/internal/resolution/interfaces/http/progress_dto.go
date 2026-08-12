package http

import (
	"jago-bahe-backend/internal/resolution/domain"
)

// progressDTO is the public projection of a case, served per problem (B12).
//
// It is deliberately NOT caseDTO. That shape carries monitorOfficialId,
// escalationLevel and disputeReason — the supervision ladder and the official's
// own dispute, which are working internals of the official's view. Publishing
// them here would disclose, on every problem page, how far up the ladder a case
// has climbed.
//
// The field set is scorecard/domain.CaseRecord — B10's already-settled decision
// about what a case discloses publicly — plus the updates and evidence, which are
// the running account a resident is actually here to read. Nothing else is added
// without a matching amendment to Scaffold §2 "The public record".
type progressDTO struct {
	CaseID    string `json:"caseId"`
	ProblemID string `json:"problemId"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
	Deadline  string `json:"deadline"`
	// Silence is itself a fact worth publishing, so a case with no acknowledgement
	// is still reported rather than hidden.
	Acknowledged   bool    `json:"acknowledged"`
	AcknowledgedAt *string `json:"acknowledgedAt"`

	Plan     *progressPlanDTO `json:"plan"`
	Updates  []updateDTO      `json:"updates"`
	Evidence []evidenceDTO    `json:"evidence"`

	// Obstacles is the case's obstacle history — the public "cause of not solving".
	// It is served here, on the one public projection of a case, so the reason a
	// case stalled stays on the record after it leaves Blocked (obstacle denied →
	// InProgress, or a resident reopened it), not only while the live judgment
	// widget shows it. The shape is the same obstacleDTO the official's caseDTO
	// carries, so the two public views of one case stay byte-identical (Concept §8:
	// an obstacle is public by design — this is persistence, not new disclosure).
	Obstacles []obstacleDTO `json:"obstacles"`

	// BlockedOnHigherAuthority is the fairness nuance the public view owes the
	// official: a case waiting on an obstacle a named authority judged real is not
	// their failure, and must not read as neglect.
	BlockedOnHigherAuthority bool `json:"blockedOnHigherAuthority"`
}

// progressPlanDTO carries the answered-suggestion snapshot that planDTO omits.
// The pairing of the community's question with the official's answer is the point
// of the public plan (Concept §4, Figure 3), and the snapshot — never today's top
// — is what it must be paired with.
type progressPlanDTO struct {
	ID                 string        `json:"id"`
	Strategy           string        `json:"strategy"`
	TimelineWeeks      int           `json:"timelineWeeks"`
	Obstacles          string        `json:"obstacles"`
	SuggestionResponse string        `json:"suggestionResponse"`
	AnsweredSuggestion string        `json:"answeredSuggestion"`
	Tasks              []planTaskDTO `json:"tasks"`
	CreatedAt          string        `json:"createdAt"`
}

func toProgressDTO(c *domain.Case) progressDTO {
	dto := progressDTO{
		CaseID:                   c.ID,
		ProblemID:                c.ProblemID,
		Status:                   string(c.Status),
		CreatedAt:                c.CreatedAt.Format(timeLayout),
		Deadline:                 c.Deadline.Format(timeLayout),
		Acknowledged:             c.AcknowledgedAt != nil,
		AcknowledgedAt:           ptrTime(c.AcknowledgedAt),
		Updates:                  make([]updateDTO, 0, len(c.Updates)),
		Evidence:                 make([]evidenceDTO, 0, len(c.Evidence)),
		Obstacles:                make([]obstacleDTO, 0, len(c.Obstacles)),
		BlockedOnHigherAuthority: c.BlockedOnHigherAuthority(),
	}
	if c.Plan != nil {
		dto.Plan = &progressPlanDTO{
			ID:                 c.Plan.ID,
			Strategy:           c.Plan.Strategy,
			TimelineWeeks:      c.Plan.TimelineWeeks,
			Obstacles:          c.Plan.Obstacles,
			SuggestionResponse: c.Plan.SuggestionResponse,
			AnsweredSuggestion: c.Plan.AnsweredSuggestionTxt,
			Tasks:              toPlanTaskDTOs(c.Plan.Tasks),
			CreatedAt:          c.Plan.CreatedAt.Format(timeLayout),
		}
	}
	for _, u := range c.Updates {
		dto.Updates = append(dto.Updates, updateDTO{
			ID: u.ID, CaseID: u.CaseID, Kind: string(u.Kind), Text: u.Text,
			CreatedAt: u.CreatedAt.Format(timeLayout),
		})
	}
	for _, e := range c.Evidence {
		dto.Evidence = append(dto.Evidence, evidenceDTO{
			ID: e.ID, CaseID: e.CaseID, BeforeImageURL: e.BeforeImageURL,
			AfterImageURL: e.AfterImageURL, CreatedAt: e.CreatedAt.Format(timeLayout),
		})
	}
	for i := range c.Obstacles {
		dto.Obstacles = append(dto.Obstacles, toObstacleDTO(&c.Obstacles[i]))
	}
	return dto
}
