ALTER TABLE cases DROP COLUMN IF EXISTS last_escalated_at, DROP COLUMN IF EXISTS escalation_level;
ALTER TABLE blockers DROP COLUMN IF EXISTS adjudicated_at, DROP COLUMN IF EXISTS adjudication;
DROP TABLE IF EXISTS confirmations;
