-- Reverse order, IF EXISTS throughout. Nothing here is recoverable: the
-- notifications ARE the table, so a rollback loses every unread message. That is
-- acceptable in a way the 000022 rollback is not — each notification is a derived
-- record of an event which is still on the audit trail, so the history survives
-- even though the messages do not.
DROP INDEX IF EXISTS idx_notifications_unread;
DROP INDEX IF EXISTS idx_notifications_recipient;
DROP TABLE IF EXISTS notifications;
