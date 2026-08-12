package domain

import (
	"time"

	"jago-bahe-backend/pkg/idgen"
)

// Plan is the official's public plan for a case. SuggestionResponse must
// explicitly answer the community's top suggestion (adopt it or explain why not)
// — the domain requires it non-empty; the http layer surfaces the top suggestion.
//
// AnsweredSuggestionID/Text snapshot the top suggestion as it stood when the plan
// was submitted. "Top" is derived from live upvotes and keeps moving, so the
// snapshot — never today's top — is what every public view pairs the response
// with. Both are empty when the problem had no upvoted suggestion to answer.
//
// Tasks are the plan's week-by-week checklist, one per week. TimelineWeeks is
// derived from them (= len(Tasks)) so every existing "X weeks" reader keeps
// working; it is not a separate input.
type Plan struct {
	ID                    string
	CaseID                string
	Strategy              string
	TimelineWeeks         int
	Obstacles             string
	SuggestionResponse    string
	AnsweredSuggestionID  string
	AnsweredSuggestionTxt string
	Tasks                 []PlanTask
	CreatedAt             time.Time
}

// NewPlan constructs a plan for a case, snapshotting the suggestion it answers and
// building one PlanTask per week from taskTexts (week N = position N). TimelineWeeks
// is derived from the task count rather than taken as a separate number.
func NewPlan(caseID, strategy string, taskTexts []string, obstacles, suggestionResponse, answeredSuggestionID, answeredSuggestionTxt string) *Plan {
	id := idgen.New("plan")
	tasks := make([]PlanTask, 0, len(taskTexts))
	for i, text := range taskTexts {
		tasks = append(tasks, NewPlanTask(id, i+1, text))
	}
	return &Plan{
		ID:                    id,
		CaseID:                caseID,
		Strategy:              strategy,
		TimelineWeeks:         len(tasks),
		Obstacles:             obstacles,
		SuggestionResponse:    suggestionResponse,
		AnsweredSuggestionID:  answeredSuggestionID,
		AnsweredSuggestionTxt: answeredSuggestionTxt,
		Tasks:                 tasks,
		CreatedAt:             time.Now().UTC(),
	}
}
