-- Rename the typed-"blocker" vocabulary to "obstacle", unifying it with the
-- public-judgment half (obstacle_votes, /api/problems/{id}/obstacle) that already
-- speaks "obstacle". No behaviour change — a pure rename of table, columns, indexes
-- and the timeline note kind. The lifecycle status 'Blocked' and the audit action
-- literals ('blocked', 'blocker_confirmed', 'blocker_denied') are deliberately left
-- unchanged (CLAUDE.md A.6 / A.3.5): the status is the case's amber "waiting" state,
-- and the audit literals are what the public/oversight feeds filter on.

ALTER TABLE blockers RENAME TO obstacles;
ALTER INDEX idx_blockers_case RENAME TO idx_obstacles_case;

ALTER TABLE obstacle_votes RENAME COLUMN blocker_id TO obstacle_id;
ALTER INDEX idx_obstacle_votes_blocker RENAME TO idx_obstacle_votes_obstacle;

ALTER TABLE unblocking_plans RENAME COLUMN blocker_id TO obstacle_id;
ALTER INDEX idx_unblocking_plans_blocker RENAME TO idx_unblocking_plans_obstacle;

-- The timeline note kind: 'blocker' → 'obstacle'. Migrate existing rows first, then
-- swap the CHECK. (This is the progress_updates.kind value, distinct from the audit
-- action 'blocker_note', which is a wire literal left as-is.)
UPDATE progress_updates SET kind = 'obstacle' WHERE kind = 'blocker';
ALTER TABLE progress_updates DROP CONSTRAINT progress_updates_kind_check;
ALTER TABLE progress_updates ADD CONSTRAINT progress_updates_kind_check CHECK (kind IN ('progress', 'obstacle'));
