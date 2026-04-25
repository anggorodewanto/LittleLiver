-- Track which container a dose came from and how much stock was deducted.
-- Both nullable: legacy logs and meds without structured dose info just store NULL.
ALTER TABLE med_logs ADD COLUMN container_id TEXT REFERENCES medication_containers(id) ON DELETE SET NULL;
ALTER TABLE med_logs ADD COLUMN stock_deducted REAL;
