-- B6: confirmation, blocker adjudication, and deterministic escalation.

-- Confirmations: the reporting resident's verdict on a Done case. Appended (a
-- problem can be confirmed again after a reopen), so no uniqueness constraint.
CREATE TABLE confirmations (
    id          TEXT PRIMARY KEY,
    problem_id  TEXT        NOT NULL REFERENCES problems (id) ON DELETE CASCADE,
    resident_id TEXT        NOT NULL REFERENCES accounts (id),
    outcome     TEXT        NOT NULL CHECK (outcome IN ('solved', 'not_solved')),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_confirmations_problem ON confirmations (problem_id);

-- Blocker adjudication: the named authority's verdict (the scorecard-setting
-- decision, distinct from the advisory public vote).
ALTER TABLE blockers
    ADD COLUMN adjudication   TEXT        NOT NULL DEFAULT 'pending'
        CHECK (adjudication IN ('pending', 'confirmed', 'denied')),
    ADD COLUMN adjudicated_at TIMESTAMPTZ;

-- Escalation: the visibility rung a case has climbed from measured silence. The
-- work is never moved off the official; only its visibility level rises.
ALTER TABLE cases
    ADD COLUMN escalation_level  INTEGER     NOT NULL DEFAULT 0,
    ADD COLUMN last_escalated_at TIMESTAMPTZ;
