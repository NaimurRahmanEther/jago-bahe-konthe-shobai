-- Assignments: a validated problem handed to an official, with the monitor
-- (tier up the ladder), priority, and first-response deadline D. One assignment
-- per problem (the UNIQUE constraint backs the "already assigned" invariant).
-- override_reason is set and public only when the admin overrode the public's
-- pointed official.
CREATE TABLE assignments (
    id                  TEXT PRIMARY KEY,
    problem_id          TEXT        NOT NULL UNIQUE REFERENCES problems (id) ON DELETE CASCADE,
    official_id         TEXT        NOT NULL REFERENCES officials (id),
    monitor_official_id TEXT        REFERENCES officials (id),
    priority            TEXT        NOT NULL DEFAULT 'normal',
    deadline            TIMESTAMPTZ NOT NULL,
    override_reason     TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_assignments_official ON assignments (official_id);

-- Admin votes: the above-union assignment decision. eligible_voters is the voter
-- set fixed at open time, so quorum is measured against a stable denominator.
-- The tally (every ballot) is public.
CREATE TABLE admin_votes (
    id                   TEXT        PRIMARY KEY,
    problem_id           TEXT        NOT NULL REFERENCES problems (id) ON DELETE CASCADE,
    scope                TEXT        NOT NULL CHECK (scope IN ('upazila', 'seat')),
    proposed_official_id TEXT        NOT NULL REFERENCES officials (id),
    eligible_voters      INTEGER     NOT NULL,
    opened_at            TIMESTAMPTZ NOT NULL,
    closes_at            TIMESTAMPTZ NOT NULL,
    outcome              TEXT        NOT NULL DEFAULT 'pending'
        CHECK (outcome IN ('pending', 'approved', 'no_majority'))
);

CREATE INDEX idx_admin_votes_problem ON admin_votes (problem_id);

-- Ballots: one per admin per vote (the distinctness invariant). suggested_official_id
-- is set only when choice is 'suggest'.
CREATE TABLE admin_vote_ballots (
    vote_id              TEXT        NOT NULL REFERENCES admin_votes (id) ON DELETE CASCADE,
    voter_id             TEXT        NOT NULL REFERENCES accounts (id),
    choice               TEXT        NOT NULL CHECK (choice IN ('approve', 'reject', 'suggest')),
    suggested_official_id TEXT       REFERENCES officials (id),
    cast_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (vote_id, voter_id)
);
