-- Stools table (metric pattern)
CREATE TABLE stools (
    id              TEXT PRIMARY KEY,
    baby_id         TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by       TEXT REFERENCES users(id) NOT NULL,
    updated_by      TEXT REFERENCES users(id),
    timestamp       DATETIME NOT NULL,
    color_rating    INTEGER NOT NULL CHECK (color_rating BETWEEN 1 AND 7),
    color_label     TEXT,
    consistency     TEXT,
    volume_estimate TEXT,
    photo_keys      TEXT,
    notes           TEXT,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_stools_baby_timestamp ON stools (baby_id, timestamp);

-- Urine table (metric pattern)
CREATE TABLE urine (
    id          TEXT PRIMARY KEY,
    baby_id     TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by   TEXT REFERENCES users(id) NOT NULL,
    updated_by  TEXT REFERENCES users(id),
    timestamp   DATETIME NOT NULL,
    color       TEXT,
    notes       TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_urine_baby_timestamp ON urine (baby_id, timestamp);
