UPDATE progress_updates SET kind = 'blocker' WHERE kind = 'obstacle';
ALTER TABLE progress_updates DROP CONSTRAINT progress_updates_kind_check;
ALTER TABLE progress_updates ADD CONSTRAINT progress_updates_kind_check CHECK (kind IN ('progress', 'blocker'));

ALTER INDEX idx_unblocking_plans_obstacle RENAME TO idx_unblocking_plans_blocker;
ALTER TABLE unblocking_plans RENAME COLUMN obstacle_id TO blocker_id;

ALTER INDEX idx_obstacle_votes_obstacle RENAME TO idx_obstacle_votes_blocker;
ALTER TABLE obstacle_votes RENAME COLUMN obstacle_id TO blocker_id;

ALTER INDEX idx_obstacles_case RENAME TO idx_blockers_case;
ALTER TABLE obstacles RENAME TO blockers;
