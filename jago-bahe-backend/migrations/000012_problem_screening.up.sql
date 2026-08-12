-- B9: the pre-publication screening gate. A new problem now starts
-- PendingApproval and is not publicly readable until its union's admin approves
-- it — or until the screening window A expires and the worker publishes it.

ALTER TABLE problems
    ADD COLUMN rejection_reason TEXT,
    ADD COLUMN reviewed_by      TEXT,
    ADD COLUMN reviewed_at      TIMESTAMPTZ;

-- reviewed_by holds an admin's account id, or 'system' when the window A expired
-- and the problem published itself. It deliberately carries no FK to accounts —
-- mirroring audit_entries.actor, which is FK-free for exactly this reason: the
-- system actor is not an account.
COMMENT ON COLUMN problems.reviewed_by IS 'admin account id, or ''system'' for auto-approval after window A; no FK so the system actor is representable (mirrors audit_entries.actor)';
COMMENT ON COLUMN problems.rejection_reason IS 'one of spam|abusive|duplicate|wrong_area; set only when status = Rejected, and shown publicly';

ALTER TABLE problems ALTER COLUMN status SET DEFAULT 'PendingApproval';

-- Existing Reported rows are intentionally left alone. They are already publicly
-- readable, and retroactively hiding reports residents have already filed —
-- pending review by an admin who never saw them — would be a silent takedown of
-- the existing record.

-- The worker scans for pending problems older than A on every tick.
CREATE INDEX idx_problems_status_created ON problems (status, created_at);
