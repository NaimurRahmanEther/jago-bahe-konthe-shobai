-- Reverses B9's pre-publication gate (000012). A problem is public and votable
-- the moment it is reported: the community (V) is the first filter, and you
-- cannot vote on what you cannot see. Screening becomes post-publication — an
-- admin can still take a problem down as spam/abusive/duplicate/wrong_area, but
-- the takedown is public, narrow (pre-assignment only), and blocks nobody's read.

ALTER TABLE problems ALTER COLUMN status SET DEFAULT 'Reported';

-- Publish everything still waiting behind the gate. These reports were filed in
-- good faith and have been readable by nobody but their reporter; leaving them
-- pending after the gate is gone would strand them in a state no code can now
-- clear, which is exactly the invisible veto the auto-approve worker existed to
-- prevent. Reviewed_by is left NULL: no admin decided this, and stamping 'system'
-- would claim a screening decision that never happened.
UPDATE problems SET status = 'Reported' WHERE status = 'PendingApproval';

-- rejection_reason / reviewed_by / reviewed_at all survive: post-publication
-- moderation still records who took a problem down and on what ground.
COMMENT ON COLUMN problems.reviewed_by IS 'admin account id who rejected this problem; NULL otherwise. No FK, mirroring audit_entries.actor.';

-- idx_problems_status_created (000012) survives too: the moderation queue and the
-- feed both filter on status and order by created_at.
