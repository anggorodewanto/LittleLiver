-- Weights table (metric pattern)
CREATE TABLE weights (
    id                  TEXT PRIMARY KEY,
    baby_id             TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by           TEXT REFERENCES users(id) NOT NULL,
    updated_by          TEXT REFERENCES users(id),
    timestamp           DATETIME NOT NULL,
    weight_kg           REAL NOT NULL,
    measurement_source  TEXT,
    notes               TEXT,
    created_at          DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at          DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_weights_baby_timestamp ON weights (baby_id, timestamp);

-- Temperatures table (metric pattern)
CREATE TABLE temperatures (
    id          TEXT PRIMARY KEY,
    baby_id     TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by   TEXT REFERENCES users(id) NOT NULL,
    updated_by  TEXT REFERENCES users(id),
    timestamp   DATETIME NOT NULL,
    value       REAL NOT NULL,
    method      TEXT NOT NULL,
    notes       TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_temperatures_baby_timestamp ON temperatures (baby_id, timestamp);

-- Abdomen observations table (metric pattern)
CREATE TABLE abdomen_observations (
    id          TEXT PRIMARY KEY,
    baby_id     TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by   TEXT REFERENCES users(id) NOT NULL,
    updated_by  TEXT REFERENCES users(id),
    timestamp   DATETIME NOT NULL,
    firmness    TEXT NOT NULL,
    tenderness  BOOLEAN DEFAULT FALSE,
    girth_cm    REAL,
    photo_keys  TEXT,
    notes       TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_abdomen_observations_baby_timestamp ON abdomen_observations (baby_id, timestamp);
