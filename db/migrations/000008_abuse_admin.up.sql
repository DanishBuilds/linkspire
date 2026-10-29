-- db/migrations/000008_abuse_admin.up.sql
-- Abuse review queue plus a platform-staff flag for the admin surface.

CREATE TABLE abuse_reports (
    id             INT8        NOT NULL DEFAULT unique_rowid(),
    short_url      STRING      NOT NULL,
    reason         STRING      NOT NULL,
    status         STRING      NOT NULL DEFAULT 'open',   -- open|reviewing|actioned|dismissed
    reporter_email STRING,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    resolved_at    TIMESTAMPTZ,
    resolved_by    INT8,
    CONSTRAINT pk_abuse_reports PRIMARY KEY (id),
    CONSTRAINT ck_abuse_status CHECK (status IN ('open', 'reviewing', 'actioned', 'dismissed'))
);
CREATE INDEX ix_abuse_reports_status ON abuse_reports (status, created_at DESC);

-- Platform staff can view and action the abuse queue. Workspace RBAC is unchanged.
ALTER TABLE users ADD COLUMN IF NOT EXISTS is_staff BOOL NOT NULL DEFAULT false;
