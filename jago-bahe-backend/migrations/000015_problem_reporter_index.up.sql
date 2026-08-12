-- B12: the reporter's own view (GET /api/me/problems).
--
-- problems.reporter_id already exists (000006, NOT NULL, FK to accounts); this
-- adds only the index that read needs. Composite with created_at DESC because the
-- listing's ORDER BY is created_at DESC, so one index serves both the scope and
-- the sort — a plain reporter_id index would still leave a sort node.
CREATE INDEX idx_problems_reporter_created ON problems (reporter_id, created_at DESC);
