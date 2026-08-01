-- Solid food intake on feedings.
-- Solids are measured either in mL (purees — reuse volume_ml so they still count
-- toward fluid intake) or in grams (amount_g, which is not a fluid volume).
-- ingredients is freeform text listing what the meal contained.
ALTER TABLE feedings ADD COLUMN amount_g REAL;
ALTER TABLE feedings ADD COLUMN ingredients TEXT;
