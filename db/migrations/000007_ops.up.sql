-- db/migrations/000007_ops.up.sql
-- HiLo counter for short-code generation, webhook outbox, and a platform system domain.

CREATE TABLE hilo_counters (
    name       STRING      NOT NULL,
    next_hi    INT8        NOT NULL DEFAULT 0,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_hilo PRIMARY KEY (name),
    CONSTRAINT ck_hilo_hi CHECK (next_hi >= 0)
);
INSERT INTO hilo_counters (name, next_hi) VALUES ('codes', 0);

CREATE TABLE webhook_deliveries (
    id              INT8        NOT NULL DEFAULT unique_rowid(),
    endpoint_id     INT8        NOT NULL,
    workspace_id    INT8        NOT NULL,
    event           STRING      NOT NULL,
    payload         JSONB       NOT NULL,
    attempts        INT4        NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    delivered_at    TIMESTAMPTZ,
    last_error      STRING,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_webhook_deliveries PRIMARY KEY (id),
    CONSTRAINT fk_wd_endpoint FOREIGN KEY (endpoint_id)
        REFERENCES webhook_endpoints (id) ON DELETE CASCADE
);
CREATE INDEX ix_wd_due ON webhook_deliveries (next_attempt_at ASC)
    WHERE delivered_at IS NULL;

-- Platform workspace + default short domain so the API can mint links without
-- waiting for a custom-domain onboarding. Hostname is the system edge host.
INSERT INTO workspaces (id, name, slug, home_region) VALUES
    (1, 'Linkspire Platform', 'linkspire', 'us');
INSERT INTO domains (id, workspace_id, hostname, verify_token, state) VALUES
    (1, 1, 'lnk.linkspire.io', 'platform', 'active');
