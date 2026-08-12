-- The observation ladder (B19): the read side of escalation. When an official goes
-- silent past the first-response deadline D, a copy of the case surfaces to the
-- official above them, who watches it until it is answered (Concept §7).
--
-- These rows record VISIBILITY ONLY. The case is never moved off its official and
-- the observer gets no power over it — a declared obstacle moves responsibility up,
-- silence moves only visibility. There is no "taken over" column and none may be
-- added without reversing that rule.
CREATE TABLE case_observations (
    id                   TEXT        PRIMARY KEY,
    case_id              TEXT        NOT NULL REFERENCES cases (id) ON DELETE CASCADE,
    observer_official_id TEXT        NOT NULL REFERENCES officials (id),
    level                INTEGER     NOT NULL,
    opened_at            TIMESTAMPTZ NOT NULL,
    resolved_at          TIMESTAMPTZ,
    created_at           TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The worker's idempotency guard: at most ONE OPEN observation per rung, so a scan
-- running every minute inserts a rung once. It is deliberately partial rather than
-- a plain unique: a resolved row has resolved_at set and no longer blocks, which is
-- what lets a case that went quiet, was answered, and went quiet again open rung 1
-- a second time as a NEW row — keeping both silences on the record. Inserts must
-- therefore say ON CONFLICT (case_id, level) WHERE resolved_at IS NULL DO NOTHING.
CREATE UNIQUE INDEX idx_case_observations_open
    ON case_observations (case_id, level) WHERE resolved_at IS NULL;

CREATE INDEX idx_case_observations_observer
    ON case_observations (observer_official_id, resolved_at);

-- A monitor's notes on what they did about the silence. APPEND-ONLY and never a
-- single overwritable column on case_observations: this platform's spine is a
-- public record, and each note also lands on the problem's audit trail.
CREATE TABLE observation_notes (
    id             TEXT        PRIMARY KEY,
    observation_id TEXT        NOT NULL REFERENCES case_observations (id) ON DELETE CASCADE,
    text           TEXT        NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_observation_notes_observation ON observation_notes (observation_id);

-- cases.monitor_official_id has been written since B4 and never queried; the
-- monitor's list scans it, so it needs an index of its own.
CREATE INDEX idx_cases_monitor ON cases (monitor_official_id);
