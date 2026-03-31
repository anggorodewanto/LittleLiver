-- Convert cal_density from kcal/oz to kcal/mL.
-- Divide by 29.5735 (mL per oz).
-- Recalculate calories as volume_ml * cal_density (new kcal/mL value).
UPDATE feedings
SET cal_density = cal_density / 29.5735,
    calories = volume_ml * (cal_density / 29.5735)
WHERE cal_density IS NOT NULL;
