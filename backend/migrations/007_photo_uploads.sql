-- Photo upload staging (for orphan cleanup)
CREATE TABLE photo_uploads (
    id              TEXT PRIMARY KEY,
    baby_id         TEXT REFERENCES babies(id) ON DELETE SET NULL,
    r2_key          TEXT NOT NULL UNIQUE,
    thumbnail_key   TEXT,
    uploaded_at     DATETIME DEFAULT CURRENT_TIMESTAMP,
    linked_at       DATETIME
);
