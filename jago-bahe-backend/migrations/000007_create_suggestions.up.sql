-- Suggestions: community-proposed fixes for a problem. upvote_count and the
-- "top" flag are derived on read (from suggestion_votes + the ranking service),
-- so only the votes are stored here.
CREATE TABLE suggestions (
    id         TEXT PRIMARY KEY,
    problem_id TEXT        NOT NULL REFERENCES problems (id) ON DELETE CASCADE,
    author_id  TEXT        NOT NULL REFERENCES accounts (id),
    text       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_suggestions_problem ON suggestions (problem_id);

-- Suggestion upvotes: one per resident per suggestion (the distinctness
-- invariant); an upvote is a toggle (its presence = support).
CREATE TABLE suggestion_votes (
    id            TEXT PRIMARY KEY,
    suggestion_id TEXT        NOT NULL REFERENCES suggestions (id) ON DELETE CASCADE,
    voter_id      TEXT        NOT NULL REFERENCES accounts (id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (suggestion_id, voter_id)
);

CREATE INDEX idx_suggestion_votes_suggestion ON suggestion_votes (suggestion_id);
