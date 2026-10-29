-- db/migrations/000002_identity.up.sql
-- Identity & tenancy: workspaces (tenant root), users, memberships, api_keys, webauthn.
-- Surrogate PKs default to unique_rowid() so inserts are runnable without an app-supplied
-- Snowflake id; the application MAY still supply a 64-bit Snowflake value explicitly.

-- Tenant root. `home_region` pins data residency (active-passive write-home per tenant).
CREATE TABLE workspaces (
    id          INT8        NOT NULL DEFAULT unique_rowid(),
    name        STRING      NOT NULL,
    slug        STRING      NOT NULL,
    home_region STRING      NOT NULL DEFAULT 'us',          -- 'us' | 'eu' (residency)
    plan        STRING      NOT NULL DEFAULT 'free',
    settings    JSONB       NOT NULL DEFAULT '{}'::JSONB,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at  TIMESTAMPTZ,
    CONSTRAINT pk_workspaces PRIMARY KEY (id),
    CONSTRAINT uq_workspaces_slug UNIQUE (slug),
    CONSTRAINT ck_workspaces_region CHECK (home_region IN ('us', 'eu'))
);

CREATE TABLE users (
    id                INT8        NOT NULL DEFAULT unique_rowid(),
    email             STRING      NOT NULL,
    -- normalized form used for case-insensitive uniqueness without an expression index.
    email_normalized  STRING      NOT NULL AS (lower(email)) STORED,
    email_verified    BOOL        NOT NULL DEFAULT false,
    -- Argon2id PHC-encoded string; NULL for OIDC-only accounts. Never plaintext.
    password_hash     STRING,
    -- TOTP secret encrypted at rest (envelope-encrypted by the app; never plaintext here).
    mfa_totp_secret   BYTES,
    mfa_enabled       BOOL        NOT NULL DEFAULT false,
    home_region       STRING      NOT NULL DEFAULT 'us',
    status            user_status NOT NULL DEFAULT 'active',
    -- Privacy: remembered Global Privacy Control / Do-Not-Sell preference.
    gpc_optout        BOOL        NOT NULL DEFAULT false,
    created_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at        TIMESTAMPTZ NOT NULL DEFAULT now(),
    deleted_at        TIMESTAMPTZ,
    CONSTRAINT pk_users PRIMARY KEY (id),
    CONSTRAINT uq_users_email UNIQUE (email_normalized),
    CONSTRAINT ck_users_region CHECK (home_region IN ('us', 'eu'))
);

-- Membership = user ↔ workspace with a role. PK leads with workspace_id so a tenant's
-- membership rows group together on the same ranges (tenant colocation).
CREATE TABLE memberships (
    workspace_id INT8        NOT NULL,
    user_id      INT8        NOT NULL,
    role         member_role NOT NULL DEFAULT 'member',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_memberships PRIMARY KEY (workspace_id, user_id),
    CONSTRAINT fk_memberships_workspace FOREIGN KEY (workspace_id)
        REFERENCES workspaces (id) ON DELETE CASCADE,
    CONSTRAINT fk_memberships_user FOREIGN KEY (user_id)
        REFERENCES users (id) ON DELETE CASCADE
);
-- Reverse lookup: "which workspaces does this user belong to?"
CREATE INDEX ix_memberships_user ON memberships (user_id);

-- Scoped, hashed, rotatable API keys. We store only a hash of the secret + a short prefix
-- for identification in UIs/logs. The plaintext key is shown once at creation.
CREATE TABLE api_keys (
    id           INT8        NOT NULL DEFAULT unique_rowid(),
    workspace_id INT8        NOT NULL,
    name         STRING      NOT NULL,
    key_prefix   STRING      NOT NULL,              -- e.g. 'lk_live_ab12' (non-secret)
    key_hash     BYTES       NOT NULL,              -- SHA-256 of the secret portion
    scopes       STRING[]    NOT NULL DEFAULT ARRAY[]::STRING[],
    rotated_from INT8,                              -- previous key id during rotation overlap
    created_by   INT8,
    last_used_at TIMESTAMPTZ,
    expires_at   TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    revoked_at   TIMESTAMPTZ,
    CONSTRAINT pk_api_keys PRIMARY KEY (id),
    CONSTRAINT uq_api_keys_hash UNIQUE (key_hash),
    CONSTRAINT fk_api_keys_workspace FOREIGN KEY (workspace_id)
        REFERENCES workspaces (id) ON DELETE CASCADE,
    CONSTRAINT fk_api_keys_creator FOREIGN KEY (created_by)
        REFERENCES users (id) ON DELETE SET NULL
);
CREATE INDEX ix_api_keys_workspace ON api_keys (workspace_id, created_at DESC);

-- WebAuthn/passkey credentials for phishing-resistant MFA.
CREATE TABLE webauthn_credentials (
    id            INT8        NOT NULL DEFAULT unique_rowid(),
    user_id       INT8        NOT NULL,
    credential_id BYTES       NOT NULL,
    public_key    BYTES       NOT NULL,
    sign_count    INT8        NOT NULL DEFAULT 0,
    label         STRING,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT pk_webauthn PRIMARY KEY (id),
    CONSTRAINT uq_webauthn_cred UNIQUE (credential_id),
    CONSTRAINT fk_webauthn_user FOREIGN KEY (user_id)
        REFERENCES users (id) ON DELETE CASCADE
);
CREATE INDEX ix_webauthn_user ON webauthn_credentials (user_id);
