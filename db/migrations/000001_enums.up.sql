-- db/migrations/000001_enums.up.sql
-- Enumerated types shared across the schema.
-- CockroachDB-compatible. Enums are created before any table that references them,
-- in their own migration, so later `ALTER TYPE ... ADD VALUE` migrations never share a
-- transaction with statements that use the new value (CockroachDB restriction).

CREATE TYPE user_status AS ENUM ('active', 'suspended', 'deleted');

CREATE TYPE member_role AS ENUM ('owner', 'admin', 'member', 'viewer');

CREATE TYPE domain_state AS ENUM ('pending', 'verified', 'active', 'failed', 'revoked');

CREATE TYPE link_status AS ENUM (
    'active',
    'expired',
    'disabled',
    'flagged_malicious',
    'password_protected'
);

CREATE TYPE scan_verdict AS ENUM ('pending', 'clean', 'suspicious', 'malicious');

CREATE TYPE qr_format AS ENUM ('svg', 'png');

CREATE TYPE dsar_type AS ENUM (
    'access',
    'rectification',
    'erasure',
    'restriction',
    'objection',
    'do_not_sell'
);

CREATE TYPE dsar_status AS ENUM (
    'received',
    'verifying',
    'in_progress',
    'completed',
    'rejected'
);
