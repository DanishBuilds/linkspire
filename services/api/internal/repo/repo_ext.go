// services/api/internal/repo/repo_ext.go
// Phase 7+ additions: platform-staff flag, abuse queue, QR listing, and
// WebAuthn credential storage. Kept separate from repo.go to isolate new code.
package repo

import (
	"context"
	"errors"
	"time"
)

// ---- staff ----

func (r *Repo) IsStaff(ctx context.Context, userID int64) (bool, error) {
	var staff bool
	err := r.Pool.QueryRow(ctx, `SELECT is_staff FROM users WHERE id=$1 AND deleted_at IS NULL`, userID).Scan(&staff)
	return staff, mapErr(err)
}

// ---- abuse queue ----

type AbuseReport struct {
	ID            int64
	ShortURL      string
	Reason        string
	Status        string
	ReporterEmail *string
	CreatedAt     time.Time
	ResolvedAt    *time.Time
}

func (r *Repo) InsertAbuseReport(ctx context.Context, shortURL, reason string, reporterEmail *string) (int64, error) {
	var id int64
	err := r.Pool.QueryRow(ctx, `
		INSERT INTO abuse_reports (short_url, reason, reporter_email)
		VALUES ($1,$2,$3) RETURNING id`, shortURL, reason, reporterEmail).Scan(&id)
	return id, mapErr(err)
}

func (r *Repo) ListAbuseReports(ctx context.Context, status string, limit int) ([]AbuseReport, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	q := `SELECT id, short_url, reason, status, reporter_email, created_at, resolved_at
	      FROM abuse_reports`
	args := []any{}
	if status != "" {
		q += ` WHERE status=$1`
		args = append(args, status)
		q += ` ORDER BY created_at DESC LIMIT $2`
		args = append(args, limit)
	} else {
		q += ` ORDER BY created_at DESC LIMIT $1`
		args = append(args, limit)
	}
	rows, err := r.Pool.Query(ctx, q, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []AbuseReport
	for rows.Next() {
		var a AbuseReport
		if err := rows.Scan(&a.ID, &a.ShortURL, &a.Reason, &a.Status, &a.ReporterEmail, &a.CreatedAt, &a.ResolvedAt); err != nil {
			return nil, err
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

func (r *Repo) ResolveAbuseReport(ctx context.Context, id, staffID int64, status string) error {
	ct, err := r.Pool.Exec(ctx, `
		UPDATE abuse_reports SET status=$1, resolved_at=now(), resolved_by=$2 WHERE id=$3`,
		status, staffID, id)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ---- QR listing ----

type QRRow struct {
	ID        int64
	Format    string
	ECLevel   string
	Size      int
	ObjectKey *string
	CreatedAt time.Time
}

func (r *Repo) GetQR(ctx context.Context, domainID int64, code string, id int64) (QRRow, error) {
	var q QRRow
	err := r.Pool.QueryRow(ctx, `
		SELECT id, format::text, ec_level, size, object_key, created_at
		FROM qr_codes WHERE id=$1 AND domain_id=$2 AND code=$3`, id, domainID, code,
	).Scan(&q.ID, &q.Format, &q.ECLevel, &q.Size, &q.ObjectKey, &q.CreatedAt)
	return q, err
}

func (r *Repo) ListQR(ctx context.Context, domainID int64, code string) ([]QRRow, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, format::text, ec_level, size, object_key, created_at
		FROM qr_codes WHERE domain_id=$1 AND code=$2 ORDER BY created_at DESC`, domainID, code)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []QRRow
	for rows.Next() {
		var q QRRow
		if err := rows.Scan(&q.ID, &q.Format, &q.ECLevel, &q.Size, &q.ObjectKey, &q.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

// ---- WebAuthn credentials ----

type WebAuthnCred struct {
	ID           int64
	CredentialID []byte
	PublicKey    []byte
	SignCount    int64
	Label        *string
}

func (r *Repo) ListWebAuthnByUser(ctx context.Context, userID int64) ([]WebAuthnCred, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, credential_id, public_key, sign_count, label
		FROM webauthn_credentials WHERE user_id=$1 ORDER BY created_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WebAuthnCred
	for rows.Next() {
		var c WebAuthnCred
		if err := rows.Scan(&c.ID, &c.CredentialID, &c.PublicKey, &c.SignCount, &c.Label); err != nil {
			return nil, err
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (r *Repo) InsertWebAuthnCred(ctx context.Context, userID int64, credID, pubKey []byte, signCount int64, label string) error {
	_, err := r.Pool.Exec(ctx, `
		INSERT INTO webauthn_credentials (user_id, credential_id, public_key, sign_count, label)
		VALUES ($1,$2,$3,$4,$5)`, userID, credID, pubKey, signCount, label)
	return mapErr(err)
}

func (r *Repo) UpdateWebAuthnSignCount(ctx context.Context, credID []byte, signCount int64) error {
	_, err := r.Pool.Exec(ctx, `UPDATE webauthn_credentials SET sign_count=$1 WHERE credential_id=$2`, signCount, credID)
	return err
}

func (r *Repo) DeleteWebAuthnCred(ctx context.Context, userID, id int64) error {
	ct, err := r.Pool.Exec(ctx, `DELETE FROM webauthn_credentials WHERE id=$1 AND user_id=$2`, id, userID)
	if err != nil {
		return err
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// FindOrCreateOIDCUser returns the user id for an OIDC-authenticated email,
// creating an account (with a workspace) on first login. passwordHash should be
// an unusable Argon2id hash so the account cannot be used for password login.
func (r *Repo) FindOrCreateOIDCUser(ctx context.Context, email, passwordHash, wsName, region string) (int64, error) {
	u, err := r.GetUserByEmail(ctx, email)
	if err == nil {
		return u.ID, nil
	}
	if !errors.Is(err, ErrNotFound) {
		return 0, err
	}
	uid, _, cerr := r.CreateUserWithWorkspace(ctx, email, passwordHash, wsName, region)
	return uid, cerr
}
