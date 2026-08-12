-- Reverting B9 removes the screening vocabulary from the code, so any row still
-- holding a screening status would become unreadable to the reverted app. Publish
-- what was still pending rather than stranding it: pre-B9 has no concept of a
-- hidden problem, and losing a resident's report is worse than publishing one an
-- admin had not yet reviewed.
UPDATE problems SET status = 'Reported' WHERE status = 'PendingApproval';

-- Rejected has no pre-B9 equivalent. These rows were screened out as spam or
-- abuse, so they are mapped to Reported only if that is what the operator wants;
-- by default they are left as-is and will read as an unknown status. Decide
-- deliberately before running this down migration on data you care about.

DROP INDEX IF EXISTS idx_problems_status_created;

ALTER TABLE problems ALTER COLUMN status SET DEFAULT 'Reported';

ALTER TABLE problems
    DROP COLUMN IF EXISTS reviewed_at,
    DROP COLUMN IF EXISTS reviewed_by,
    DROP COLUMN IF EXISTS rejection_reason;
