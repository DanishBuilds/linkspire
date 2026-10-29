// services/redirect/internal/repo/postgres.go
package repo

import (
	"context"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/linkspire/linkspire/services/redirect/internal/model"
)

// Repo reads from a CockroachDB (Postgres-wire) read replica. It is only hit on a full cache
// miss; it performs single-range point lookups on the links PK (domain_id, code).
type Repo struct {
	pool *pgxpool.Pool
}

// NewRepo opens a pgx pool against the given DSN.
func NewRepo(ctx context.Context, dsn string, maxConns int32) (*Repo, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	if maxConns > 0 {
		cfg.MaxConns = maxConns
	}
	cfg.MaxConnIdleTime = 5 * time.Minute
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	pctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	if err := pool.Ping(pctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Repo{pool: pool}, nil
}

const lookupSQL = `
SELECT workspace_id, destination_url, status::text, redirect_type,
       scan_verdict::text, is_password_protected, expires_at, click_limit, click_count
FROM links
WHERE domain_id = $1 AND code = $2 AND deleted_at IS NULL`

// Lookup returns the link for (domainID, code). ok=false when the row does not exist.
func (r *Repo) Lookup(ctx context.Context, domainID int64, code string) (model.Link, bool, error) {
	l := model.Link{DomainID: domainID, Code: code}
	var status, verdict string
	err := r.pool.QueryRow(ctx, lookupSQL, domainID, code).Scan(
		&l.WorkspaceID, &l.Destination, &status, &l.RedirectType,
		&verdict, &l.IsPasswordProtected, &l.ExpiresAt, &l.ClickLimit, &l.ClickCount,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return model.Link{}, false, nil
	}
	if err != nil {
		return model.Link{}, false, err
	}
	l.Status = model.LinkStatus(status)
	l.Verdict = verdict
	return l, true, nil
}

const domainByHostSQL = `SELECT id FROM domains WHERE hostname = $1 AND state = 'active'`

// DomainByHost resolves an active hostname to its domain id.
func (r *Repo) DomainByHost(ctx context.Context, host string) (int64, bool, error) {
	var id int64
	err := r.pool.QueryRow(ctx, domainByHostSQL, host).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, err
	}
	return id, true, nil
}

const passwordSQL = `SELECT password_hash FROM link_passwords WHERE domain_id = $1 AND code = $2`

// PasswordHash returns the Argon2id PHC string for a protected link.
func (r *Repo) PasswordHash(ctx context.Context, domainID int64, code string) (string, bool, error) {
	var phc string
	err := r.pool.QueryRow(ctx, passwordSQL, domainID, code).Scan(&phc)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return phc, true, nil
}

// Ping is used by readiness probes.
func (r *Repo) Ping(ctx context.Context) error { return r.pool.Ping(ctx) }

// Close releases the pool.
func (r *Repo) Close() { r.pool.Close() }
