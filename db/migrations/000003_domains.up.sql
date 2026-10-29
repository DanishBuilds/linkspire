-- db/migrations/000003_domains.up.sql
-- Custom domains (CNAME onboarding + on-demand TLS authorization).

CREATE TABLE domains (
    id              INT8         NOT NULL DEFAULT unique_rowid(),
    workspace_id    INT8         NOT NULL,
    hostname        STRING       NOT NULL,                 -- FQDN, lowercased by the app
    state           domain_state NOT NULL DEFAULT 'pending',
    verify_token    STRING       NOT NULL,                 -- high-entropy TXT ownership token
    cname_target    STRING       NOT NULL DEFAULT 'cname.linkspire.io',
    is_apex         BOOL         NOT NULL DEFAULT false,   -- apex uses ALIAS/ANAME or A fallback
    -- Per-domain settings.
    root_redirect_url STRING,                              -- default redirect for the root path
    custom_404_url    STRING,
    hsts_enabled      BOOL       NOT NULL DEFAULT false,
    -- Verifier bookkeeping (exponential backoff polling).
    verified_at     TIMESTAMPTZ,
    last_checked_at TIMESTAMPTZ,
    next_check_at   TIMESTAMPTZ,
    check_attempts  INT4         NOT NULL DEFAULT 0,
    created_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ  NOT NULL DEFAULT now(),
    revoked_at      TIMESTAMPTZ,
    CONSTRAINT pk_domains PRIMARY KEY (id),
    -- Hostname is globally unique across the whole platform (and all regions): a given
    -- FQDN can be claimed by exactly one workspace. Enforced by a global unique index.
    CONSTRAINT uq_domains_hostname UNIQUE (hostname),
    CONSTRAINT fk_domains_workspace FOREIGN KEY (workspace_id)
        REFERENCES workspaces (id) ON DELETE CASCADE
);

-- Dashboard listing of a workspace's domains.
CREATE INDEX ix_domains_workspace ON domains (workspace_id, created_at DESC);

-- Verifier work queue: find domains due for a DNS re-check. Partial index keeps it tiny —
-- only rows actively being polled are indexed.
CREATE INDEX ix_domains_due_check ON domains (next_check_at ASC)
    WHERE state IN ('pending', 'verified');

-- Fast authorization lookup for the on-demand-TLS "ask" endpoint and SNI routing:
-- "is this hostname served, and by which workspace?"
CREATE INDEX ix_domains_active_host ON domains (hostname)
    STORING (workspace_id, state)
    WHERE state = 'active';
