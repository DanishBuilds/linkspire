-- db/migrations/000007_ops.down.sql
DELETE FROM domains WHERE id = 1;
DELETE FROM workspaces WHERE id = 1;
DROP TABLE IF EXISTS webhook_deliveries;
DROP TABLE IF EXISTS hilo_counters;
