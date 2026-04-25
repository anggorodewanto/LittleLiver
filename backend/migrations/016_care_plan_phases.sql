-- Care plan phases: ordered, dated steps within a care plan.
-- "Current" phase is the one whose start_date is the most recent <= today.
-- start_date is naive 'YYYY-MM-DD' resolved in the parent plan's timezone.
CREATE TABLE care_plan_phases (
    id            TEXT PRIMARY KEY,
    care_plan_id  TEXT REFERENCES care_plans(id) ON DELETE CASCADE NOT NULL,
    seq           INTEGER NOT NULL,
    label         TEXT NOT NULL,
    start_date    TEXT NOT NULL,
    ends_on       TEXT,
    notes         TEXT,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    UNIQUE (care_plan_id, seq)
);

CREATE INDEX idx_care_plan_phases_plan ON care_plan_phases(care_plan_id);
CREATE INDEX idx_care_plan_phases_start ON care_plan_phases(start_date);

-- Audit ledger giving the stateless scheduler exactly-once notification semantics.
-- Scheduler does INSERT OR IGNORE keyed on (phase_id, kind); only sends a push
-- when a row was newly inserted. Cascades with the phase.
CREATE TABLE care_plan_phase_notifications (
    phase_id   TEXT NOT NULL REFERENCES care_plan_phases(id) ON DELETE CASCADE,
    kind       TEXT NOT NULL,
    sent_at    DATETIME DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (phase_id, kind)
);
