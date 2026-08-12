package domain

import "time"

// LastActivityAt returns the most recent moment the ASSIGNEE acted on this case,
// or the zero time if they never have. It is the reading of the silence clock —
// the second clock, distinct from the escalation clock, which measures lateness
// against the fixed first-response deadline instead (see SilenceLevel).
//
// Only the assignee's own acts count, and the exclusions are the rule, not an
// oversight. Each of these is a way somebody ELSE could make a silent official
// look as though they had answered:
//
//   - Obstacle.AdjudicatedAt and Obstacle.ResolvedAt are the ADMIN's verdict on an
//     obstacle, not the official's work.
//   - An obstacle's advisory tallies and its community UnblockingPlans are the
//     PUBLIC judging it. That is pressure, never an answer (Concept §8).
//   - A monitor's observation note is the MONITOR writing. Counting it would let a
//     monitor close their own observation by asking about it.
//
// Declaring an obstacle (Obstacle.CreatedAt) does count: explaining why you cannot
// proceed is one of the four honest responses, and the opposite of silence.
func (c *Case) LastActivityAt() time.Time {
	var last time.Time
	bump := func(t time.Time) {
		if t.After(last) {
			last = t
		}
	}
	bumpPtr := func(t *time.Time) {
		if t != nil {
			bump(*t)
		}
	}

	bumpPtr(c.AcknowledgedAt)
	if c.Plan != nil {
		bump(c.Plan.CreatedAt)
		for _, task := range c.Plan.Tasks {
			bumpPtr(task.CompletedAt)
		}
	}
	for _, u := range c.Updates {
		bump(u.CreatedAt)
	}
	for _, e := range c.Evidence {
		bump(e.CreatedAt)
	}
	for _, o := range c.Obstacles {
		bump(o.CreatedAt)
	}
	return last
}
