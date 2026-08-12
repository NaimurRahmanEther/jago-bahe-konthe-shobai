-- Append-only public audit trail. Application code only ever INSERTs here.
CREATE TABLE audit_entries (
    id          TEXT PRIMARY KEY,
    target_type TEXT        NOT NULL,
    target_id   TEXT        NOT NULL,
    actor       TEXT        NOT NULL,
    action      TEXT        NOT NULL,
    reason      TEXT,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_target ON audit_entries (target_type, target_id, created_at);
