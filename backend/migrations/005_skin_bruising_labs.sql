-- Skin observations table (metric pattern)
CREATE TABLE skin_observations (
    id              TEXT PRIMARY KEY,
    baby_id         TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by       TEXT REFERENCES users(id) NOT NULL,
    updated_by      TEXT REFERENCES users(id),
    timestamp       DATETIME NOT NULL,
    jaundice_level  TEXT,
    scleral_icterus BOOLEAN DEFAULT FALSE,
    rashes          TEXT,
    bruising        TEXT,
    photo_keys      TEXT,
    notes           TEXT,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_skin_observations_baby_timestamp ON skin_observations (baby_id, timestamp);

-- Bruising table (metric pattern)
CREATE TABLE bruising (
    id            TEXT PRIMARY KEY,
    baby_id       TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by     TEXT REFERENCES users(id) NOT NULL,
    updated_by    TEXT REFERENCES users(id),
    timestamp     DATETIME NOT NULL,
    location      TEXT NOT NULL,
    size_estimate TEXT NOT NULL,
    size_cm       REAL,
    color         TEXT,
    photo_keys    TEXT,
    notes         TEXT,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_bruising_baby_timestamp ON bruising (baby_id, timestamp);

-- Lab results table (EAV-style, metric pattern)
CREATE TABLE lab_results (
    id           TEXT PRIMARY KEY,
    baby_id      TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by    TEXT REFERENCES users(id) NOT NULL,
    updated_by   TEXT REFERENCES users(id),
    timestamp    DATETIME NOT NULL,
    test_name    TEXT NOT NULL,
    value        TEXT NOT NULL,
    unit         TEXT,
    normal_range TEXT,
    notes        TEXT,
    created_at   DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at   DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_lab_results_baby_timestamp ON lab_results (baby_id, timestamp);
