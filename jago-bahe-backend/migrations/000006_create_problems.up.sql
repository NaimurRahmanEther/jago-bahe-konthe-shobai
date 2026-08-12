-- Problems: reported local issues, their pointing, and denormalized valid_count.
CREATE TABLE problems (
    id                  TEXT PRIMARY KEY,
    title               TEXT        NOT NULL,
    description         TEXT        NOT NULL,
    area_id             TEXT        NOT NULL REFERENCES areas (id),
    address             TEXT,
    lat                 DOUBLE PRECISION,
    lng                 DOUBLE PRECISION,
    reporter_id         TEXT        NOT NULL REFERENCES accounts (id),
    pointed_official_id TEXT        NOT NULL REFERENCES officials (id),
    proposed_solution   TEXT,
    image_url           TEXT,
    status              TEXT        NOT NULL DEFAULT 'Reported',
    valid_count         INTEGER     NOT NULL DEFAULT 0,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_problems_area ON problems (area_id);
CREATE INDEX idx_problems_status ON problems (status);
CREATE INDEX idx_problems_official ON problems (pointed_official_id);

-- Validation votes: one per resident per problem (the distinctness invariant).
CREATE TABLE validation_votes (
    id         TEXT PRIMARY KEY,
    problem_id TEXT        NOT NULL REFERENCES problems (id) ON DELETE CASCADE,
    voter_id   TEXT        NOT NULL REFERENCES accounts (id),
    vote       TEXT        NOT NULL CHECK (vote IN ('valid', 'invalid')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (problem_id, voter_id)
);

CREATE INDEX idx_votes_problem ON validation_votes (problem_id);
