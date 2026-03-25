-- 0004_drop_severity_udf.sql
-- C1: Remove the greatest_severity Postgres UDF.
--
-- Severity resolution is now performed in Go application code using a rank
-- map (see workers/internal/incident/store.go: maxSeverity()):
--
--   critical=4 > high=3 > med=2 > low=1
--
-- Using IF EXISTS makes this safe to re-run (idempotent).

drop function if exists greatest_severity(text, text);
