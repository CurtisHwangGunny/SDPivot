-- This repair may adopt a column created by an earlier baseline, so rollback
-- must not remove a column that migration 000024 did not necessarily create.
SELECT 1;
