-- Dropping the snapshot loses which suggestion each plan answered; the reverted
-- code pairs responses with today's top instead.
ALTER TABLE plans
    DROP COLUMN IF EXISTS answered_suggestion_text,
    DROP COLUMN IF EXISTS answered_suggestion_id;
