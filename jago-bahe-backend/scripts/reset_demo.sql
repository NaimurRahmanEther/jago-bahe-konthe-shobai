-- Clean slate before re-seeding the demo board (report/scripts/seed_demo.mjs).
--
-- Removes EVERY problem and every account that was created at runtime, leaving
-- only what the migrations put there: the geography, the officials directory,
-- the nine union admins, the super admin, and the two logins information.md
-- documents. The seeder then rebuilds the whole board through the real HTTP API.
--
-- This REPLACES the older scripts/reset_testdata.sql, which tried to preserve one
-- hand-built report by name. That keep-list went stale the moment the demo data
-- changed, and a stale keep-list here does not fail loudly -- it silently deletes
-- the board you meant to keep. A full wipe plus a re-runnable seeder has no such
-- failure mode: the seeder IS the definition of what should exist.
--
-- ORDER IS LOAD-BEARING. Foreign keys into accounts are NO ACTION, not CASCADE
-- (problems.reporter_id, validation_votes.voter_id, suggestions.author_id,
-- confirmations.resident_id, forwarding_suggestions.admin_account_id and four
-- more), so an account cannot be deleted until the rows referencing it are gone.
-- Problems must therefore go FIRST -- deleting them is what frees the accounts.
-- official_claims is the one true CASCADE off accounts.
--
-- Problems cascade to validation votes, suggestions, assignments, cases, plans,
-- plan tasks, progress updates, evidence, obstacles, obstacle votes, unblocking
-- plans, case observations, observation notes, confirmations, forwarding
-- suggestions and notifications (000006-000023), so one DELETE takes the tree.
--
-- audit_entries has NO foreign key -- target_id is polymorphic TEXT (CLAUDE.md
-- A.3.3 constraint 3) -- so nothing removes those for us; they are deleted by
-- hand below.
--
-- Problems before audit, matching the reporter's hard delete (A.3.3 constraint
-- 4). Inside this transaction the two are atomic either way, but audit-first
-- would destroy the trail of problems that still exist if the second statement
-- failed -- do not swap them.
--
-- Run it with the existing runner (no psql needed on PATH):
--     go run ./cmd/reset -file scripts/reset_demo.sql confirm
--   or
--     make reset-demo

BEGIN;

-- 0. Who survives. Stated positively -- keep these, drop the rest -- so a test
--    account registered tomorrow is swept without anyone having to remember to
--    add it to a delete list.
--
--    DO NOT switch this to an id prefix. `acct-seed-%` looks like a seed marker
--    and is not one: the nine union admins are `acct-admin-*`, and every account
--    created through the UI has a hashed id, so a prefix rule would delete the
--    entire admin pool and the geography's reason for existing.
CREATE TEMP TABLE keep_accounts ON COMMIT DROP AS
SELECT id FROM accounts
 WHERE role IN ('admin', 'super_admin')   -- the nine union admins + the super admin
    OR phone IN (
         '01810000001',                   -- রহিমা খাতুন   — seeded demo resident (information.md)
         '01710000001'                    -- মো. করিম উদ্দিন — seeded official login, bound to off-mayor
       );

CREATE TEMP TABLE dropped_accounts ON COMMIT DROP AS
SELECT id FROM accounts WHERE id NOT IN (SELECT id FROM keep_accounts);

-- 1. Every problem, and everything hanging off it.
DELETE FROM problems;

-- 2. Their audit entries. Safe to scope by target_type alone here, unlike the
--    scoped reset this replaces, because step 1 removed every problem.
DELETE FROM audit_entries WHERE target_type = 'problem';

-- 3. The runtime accounts. Reachable only now: step 1 removed the reports, votes,
--    suggestions and advice rows that were holding them.
DELETE FROM accounts WHERE id IN (SELECT id FROM dropped_accounts);

-- 4. Their audit entries -- BOTH shapes, because the two account-scoped kinds do
--    not agree on what target_id means:
--      target_type='resident'  -> target_id IS the account id ('registered', and
--                                 'verified' whose actor is the admin who did it)
--      target_type='official'  -> target_id is the DIRECTORY OFFICE id
--                                 ('off-chair-aranagar'), and the account is the
--                                 actor. Scoping that one by target_id would
--                                 delete nothing and leave every claim_filed by a
--                                 deleted account dangling.
--    Matching on actor as well is what removes those.
DELETE FROM audit_entries
 WHERE actor IN (SELECT id FROM dropped_accounts)
    OR (target_type = 'resident' AND target_id IN (SELECT id FROM dropped_accounts));

-- 5. Sweep account-scoped entries left over from EARLIER cleanups -- accounts
--    removed before this script existed, whose audit rows nothing deleted because
--    audit_entries has no FK. Scoped to target_type='resident', where target_id
--    is reliably an account id.
--
--    Do NOT generalise this to `actor NOT IN (SELECT id FROM accounts)`. The actor
--    of an official's action is the DIRECTORY OFFICE id ('off-chair-aranagar'),
--    not an account id, so that predicate reads every acknowledged/plan_submitted
--    entry as an orphan.
DELETE FROM audit_entries
 WHERE target_type = 'resident'
   AND target_id NOT IN (SELECT id FROM accounts);

COMMIT;
