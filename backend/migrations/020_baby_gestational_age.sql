-- Optional gestational age at birth for preterm babies.
-- Used to compute corrected age on the dashboard. Both columns nullable;
-- existing rows remain NULL and are treated as full-term (no correction).
ALTER TABLE babies ADD COLUMN gestational_age_weeks INTEGER;
ALTER TABLE babies ADD COLUMN gestational_age_days INTEGER;
