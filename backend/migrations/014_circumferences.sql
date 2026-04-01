CREATE TABLE head_circumferences (
    id                  TEXT PRIMARY KEY,
    baby_id             TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by           TEXT REFERENCES users(id) NOT NULL,
    updated_by          TEXT REFERENCES users(id),
    timestamp           DATETIME NOT NULL,
    circumference_cm    REAL NOT NULL,
    measurement_source  TEXT,
    notes               TEXT,
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_head_circumferences_baby_timestamp ON head_circumferences (baby_id, timestamp);

CREATE TABLE upper_arm_circumferences (
    id                  TEXT PRIMARY KEY,
    baby_id             TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by           TEXT REFERENCES users(id) NOT NULL,
    updated_by          TEXT REFERENCES users(id),
    timestamp           DATETIME NOT NULL,
    circumference_cm    REAL NOT NULL,
    measurement_source  TEXT,
    notes               TEXT,
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX idx_upper_arm_circumferences_baby_timestamp ON upper_arm_circumferences (baby_id, timestamp);
