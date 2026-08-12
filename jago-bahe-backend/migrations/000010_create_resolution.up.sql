-- Cases: an assigned problem worked in public by an official. One case per
-- problem (UNIQUE problem_id). monitor_official_id (copied from the assignment)
-- is the tier notified on blockers/escalation.
CREATE TABLE cases (
    id                  TEXT PRIMARY KEY,
    problem_id          TEXT        NOT NULL UNIQUE REFERENCES problems (id) ON DELETE CASCADE,
    official_id         TEXT        NOT NULL REFERENCES officials (id),
    monitor_official_id TEXT        REFERENCES officials (id),
    status              TEXT        NOT NULL DEFAULT 'Assigned',
    acknowledged_at     TIMESTAMPTZ,
    deadline            TIMESTAMPTZ NOT NULL,
    dispute_reason      TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_cases_official ON cases (official_id);

-- Plans: one public plan per case; must answer the community's top suggestion.
CREATE TABLE plans (
    id                  TEXT PRIMARY KEY,
    case_id             TEXT        NOT NULL UNIQUE REFERENCES cases (id) ON DELETE CASCADE,
    strategy            TEXT        NOT NULL,
    timeline_weeks      INTEGER     NOT NULL DEFAULT 0,
    obstacles           TEXT,
    suggestion_response TEXT        NOT NULL,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- Progress updates: the public weekly timeline (progress notes and blocker notes).
CREATE TABLE progress_updates (
    id         TEXT PRIMARY KEY,
    case_id    TEXT        NOT NULL REFERENCES cases (id) ON DELETE CASCADE,
    kind       TEXT        NOT NULL CHECK (kind IN ('progress', 'blocker')),
    text       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_updates_case ON progress_updates (case_id);

-- Evidence: before/after photo pairs; at least one is required before Done.
CREATE TABLE evidence (
    id               TEXT PRIMARY KEY,
    case_id          TEXT        NOT NULL REFERENCES cases (id) ON DELETE CASCADE,
    before_image_url TEXT        NOT NULL,
    after_image_url  TEXT        NOT NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_evidence_case ON evidence (case_id);

-- Blockers: typed obstacles; the category routes the forward to a higher
-- authority. Advisory tallies + upvote counts are derived on read from the
-- vote/plan tables below, never stored here.
CREATE TABLE blockers (
    id           TEXT PRIMARY KEY,
    case_id      TEXT        NOT NULL REFERENCES cases (id) ON DELETE CASCADE,
    category     TEXT        NOT NULL CHECK (category IN
                     ('budget', 'legal_authority', 'higher_tier', 'land_dispute', 'inter_department', 'technical')),
    what_blocks  TEXT        NOT NULL,
    who_unblocks TEXT        NOT NULL,
    proof_tried  TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at  TIMESTAMPTZ
);

CREATE INDEX idx_blockers_case ON blockers (case_id);

-- Obstacle votes: the advisory "is this obstacle real?" tally; one per resident
-- per blocker. Pressure only — never changes status or the scorecard.
CREATE TABLE obstacle_votes (
    id         TEXT PRIMARY KEY,
    blocker_id TEXT        NOT NULL REFERENCES blockers (id) ON DELETE CASCADE,
    voter_id   TEXT        NOT NULL REFERENCES accounts (id),
    choice     TEXT        NOT NULL CHECK (choice IN ('real', 'not_convinced')),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (blocker_id, voter_id)
);

CREATE INDEX idx_obstacle_votes_blocker ON obstacle_votes (blocker_id);

-- Unblocking plans: community-proposed ways to overcome a blocker (upvote_count
-- derived on read from unblocking_plan_votes).
CREATE TABLE unblocking_plans (
    id         TEXT PRIMARY KEY,
    blocker_id TEXT        NOT NULL REFERENCES blockers (id) ON DELETE CASCADE,
    author_id  TEXT        NOT NULL REFERENCES accounts (id),
    text       TEXT        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_unblocking_plans_blocker ON unblocking_plans (blocker_id);

-- Unblocking-plan upvotes: one per resident per plan (a toggle).
CREATE TABLE unblocking_plan_votes (
    id         TEXT PRIMARY KEY,
    plan_id    TEXT        NOT NULL REFERENCES unblocking_plans (id) ON DELETE CASCADE,
    voter_id   TEXT        NOT NULL REFERENCES accounts (id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (plan_id, voter_id)
);

CREATE INDEX idx_unblocking_plan_votes_plan ON unblocking_plan_votes (plan_id);
