-- B16/F19: re-instate the pre-publication approval gate (reverses 000016). A new
-- problem starts PendingApproval — invisible to everyone but its reporter and its
-- union's admin — and is published (status Reported) only when that admin approves
-- it, or moved to Rejected when they reject it on a fixed ground. There is
-- deliberately NO auto-publish: an admin who never acts leaves the report hidden.
-- That is the accountable cost of the gate, chosen over publishing spam and
-- defamation before any human has seen them.

ALTER TABLE problems ALTER COLUMN status SET DEFAULT 'PendingApproval';

-- Existing rows are left as they are. Reports already Reported (or further along)
-- are publicly readable and stay so — retroactively hiding a report residents can
-- already see would be a silent takedown of the existing record, the same
-- principle 000012 and 000016 both kept.

-- reviewed_by now holds only a real admin's account id (approve or reject); there
-- is no 'system' actor, since the hard gate has no auto-approve worker. It still
-- carries no FK to accounts, mirroring audit_entries.actor.
COMMENT ON COLUMN problems.reviewed_by IS 'admin account id who approved or rejected this problem; NULL otherwise. No FK, mirroring audit_entries.actor.';

-- idx_problems_status_created (000012) is unchanged: the screening queue filters
-- on status (PendingApproval) and orders by created_at, exactly as it needs.
