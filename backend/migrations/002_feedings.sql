-- Feedings table (metric pattern)
CREATE TABLE feedings (
    id              TEXT PRIMARY KEY,
    baby_id         TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by       TEXT REFERENCES users(id) NOT NULL,
    updated_by      TEXT REFERENCES users(id),
    timestamp       DATETIME NOT NULL,
    feed_type       TEXT NOT NULL CHECK (feed_type IN ('breast_milk', 'formula', 'fortified_breast_milk', 'solid', 'other')),
    volume_ml       REAL,
    cal_density     REAL,
    calories        REAL,
    used_default_cal BOOLEAN DEFAULT FALSE,
    duration_min    INTEGER,
    notes           TEXT,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_feedings_baby_timestamp ON feedings (baby_id, timestamp);
