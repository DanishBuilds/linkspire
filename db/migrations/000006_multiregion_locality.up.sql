-- db/migrations/000006_multiregion_locality.up.sql
--
-- PRODUCTION / MULTI-REGION ONLY. Skip this migration on a single-node dev cluster.
-- It requires the CockroachDB cluster nodes to be started with --locality flags for the
-- region names referenced below. Edit the database name ("linkspire") and region names to
-- match your environment before applying (golang-migrate: run with a build tag / env gate).
--
-- Strategy (active-passive write-home per tenant, ADR-002):
--   * GLOBAL tables: read-mostly reference data, low-latency reads in every region, writes
--     are rare and go through consensus. Used for reserved slugs & retention policy.
--   * REGIONAL BY ROW: tenant data. The app sets each row's region to the owning tenant's
--     home_region ('us'/'eu') so link data & analytics are physically pinned for residency.
--     Reads/writes for a tenant are served from its home region (fast, compliant).
--
-- SURVIVE REGION FAILURE gives RPO=0 across a full region loss (RTO<5min via auto-failover).

ALTER DATABASE linkspire PRIMARY REGION "us-east1";
ALTER DATABASE linkspire ADD REGION "europe-west1";
ALTER DATABASE linkspire SURVIVE REGION FAILURE;

-- Global reference tables (fast local reads everywhere).
ALTER TABLE reserved_slugs     SET LOCALITY GLOBAL;
ALTER TABLE retention_policies SET LOCALITY GLOBAL;

-- Tenant-scoped data pinned by region. The hidden `crdb_region` column defaults to the
-- gateway region; the application overrides it to the tenant's home region on insert.
ALTER TABLE workspaces            SET LOCALITY REGIONAL BY ROW;
ALTER TABLE users                 SET LOCALITY REGIONAL BY ROW;
ALTER TABLE memberships           SET LOCALITY REGIONAL BY ROW;
ALTER TABLE api_keys              SET LOCALITY REGIONAL BY ROW;
ALTER TABLE webauthn_credentials  SET LOCALITY REGIONAL BY ROW;
ALTER TABLE domains               SET LOCALITY REGIONAL BY ROW;
ALTER TABLE links                 SET LOCALITY REGIONAL BY ROW;
ALTER TABLE link_passwords        SET LOCALITY REGIONAL BY ROW;
ALTER TABLE qr_codes              SET LOCALITY REGIONAL BY ROW;
ALTER TABLE link_metadata         SET LOCALITY REGIONAL BY ROW;
ALTER TABLE scan_results          SET LOCALITY REGIONAL BY ROW;
ALTER TABLE audit_log             SET LOCALITY REGIONAL BY ROW;
ALTER TABLE data_subject_requests SET LOCALITY REGIONAL BY ROW;
ALTER TABLE idempotency_keys      SET LOCALITY REGIONAL BY ROW;
ALTER TABLE webhook_endpoints     SET LOCALITY REGIONAL BY ROW;

-- NOTE on global uniqueness: unique indexes (domains.hostname, users.email_normalized,
-- links PK (domain_id, code)) remain globally enforced even under REGIONAL BY ROW, so no
-- duplicate codes/hostnames can ever be created across regions (the core RPO=0 guarantee).
