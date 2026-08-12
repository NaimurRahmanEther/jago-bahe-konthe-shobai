-- Officials directory: the public list of elected representatives.
CREATE TABLE officials (
    id      TEXT PRIMARY KEY,
    name    TEXT NOT NULL,
    phone   TEXT NOT NULL,
    tier    TEXT NOT NULL,
    area_id TEXT NOT NULL REFERENCES areas (id)
);

CREATE INDEX idx_officials_tier ON officials (tier);
CREATE INDEX idx_officials_area ON officials (area_id);

-- Accounts: the authentication identity for all three roles. Residents carry
-- nid/union_id/verified; officials and admins link to a directory official.
CREATE TABLE accounts (
    id            TEXT PRIMARY KEY,
    name          TEXT        NOT NULL,
    phone         TEXT        NOT NULL UNIQUE,
    password_hash TEXT        NOT NULL,
    role          TEXT        NOT NULL CHECK (role IN ('resident', 'official', 'admin')),
    nid           TEXT,
    union_id      TEXT REFERENCES areas (id),
    verified      BOOLEAN     NOT NULL DEFAULT false,
    official_id   TEXT REFERENCES officials (id),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now()
);
