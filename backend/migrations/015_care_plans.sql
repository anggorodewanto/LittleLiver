-- Care plans: a named, ordered sequence of phases (e.g. rotating monthly antibiotics).
-- Pure schedule/display surface — no logs, no per-dose confirmations.
CREATE TABLE care_plans (
    id          TEXT PRIMARY KEY,
    baby_id     TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    logged_by   TEXT REFERENCES users(id) NOT NULL,
    updated_by  TEXT REFERENCES users(id),
    name        TEXT NOT NULL,
    notes       TEXT,
    timezone    TEXT NOT NULL,
    active      BOOLEAN DEFAULT TRUE,
    created_at  DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at  DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_care_plans_baby_id ON care_plans(baby_id);
