-- Medication administration log
CREATE TABLE med_logs (
    id              TEXT PRIMARY KEY,
    medication_id   TEXT REFERENCES medications(id) ON DELETE CASCADE NOT NULL,
    baby_id         TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by       TEXT REFERENCES users(id) NOT NULL,
    updated_by      TEXT REFERENCES users(id),
    scheduled_time  DATETIME,
    given_at        DATETIME,
    skipped         BOOLEAN DEFAULT FALSE,
    skip_reason     TEXT,
    notes           TEXT,
    created_at      DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at      DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_med_logs_baby_id ON med_logs(baby_id);
CREATE INDEX idx_med_logs_medication_id ON med_logs(medication_id);
