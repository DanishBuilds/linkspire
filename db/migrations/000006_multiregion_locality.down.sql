-- db/migrations/000006_multiregion_locality.down.sql
-- Revert table localities to REGIONAL BY TABLE in the primary region. Removing database
-- regions (ADD REGION) is a deliberate operational step and is intentionally NOT automated
-- here to avoid data-movement surprises during a rollback.

ALTER TABLE webhook_endpoints     SET LOCALITY REGIONAL BY TABLE;
ALTER TABLE idempotency_keys      SET LOCALITY REGIONAL BY TABLE;
ALTER TABLE data_subject_requests SET LOCALITY REGIONAL BY TABLE;
ALTER TABLE audit_log             SET LOCALITY REGIONAL BY TABLE;
ALTER TABLE scan_results          SET LOCALITY REGIONAL BY TABLE;
ALTER TABLE link_metadata         SET LOCALITY REGIONAL BY TABLE;
ALTER TABLE qr_codes              SET LOCALITY REGIONAL BY TABLE;
ALTER TABLE link_passwords        SET LOCALITY REGIONAL BY TABLE;
ALTER TABLE links                 SET LOCALITY REGIONAL BY TABLE;
ALTER TABLE domains               SET LOCALITY REGIONAL BY TABLE;
ALTER TABLE webauthn_credentials  SET LOCALITY REGIONAL BY TABLE;
ALTER TABLE api_keys              SET LOCALITY REGIONAL BY TABLE;
ALTER TABLE memberships           SET LOCALITY REGIONAL BY TABLE;
ALTER TABLE users                 SET LOCALITY REGIONAL BY TABLE;
ALTER TABLE workspaces            SET LOCALITY REGIONAL BY TABLE;

ALTER TABLE retention_policies SET LOCALITY REGIONAL BY TABLE;
ALTER TABLE reserved_slugs     SET LOCALITY REGIONAL BY TABLE;
