-- B10: a plan snapshots the suggestion it answered.
--
-- "Top" is derived from live upvotes (B3 ranking: most upvotes, ties to oldest,
-- zero upvotes is never top), so it moves as the community keeps voting. Reading
-- it back later would pair an official's response with whatever question leads
-- today — showing them, on their own public record, answering something they were
-- never asked. The plan therefore carries the question with it.
--
-- Both columns are nullable: a problem whose suggestions have no upvotes has no
-- top suggestion, and the official's response then stands on its own.

ALTER TABLE plans
    ADD COLUMN answered_suggestion_id   TEXT REFERENCES suggestions (id) ON DELETE SET NULL,
    ADD COLUMN answered_suggestion_text TEXT;

COMMENT ON COLUMN plans.answered_suggestion_id IS 'the top suggestion as it stood when this plan was submitted; NULL when the problem had no upvoted suggestion';
COMMENT ON COLUMN plans.answered_suggestion_text IS 'the snapshotted text, kept independently of the suggestion row so the public record survives the suggestion being deleted';

-- Existing plans predate the snapshot and are left NULL rather than backfilled
-- from today's top: guessing which suggestion they answered would fabricate the
-- exact pairing this column exists to make truthful. They render as a response
-- with no question shown, which is honest about what was actually recorded.
