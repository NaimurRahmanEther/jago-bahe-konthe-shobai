-- B21: in-app notifications — the LOOP-CLOSING SET, and nothing else.
--
-- Eight events, chosen by one test: is there any surface on which this party
-- could learn this today? The reporter had none for approval, rejection,
-- assignment or the request to confirm — a report goes into a feed of everyone
-- else's and /me shows a status, not an event. The assigned official had none for
-- being given a case, for a resident reopening it, or for the verdict on their own
-- obstacle. The monitor had none for an obstacle declared on a case they watch,
-- which is the one notification the design documents promise outright (Concept §8,
-- Scaffold §2, Backend Plan B5: a blocker "notifies the named higher authority").
--
-- There are deliberately NO admin notifications — they already have three queues
-- with live counts on /admin — and nothing about validation, plans, progress,
-- verification or claims. See CLAUDE.md A.3.9.
CREATE TABLE notifications (
    id             TEXT        PRIMARY KEY,

    -- Two id spaces, and the row says WHICH. The platform has an account id for
    -- residents, admins and the super admin, and a DIRECTORY OFFICE id for
    -- officials (an official's actions are recorded against `off-mayor`, never
    -- their account). audit_entries.actor carries both in one column and tells
    -- them apart by prefix convention alone; cmd/api/adapters.go's
    -- auditActorAdapter says so and accepts that a hypothetical id present in both
    -- would resolve to the account.
    --
    -- That is fine for resolving a display NAME, where a collision costs a wrong
    -- label. It is not fine here, where a collision delivers a private message to
    -- the wrong person — the authorship graph is a safety matter (A.3.2 rule 1).
    -- So the read query matches the PAIR, not the id.
    recipient_kind TEXT        NOT NULL CHECK (recipient_kind IN ('account', 'official')),

    -- The non-empty CHECK is load-bearing, not tidiness. Case.monitor_official_id
    -- is '' when the assignee is at the top of the ladder — MonitorFor returns ""
    -- for the MP, who has nobody above them — and the read query matches the
    -- caller's own account id AND official id, the latter being '' for every
    -- resident, admin and super admin. ONE blank row would therefore be delivered
    -- to every non-official in the seat. Guarded three ways: the domain's
    -- ForResident/ForOfficial constructors, Append's ErrNoRecipient, and here.
    recipient_id   TEXT        NOT NULL CHECK (recipient_id <> ''),

    -- No FK on recipient_id: it is polymorphic across accounts.id and officials.id,
    -- exactly as audit_entries.actor is. That is precisely why the CHECK above
    -- matters — nothing else constrains this column.

    -- The closed set, in the DB as well as in Type.Valid(). problems.status has no
    -- CHECK and its comment says Valid() is the only guard; that is right for a
    -- vocabulary which has been reversed three times. This one is closed by design,
    -- and a typo'd literal would otherwise produce a row the UI renders as a raw
    -- i18n key. Needing a migration to add a ninth type is the point, not the cost:
    -- a type written by nothing is invisible, and a page nobody ever sees is
    -- indistinguishable from a seat where the event never happens — the
    -- `adjudicated` failure (A.3.5 rule 2), which is this table's exact failure mode.
    type           TEXT        NOT NULL CHECK (type IN (
                                   'problem_approved',
                                   'problem_rejected',
                                   'problem_assigned',
                                   'confirmation_requested',
                                   'case_assigned',
                                   'case_reopened',
                                   'obstacle_declared',
                                   'obstacle_adjudicated')),

    -- ON DELETE CASCADE is what satisfies A.3.3: a hard-deleted report must leave
    -- no trace anywhere, and this gives it for free — strictly better than audit's
    -- manual DeleteByTarget, which exists only because audit_entries has no FK to
    -- problems. Every one of the eight events is ABOUT a problem, which is what
    -- makes NOT NULL honest. A future notification that is not about a problem
    -- (a claim approval, say) must solve its own erasure before it may be added.
    problem_id     TEXT        NOT NULL REFERENCES problems (id) ON DELETE CASCADE,

    -- SNAPSHOTTED, not resolved at read time — the argument
    -- plans.answered_suggestion_text already makes: a notification records what was
    -- true when it was SENT, and a reporter may edit their own title while the
    -- report is unvalidated. It also means the read path never touches `problems`,
    -- so it can never become an oracle over problem ids, and the context needs no
    -- ports at all (unlike audit, which needs Actors and Problems).
    --
    -- The cost, stated plainly: this bypasses TitlesByIDs's publicly-visible-only
    -- filter, which is a safety property rather than an optimisation. So the
    -- invariant "a notification may only ever be written to a party already
    -- entitled to see the thing it is about" is enforced at each of the eight write
    -- sites instead, and is checked event by event in Scaffold §2 and A.3.9.
    problem_title  TEXT        NOT NULL DEFAULT '',

    -- Type-specific payload, never prose: the rejection ground enum, the assigned
    -- official's name, the deadline, 'confirmed'/'denied', or the obstacle's
    -- free-text whoUnblocks. It is only ever read inside a switch on `type`.
    detail         TEXT        NOT NULL DEFAULT '',

    -- Nullable timestamp, not a boolean: "when" is a fact and "whether" derives
    -- from it. Setting this is the ONE state change in the platform that appends no
    -- audit entry — the log is public, so an entry per mark-read would publish who
    -- is reading what (A.3.2 rule 4). Commented again in mark_read.go and in the
    -- repository port.
    read_at        TIMESTAMPTZ,

    created_at     TIMESTAMPTZ NOT NULL DEFAULT now()
);

-- The list query: newest first for one recipient pair. Composite because the
-- ORDER BY is created_at DESC, the same shape as idx_problems_reporter_created.
CREATE INDEX idx_notifications_recipient
    ON notifications (recipient_kind, recipient_id, created_at DESC);

-- The bell's count. Partial, so it holds only the rows the count is over — the
-- badge is fetched for every signed-in caller on every page and is the hottest
-- query this phase adds.
CREATE INDEX idx_notifications_unread
    ON notifications (recipient_kind, recipient_id) WHERE read_at IS NULL;

-- NO partial unique index anywhere, and that is a DECISION, not an omission. The
-- obvious model is idx_case_observations_open, and the reasoning there points the
-- other way here: that index is partial precisely so a case which went quiet, was
-- answered, and went quiet again opens rung 1 as a NEW row. The same is true of
-- these events. A second "please confirm" after a resident rejected the first and
-- the official redid the work is a NEW claim of completion, not a repeat; a second
-- obstacle is a second declaration.
--
-- The events that must NOT repeat are already single-shot behind a guarded
-- transition — SetScreening's compare-and-set on the source status,
-- ErrAlreadyAssigned, ErrAlreadyAdjudicated — and each returns before its notify
-- line is reached. A unique index here would be a second, weaker copy of a guard
-- the domain already holds.
