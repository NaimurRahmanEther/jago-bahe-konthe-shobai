-- Reverts the gate's default so a new problem publishes on report again.
--
-- THIS DOWN IS DELIBERATELY INCOMPLETE, AND CANNOT BE OTHERWISE. The up migration
-- did not move any existing rows, but while the gate was live new reports were
-- created as PendingApproval, and which rows those are is not recorded as history —
-- status is a single column. Rolling the default back does not publish the reports
-- that are already sitting in PendingApproval; clearing them is an application
-- decision (approve each, or the 000016-style bulk publish), not a schema one, so
-- this migration does not guess at it.

ALTER TABLE problems ALTER COLUMN status SET DEFAULT 'Reported';

COMMENT ON COLUMN problems.reviewed_by IS 'admin account id who rejected this problem; NULL otherwise. No FK, mirroring audit_entries.actor.';
