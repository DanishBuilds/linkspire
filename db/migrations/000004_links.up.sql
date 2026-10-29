-- db/migrations/000004_links.up.sql
-- The hot entity: links, plus its satellite tables (passwords, QR, scraped metadata, scans).
--
-- Distribution key = (domain_id, code). In a Citus deployment this is literally
--   SELECT create_distributed_table('links', 'code');  -- colocated by (domain_id, code)
-- In CockroachDB, distribution is by primary-key range. Because `code` is produced by a
-- keyed Feistel permutation (ADR-003) it is pseudo-random, so the (domain_id, code) key
-- space is already *uniformly distributed*; ranges split evenly with NO monotonic hotspot.
-- That is why we do NOT need USING HASH here — the code entropy provides the same spread a
-- hash-sharded index would, while keeping (domain_id, code) directly FK-referenceable.

CREATE TABLE links (
    domain_id          INT8        NOT NULL,
    code               STRING      NOT NULL,                 -- Base62, len 7 default; vanity up to 64
    workspace_id       INT8        NOT NULL,
    destination_url    STRING      NOT NULL,
    status             link_status NOT NULL DEFAULT 'active',
    redirect_type      INT2        NOT NULL DEFAULT 302,      -- 301|302|307|308 (ADR-004)
    scan_verdict       scan_verdict NOT NULL DEFAULT 'pending',
    is_password_protected BOOL     NOT NULL DEFAULT false,
    -- Owner-supplied preview overrides (used for crawlers; suppressed for password links).
    title_override       STRING,
    description_override STRING,
    image_override       STRING,
    -- Expiry / click-limit state.
    expires_at         TIMESTAMPTZ,
    click_limit        INT8,
    -- Approximate count maintained in BATCHES by the analytics consumer (NOT per-hit) to
    -- avoid hot-row write contention; used only to enforce click_limit. Exact analytics
    -- live in ClickHouse.
    click_count        INT8        NOT NULL DEFAULT 0,
    tags               STRING[]    NOT NULL DEFAULT ARRAY[]::STRING[],
    created_by         INT8,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at         TIMESTAMPTZ,
    CONSTRAINT pk_links PRIMARY KEY (domain_id, code),
    CONSTRAINT fk_links_domain FOREIGN KEY (domain_id)
        REFERENCES domains (id) ON DELETE CASCADE,
    CONSTRAINT fk_links_workspace FOREIGN KEY (workspace_id)
        REFERENCES workspaces (id) ON DELETE CASCADE,
    CONSTRAINT fk_links_creator FOREIGN KEY (created_by)
        REFERENCES users (id) ON DELETE SET NULL,
    CONSTRAINT ck_links_redirect_type CHECK (redirect_type IN (301, 302, 307, 308)),
    CONSTRAINT ck_links_code_len CHECK (length(code) BETWEEN 1 AND 64),
    -- Reject dangerous schemes at the storage boundary as defense-in-depth (the app also
    -- validates). data:/javascript:/vbscript: must never be stored.
    CONSTRAINT ck_links_dest_len CHECK (length(destination_url) BETWEEN 1 AND 2048),
    CONSTRAINT ck_links_dest_scheme CHECK (destination_url ~* '^https?://')
);

-- Dashboard listing + filtering by workspace, newest first. Soft-deleted rows excluded.
CREATE INDEX ix_links_workspace ON links (workspace_id, created_at DESC)
    WHERE deleted_at IS NULL;

-- Expiry sweeper: only index links that can expire. Tiny, cheap to scan.
CREATE INDEX ix_links_expiring ON links (expires_at ASC)
    WHERE expires_at IS NOT NULL AND status = 'active';

-- Re-scan worker queue: links awaiting a verdict or flagged suspicious for periodic re-scan.
CREATE INDEX ix_links_rescan ON links (updated_at ASC)
    WHERE scan_verdict IN ('pending', 'suspicious');

-- Password hashes isolated from the hot `links` row: least-exposure + the redirect path only
-- reads this table when is_password_protected = true.
CREATE TABLE link_passwords (
    domain_id     INT8        NOT NULL,
    code          STRING      NOT NULL,
    -- Argon2id PHC-encoded string, e.g. $argon2id$v=19$m=65536,t=3,p=2$...$...
    password_hash STRING      NOT NULL,
    -- Parameters captured for transparent rehash/upgrade when we tune Argon2id.
    algo_params   JSONB       NOT NULL,
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_link_passwords PRIMARY KEY (domain_id, code),
    CONSTRAINT fk_link_passwords_link FOREIGN KEY (domain_id, code)
        REFERENCES links (domain_id, code) ON DELETE CASCADE
);

CREATE TABLE qr_codes (
    id          INT8        NOT NULL DEFAULT unique_rowid(),
    domain_id   INT8        NOT NULL,
    code        STRING      NOT NULL,
    format      qr_format   NOT NULL,
    size        INT2        NOT NULL DEFAULT 512,
    margin      INT2        NOT NULL DEFAULT 4,
    ec_level    STRING      NOT NULL DEFAULT 'M',            -- L|M|Q|H (H forced when logo set)
    fg_color    STRING      NOT NULL DEFAULT '#000000',
    bg_color    STRING      NOT NULL DEFAULT '#FFFFFF',
    has_logo    BOOL        NOT NULL DEFAULT false,
    -- Deterministic hash of all style inputs; lets us cache and regenerate only on change.
    style_hash  BYTES       NOT NULL,
    object_key  STRING,                                      -- S3 key; NULL until generated
    content_hash BYTES,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_qr_codes PRIMARY KEY (id),
    CONSTRAINT uq_qr_variant UNIQUE (domain_id, code, format, style_hash),
    CONSTRAINT fk_qr_link FOREIGN KEY (domain_id, code)
        REFERENCES links (domain_id, code) ON DELETE CASCADE,
    CONSTRAINT ck_qr_ec CHECK (ec_level IN ('L', 'M', 'Q', 'H'))
);
CREATE INDEX ix_qr_link ON qr_codes (domain_id, code);

CREATE TABLE link_metadata (
    domain_id     INT8        NOT NULL,
    code          STRING      NOT NULL,
    title         STRING,
    description   STRING,
    image_url     STRING,
    favicon_url   STRING,
    canonical_url STRING,
    og            JSONB       NOT NULL DEFAULT '{}'::JSONB,
    twitter       JSONB       NOT NULL DEFAULT '{}'::JSONB,
    jsonld        JSONB       NOT NULL DEFAULT '{}'::JSONB,
    final_url     STRING,                                    -- URL after following redirects
    http_status   INT2,
    content_hash  BYTES,
    robots_allowed BOOL       NOT NULL DEFAULT true,
    fetched_at    TIMESTAMPTZ,
    refresh_after TIMESTAMPTZ,                               -- refresh-TTL for re-scrape
    CONSTRAINT pk_link_metadata PRIMARY KEY (domain_id, code),
    CONSTRAINT fk_link_metadata_link FOREIGN KEY (domain_id, code)
        REFERENCES links (domain_id, code) ON DELETE CASCADE
);
-- Scraper work queue: metadata due for refresh.
CREATE INDEX ix_link_metadata_refresh ON link_metadata (refresh_after ASC)
    WHERE refresh_after IS NOT NULL;

-- Per-scanner results history. Retained for a bounded window via row-level TTL (security
-- data class). Each (scanner, link) run is recorded for auditability and verdict provenance.
CREATE TABLE scan_results (
    id         INT8        NOT NULL DEFAULT unique_rowid(),
    domain_id  INT8        NOT NULL,
    code       STRING      NOT NULL,
    scanner    STRING      NOT NULL,                         -- safe_browsing|urlhaus|virustotal|heuristics
    verdict    scan_verdict NOT NULL,
    detail     JSONB       NOT NULL DEFAULT '{}'::JSONB,
    scanned_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_scan_results PRIMARY KEY (id),
    CONSTRAINT fk_scan_results_link FOREIGN KEY (domain_id, code)
        REFERENCES links (domain_id, code) ON DELETE CASCADE
) WITH (ttl_expire_after = '90 days', ttl_job_cron = '@daily');
CREATE INDEX ix_scan_results_link ON scan_results (domain_id, code, scanned_at DESC);
