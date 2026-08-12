-- Restores B9's default so a new problem starts behind the pre-publication gate.
--
-- THIS DOWN IS DELIBERATELY INCOMPLETE, AND CANNOT BE OTHERWISE. The up migration
-- published every row that was still PendingApproval, and which rows those were is
-- not recorded anywhere — status is a single column with no history. Rolling back
-- restores the default for future rows only; the published ones stay published.
--
-- That asymmetry is the honest one. Guessing (say, re-hiding every Reported row
-- with no votes) would take reports the public has already been able to read and
-- retroactively hide them, which is a silent takedown of the existing record.

ALTER TABLE problems ALTER COLUMN status SET DEFAULT 'PendingApproval';

COMMENT ON COLUMN problems.reviewed_by IS 'admin account id, or ''system'' for auto-approval after window A; no FK so the system actor is representable (mirrors audit_entries.actor)';
