-- Clears every reported problem so a manual test round starts from an empty
-- board. Accounts, geography, the officials directory and the admin pool are
-- left alone -- every seeded and hand-registered login keeps working.
--
-- problems cascades to validation votes, suggestions, assignments, admin votes,
-- cases, plans, updates, evidence and blockers (000006-000010), so one DELETE
-- takes the whole tree.
--
-- audit_entries has NO foreign key -- target_id is polymorphic TEXT -- so
-- nothing removes those for us; they are deleted explicitly, scoped to
-- target_type = 'problem'. The account-scoped entries (registered, verified,
-- claim_filed) survive, which is correct: those accounts still exist.
--
-- Order is problems first, audit second, matching the reporter's hard delete
-- (CLAUDE.md A.3.3 constraint 4). Inside this transaction the two are atomic
-- either way, but audit-first would destroy the trail of problems that still
-- exist if the second statement failed -- do not swap them.

BEGIN;

DELETE FROM problems;
DELETE FROM audit_entries WHERE target_type = 'problem';

COMMIT;
