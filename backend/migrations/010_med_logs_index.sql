-- Add composite indexes on med_logs for date-filtered queries.
-- med_logs uses created_at/given_at for date filtering (no timestamp column).
CREATE INDEX IF NOT EXISTS idx_med_logs_baby_created ON med_logs (baby_id, created_at);
CREATE INDEX IF NOT EXISTS idx_med_logs_baby_given ON med_logs (baby_id, given_at);
