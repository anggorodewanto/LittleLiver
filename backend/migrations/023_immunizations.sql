-- Immunization records: per-baby administered vaccine doses.
-- The reference schedule (vaccine list, recommended ages, mandatory flag) lives
-- in code (internal/immunization). This table stores only what was given; the
-- schedule/status view is computed against the baby's date of birth.
CREATE TABLE immunizations (
    id                TEXT PRIMARY KEY,
    baby_id           TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by         TEXT REFERENCES users(id) NOT NULL,
    updated_by        TEXT REFERENCES users(id),
    vaccine_code      TEXT NOT NULL DEFAULT '',
    vaccine_name      TEXT NOT NULL,
    dose_number       INTEGER,
    administered_date TEXT NOT NULL,
    provider          TEXT,
    lot_number        TEXT,
    notes             TEXT,
    created_at        DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at        DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_immunizations_baby_id ON immunizations(baby_id);
