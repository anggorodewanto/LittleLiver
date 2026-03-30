-- Add volume_ml to urine and stools for fluid balance tracking
ALTER TABLE urine ADD COLUMN volume_ml REAL;
ALTER TABLE stools ADD COLUMN volume_ml REAL;

-- Unified fluid I&O ledger
CREATE TABLE fluid_log (
    id          TEXT PRIMARY KEY,
    baby_id     TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by   TEXT REFERENCES users(id) NOT NULL,
    updated_by  TEXT REFERENCES users(id),
    timestamp   DATETIME NOT NULL,
    direction   TEXT NOT NULL CHECK (direction IN ('intake', 'output')),
    method      TEXT NOT NULL,
    volume_ml   REAL,
    source_type TEXT CHECK (source_type IN ('feeding', 'urine', 'stool')),
    source_id   TEXT,
    notes       TEXT,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_fluid_log_baby_timestamp ON fluid_log (baby_id, timestamp);
CREATE UNIQUE INDEX idx_fluid_log_source ON fluid_log (source_type, source_id) WHERE source_type IS NOT NULL;
