-- Medications table (definitions / schedules)
CREATE TABLE medications (
    id          TEXT PRIMARY KEY,
    baby_id     TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by   TEXT REFERENCES users(id) NOT NULL,
    updated_by  TEXT REFERENCES users(id),
    name        TEXT NOT NULL,
    dose        TEXT NOT NULL,
    frequency   TEXT NOT NULL,
    schedule    TEXT,
    timezone    TEXT,
    active      BOOLEAN DEFAULT TRUE,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_medications_baby_id ON medications(baby_id);
