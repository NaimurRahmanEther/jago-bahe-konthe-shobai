-- Recreates the admin-vote tables as they stood in 000008. It CANNOT restore any
-- vote or ballot that existed: the up migration dropped them, and nothing copied
-- the rows anywhere first. A rollback therefore returns the SHAPE of the vote and
-- none of its history — an above-union problem that had been voted on comes back
-- with no record that it was.
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

CREATE TABLE admin_vote_ballots (
    vote_id              TEXT        NOT NULL REFERENCES admin_votes (id) ON DELETE CASCADE,
    voter_id             TEXT        NOT NULL REFERENCES accounts (id),
    choice               TEXT        NOT NULL CHECK (choice IN ('approve', 'reject', 'suggest')),
    suggested_official_id TEXT       REFERENCES officials (id),
    cast_at              TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (vote_id, voter_id)
);

DROP INDEX IF EXISTS idx_forwarding_suggestions_problem;
DROP INDEX IF EXISTS idx_forwarding_suggestions_one_per_admin;
DROP TABLE IF EXISTS forwarding_suggestions;
