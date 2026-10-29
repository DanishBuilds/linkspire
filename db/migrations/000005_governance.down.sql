-- db/migrations/000005_governance.down.sql
DROP TABLE IF EXISTS webhook_endpoints;
DROP TABLE IF EXISTS idempotency_keys;
DROP TABLE IF EXISTS reserved_slugs;
DROP TABLE IF EXISTS retention_policies;
DROP TABLE IF EXISTS data_subject_requests;
DROP TABLE IF EXISTS audit_log;
