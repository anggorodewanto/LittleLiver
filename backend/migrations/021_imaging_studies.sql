-- Imaging studies table — non-numeric lab artifacts (CT, Ultrasound, MRI, radiology PDFs).
-- Distinct from lab_results, which is EAV-style for numeric tests.
CREATE TABLE imaging_studies (
    id          TEXT PRIMARY KEY,
    baby_id     TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by   TEXT REFERENCES users(id) NOT NULL,
    updated_by  TEXT REFERENCES users(id),
    timestamp   DATETIME NOT NULL,
    study_date  TEXT NOT NULL,
    study_type  TEXT NOT NULL,
    notes       TEXT,
    photo_keys  TEXT NOT NULL,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_imaging_studies_baby_timestamp ON imaging_studies (baby_id, timestamp);
