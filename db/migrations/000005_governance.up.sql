-- db/migrations/000005_governance.up.sql
-- Governance, privacy & operational tables: audit log, DSAR tracking, retention policy,
-- reserved slugs, idempotency keys, webhook endpoints.

-- Append-only audit trail. Contains NO PII (no raw IP/UA/email); actors referenced by id.
-- Retained 90 days (security data class) via row-level TTL.
CREATE TABLE audit_log (
    id               INT8        NOT NULL DEFAULT unique_rowid(),
    workspace_id     INT8,                                   -- NULL for platform-level actions
    actor_user_id    INT8,
    actor_api_key_id INT8,
    action           STRING      NOT NULL,                   -- e.g. 'link.create', 'domain.verify'
    target_type      STRING,                                 -- 'link' | 'domain' | ...
    target_id        STRING,
    -- Truncated client fingerprint only (e.g. IPv4 /24, IPv6 /48) — never a raw IP.
    client_fingerprint STRING,
    metadata         JSONB       NOT NULL DEFAULT '{}'::JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_audit_log PRIMARY KEY (id)
) WITH (ttl_expire_after = '90 days', ttl_job_cron = '@daily');
CREATE INDEX ix_audit_workspace ON audit_log (workspace_id, created_at DESC);
CREATE INDEX ix_audit_target ON audit_log (target_type, target_id, created_at DESC);

-- Data-subject requests with SLA timers and an audit trail of handling.
CREATE TABLE data_subject_requests (
    id               INT8        NOT NULL DEFAULT unique_rowid(),
    workspace_id     INT8,
    subject_user_id  INT8,
    -- Requester contact needed to act on the request; pinned to the residency region.
    subject_email    STRING      NOT NULL,
    type             dsar_type   NOT NULL,
    status           dsar_status NOT NULL DEFAULT 'received',
    regime           STRING      NOT NULL DEFAULT 'gdpr',     -- 'gdpr' | 'ccpa'
    verification_method STRING,
    sla_due_at       TIMESTAMPTZ NOT NULL,                    -- +30d GDPR / +45d CCPA
    requested_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at     TIMESTAMPTZ,
    report_object_key STRING,                                 -- S3 key of export (access requests)
    notes            JSONB       NOT NULL DEFAULT '{}'::JSONB,
    CONSTRAINT pk_dsar PRIMARY KEY (id),
    CONSTRAINT fk_dsar_workspace FOREIGN KEY (workspace_id)
        REFERENCES workspaces (id) ON DELETE SET NULL,
    CONSTRAINT fk_dsar_user FOREIGN KEY (subject_user_id)
        REFERENCES users (id) ON DELETE SET NULL,
    CONSTRAINT ck_dsar_regime CHECK (regime IN ('gdpr', 'ccpa'))
);
-- SLA monitor: find open requests approaching their deadline.
CREATE INDEX ix_dsar_sla ON data_subject_requests (sla_due_at ASC)
    WHERE status NOT IN ('completed', 'rejected');

-- Retention policy per data class. Config table read by the retention job and ClickHouse
-- TTL reconciliation. GLOBAL semantics (same everywhere) — applied as locality in 000006.
CREATE TABLE retention_policies (
    data_class     STRING      NOT NULL,
    retention_days INT4        NOT NULL,
    description    STRING      NOT NULL,
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_retention PRIMARY KEY (data_class),
    CONSTRAINT ck_retention_days CHECK (retention_days > 0)
);
INSERT INTO retention_policies (data_class, retention_days, description) VALUES
    ('raw_click_events',     30,  'Raw click events in ClickHouse (TTL drop).'),
    ('aggregated_analytics', 760, 'Aggregated daily rollups (~25 months).'),
    ('security_logs',        90,  'Audit/security logs incl. truncated fingerprints.'),
    ('deleted_account_data', 30,  'Grace window before hard purge of deleted accounts.'),
    ('backups',              35,  'Backup retention / expiry.');

-- Reserved + profanity slug blocklist for vanity aliases. GLOBAL (every region).
CREATE TABLE reserved_slugs (
    slug   STRING NOT NULL,
    reason STRING NOT NULL,                                  -- 'reserved' | 'profanity'
    CONSTRAINT pk_reserved_slugs PRIMARY KEY (slug),
    CONSTRAINT ck_reserved_reason CHECK (reason IN ('reserved', 'profanity'))
);
INSERT INTO reserved_slugs (slug, reason) VALUES
    ('api', 'reserved'), ('admin', 'reserved'), ('login', 'reserved'),
    ('logout', 'reserved'), ('signup', 'reserved'), ('dashboard', 'reserved'),
    ('settings', 'reserved'), ('billing', 'reserved'), ('help', 'reserved'),
    ('support', 'reserved'), ('status', 'reserved'), ('health', 'reserved'),
    ('robots.txt', 'reserved'), ('favicon.ico', 'reserved'), ('.well-known', 'reserved'),
    ('v1', 'reserved'), ('qr', 'reserved'), ('privacy', 'reserved'), ('terms', 'reserved');

-- Idempotency keys for safe POST retries. Short-lived via row-level TTL (24h).
CREATE TABLE idempotency_keys (
    workspace_id       INT8        NOT NULL,
    idempotency_key    STRING      NOT NULL,
    request_hash       BYTES       NOT NULL,                 -- hash of canonical request body
    response_status    INT2,
    response_body_hash BYTES,
    link_ref           STRING,                               -- created resource ref, if any
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_idempotency PRIMARY KEY (workspace_id, idempotency_key)
) WITH (ttl_expire_after = '24 hours', ttl_job_cron = '@hourly');

-- Outbound webhook endpoints (link.created, link.flagged, domain.verified).
CREATE TABLE webhook_endpoints (
    id            INT8        NOT NULL DEFAULT unique_rowid(),
    workspace_id  INT8        NOT NULL,
    url           STRING      NOT NULL,
    secret        BYTES       NOT NULL,                      -- HMAC signing secret (encrypted)
    events        STRING[]    NOT NULL DEFAULT ARRAY[]::STRING[],
    active        BOOL        NOT NULL DEFAULT true,
    failure_count INT4        NOT NULL DEFAULT 0,
    last_delivery_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_webhook_endpoints PRIMARY KEY (id),
    CONSTRAINT fk_webhook_workspace FOREIGN KEY (workspace_id)
        REFERENCES workspaces (id) ON DELETE CASCADE
);
CREATE INDEX ix_webhook_workspace ON webhook_endpoints (workspace_id);
