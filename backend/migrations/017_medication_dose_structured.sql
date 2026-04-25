-- Structured dose fields enable auto-decrement of medication stock when a dose is logged.
-- All nullable so existing medications keep working without stock tracking.
ALTER TABLE medications ADD COLUMN dose_amount REAL;
ALTER TABLE medications ADD COLUMN dose_unit TEXT;
ALTER TABLE medications ADD COLUMN low_stock_threshold INTEGER;
ALTER TABLE medications ADD COLUMN expiry_warning_days INTEGER;
