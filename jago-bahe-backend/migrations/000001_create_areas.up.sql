CREATE TABLE areas (
    id        TEXT PRIMARY KEY,
    name      TEXT NOT NULL,
    level     TEXT NOT NULL CHECK (level IN ('seat', 'upazila', 'union', 'ward')),
    parent_id TEXT REFERENCES areas (id)
);

CREATE INDEX idx_areas_parent_id ON areas (parent_id);
