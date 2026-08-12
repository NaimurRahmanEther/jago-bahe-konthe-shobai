-- B20: above-union forwarding becomes the super admin's decision, advised by the
-- union admins. This replaces the binding admin vote (quorum Q, majority within
-- window W, no-majority fallback to the Union Chairman) with advice plus one
-- accountable decider. See CLAUDE.md A.3.8.

-- One admin's advice on where an above-union report should go. It settles
-- nothing: there is no quorum, no window, and no outcome column, because the
-- super admin may forward at any count including none. What it buys is that the
-- decision is made in the open — the tally is public, so forwarding against the
-- seat's admins is visible.
CREATE TABLE forwarding_suggestions (
    id                    TEXT        PRIMARY KEY,
    problem_id            TEXT        NOT NULL REFERENCES problems (id) ON DELETE CASCADE,
    admin_account_id      TEXT        NOT NULL REFERENCES accounts (id),
    suggested_official_id TEXT        NOT NULL REFERENCES officials (id),
    reason                TEXT        NOT NULL DEFAULT '',
    created_at            TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- One row per admin per problem. An UPSERT on this index is how an admin CHANGES
-- their advice, which the ballot it replaced deliberately refused — a ballot
-- settled something, so changing it would have rewritten a result, while advice
-- that settles nothing should be revisable.
CREATE UNIQUE INDEX idx_forwarding_suggestions_one_per_admin
    ON forwarding_suggestions (problem_id, admin_account_id);

CREATE INDEX idx_forwarding_suggestions_problem ON forwarding_suggestions (problem_id);

-- The vote is gone, so its tables go with it. A table nothing reads or writes is
-- how a later session concludes the vote still exists and re-wires it.
DROP TABLE IF EXISTS admin_vote_ballots;
DROP TABLE IF EXISTS admin_votes;
