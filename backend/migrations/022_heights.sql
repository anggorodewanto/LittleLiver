CREATE TABLE heights (
    id                  TEXT PRIMARY KEY,
    baby_id             TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by           TEXT REFERENCES users(id) NOT NULL,
    updated_by          TEXT REFERENCES users(id),
    timestamp           DATETIME NOT NULL,
    height_cm           REAL NOT NULL,
    measurement_source  TEXT,
    notes               TEXT,
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_heights_baby_timestamp ON heights (baby_id, timestamp);
