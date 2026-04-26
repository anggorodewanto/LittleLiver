-- Per-medication stock containers (bottles, packets, pill packs, etc.).
-- A medication may have multiple containers; auto-decrement targets one per dose.
CREATE TABLE medication_containers (
    id                     TEXT PRIMARY KEY,
    medication_id          TEXT REFERENCES medications(id) ON DELETE CASCADE NOT NULL,
    baby_id                TEXT REFERENCES babies(id) ON DELETE CASCADE NOT NULL,
    kind                   TEXT NOT NULL,
    unit                   TEXT NOT NULL,
    quantity_initial       REAL NOT NULL,
    quantity_remaining     REAL NOT NULL,
    opened_at              DATETIME,
    max_days_after_opening INTEGER,
    expiration_date        TEXT,
    depleted               BOOLEAN NOT NULL DEFAULT FALSE,
    notes                  TEXT,
    created_by             TEXT REFERENCES users(id) NOT NULL,
    updated_by             TEXT REFERENCES users(id),
    created_at             DATETIME DEFAULT CURRENT_TIMESTAMP,
    updated_at             DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_medication_containers_medication_id ON medication_containers(medication_id);
CREATE INDEX idx_medication_containers_baby_id ON medication_containers(baby_id);

-- Audit trail of manual stock adjustments (non-dose changes).
CREATE TABLE medication_stock_adjustments (
    id            TEXT PRIMARY KEY,
    container_id  TEXT REFERENCES medication_containers(id) ON DELETE CASCADE NOT NULL,
    delta         REAL NOT NULL,
    reason        TEXT,
    adjusted_by   TEXT REFERENCES users(id) NOT NULL,
    adjusted_at   DATETIME NOT NULL,
    created_at    DATETIME DEFAULT CURRENT_TIMESTAMP
);

CREATE INDEX idx_medication_stock_adjustments_container_id ON medication_stock_adjustments(container_id);
