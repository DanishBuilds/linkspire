// services/workers/internal/store/store.go
package store

import (
	"context"
	"encoding/json"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Store struct{ Pool *pgxpool.Pool }

func New(ctx context.Context, dsn string, max int32) (*Store, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	if max > 0 {
		cfg.MaxConns = max
	}
	p, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := p.Ping(ctx); err != nil {
		p.Close()
		return nil, err
	}
	return &Store{Pool: p}, nil
}

func (s *Store) Close() { s.Pool.Close() }

type LinkRow struct {
	DomainID, WorkspaceID int64
	Code, Destination, Verdict string
}

func (s *Store) LinksNeedingScan(ctx context.Context, limit int) ([]LinkRow, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT domain_id, code, workspace_id, destination_url, scan_verdict::text
		FROM links WHERE deleted_at IS NULL AND (
		  scan_verdict IN ('pending','suspicious')
		  OR (scan_verdict = 'clean' AND updated_at < now() - INTERVAL '24 hours')
		)
		ORDER BY updated_at ASC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LinkRow
	for rows.Next() {
		var l LinkRow
		if err := rows.Scan(&l.DomainID, &l.Code, &l.WorkspaceID, &l.Destination, &l.Verdict); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Store) SaveScan(ctx context.Context, l LinkRow, scanner, verdict string, detail any) error {
	b, _ := json.Marshal(detail)
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO scan_results (domain_id, code, scanner, verdict, detail)
		VALUES ($1,$2,$3,$4::scan_verdict,$5)`, l.DomainID, l.Code, scanner, verdict, b)
	if err != nil {
		return err
	}
	_, err = s.Pool.Exec(ctx, `
		UPDATE links SET scan_verdict=$3::scan_verdict,
		                 status = CASE WHEN $3 = 'malicious' THEN 'flagged_malicious'::link_status ELSE status END,
		                 updated_at=now()
		WHERE domain_id=$1 AND code=$2`, l.DomainID, l.Code, verdict)
	return err
}

func (s *Store) LinksNeedingMetadata(ctx context.Context, limit int) ([]LinkRow, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT l.domain_id, l.code, l.workspace_id, l.destination_url, l.scan_verdict::text
		FROM links l
		LEFT JOIN link_metadata m ON m.domain_id=l.domain_id AND m.code=l.code
		WHERE l.deleted_at IS NULL AND l.is_password_protected = false
		  AND (m.code IS NULL OR m.refresh_after < now())
		ORDER BY l.created_at DESC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []LinkRow
	for rows.Next() {
		var l LinkRow
		if err := rows.Scan(&l.DomainID, &l.Code, &l.WorkspaceID, &l.Destination, &l.Verdict); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (s *Store) UpsertMetadata(ctx context.Context, l LinkRow, title, desc, image, favicon, canonical, finalURL string, status int, robots bool, og, twitter, jsonld []byte) error {
	_, err := s.Pool.Exec(ctx, `
		INSERT INTO link_metadata (domain_id, code, title, description, image_url, favicon_url, canonical_url,
		                           og, twitter, jsonld, final_url, http_status, robots_allowed, fetched_at, refresh_after)
		VALUES ($1,$2,$3,$4,$5,$6,$7, COALESCE($8::jsonb,'{}'), COALESCE($9::jsonb,'{}'), COALESCE($10::jsonb,'{}'),
		        $11,$12,$13, now(), now() + INTERVAL '24 hours')
		ON CONFLICT (domain_id, code) DO UPDATE SET
		  title=excluded.title, description=excluded.description, image_url=excluded.image_url,
		  favicon_url=excluded.favicon_url, canonical_url=excluded.canonical_url,
		  og=excluded.og, twitter=excluded.twitter, jsonld=excluded.jsonld,
		  final_url=excluded.final_url, http_status=excluded.http_status,
		  robots_allowed=excluded.robots_allowed, fetched_at=now(),
		  refresh_after=now() + INTERVAL '24 hours'`,
		l.DomainID, l.Code, title, desc, image, favicon, canonical, nullJSON(og), nullJSON(twitter), nullJSON(jsonld),
		finalURL, status, robots)
	return err
}

func nullJSON(b []byte) any {
	if len(b) == 0 {
		return []byte("{}")
	}
	return b
}

type DomainRow struct {
	ID, WorkspaceID int64
	Hostname, Token, CNAME, State string
	Attempts                      int
}

func (s *Store) DomainsDue(ctx context.Context, limit int) ([]DomainRow, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, workspace_id, hostname, verify_token, cname_target, state::text, check_attempts
		FROM domains WHERE state IN ('pending','verified') AND (next_check_at IS NULL OR next_check_at <= now())
		ORDER BY next_check_at ASC NULLS FIRST LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DomainRow
	for rows.Next() {
		var d DomainRow
		if err := rows.Scan(&d.ID, &d.WorkspaceID, &d.Hostname, &d.Token, &d.CNAME, &d.State, &d.Attempts); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) UpdateDomainCheck(ctx context.Context, id int64, state string, ok bool, attempts int) error {
	next := time.Now().Add(backoff(attempts))
	if ok && state == "verified" {
		_, err := s.Pool.Exec(ctx, `
			UPDATE domains SET state='verified', verified_at=now(), last_checked_at=now(),
			                   check_attempts=$2, next_check_at=$3, updated_at=now() WHERE id=$1`, id, attempts, next)
		return err
	}
	if ok && state == "active" {
		_, err := s.Pool.Exec(ctx, `
			UPDATE domains SET state='active', last_checked_at=now(), check_attempts=$2, next_check_at=NULL, updated_at=now()
			WHERE id=$1`, id, attempts)
		return err
	}
	if attempts >= 12 {
		_, err := s.Pool.Exec(ctx, `
			UPDATE domains SET state='failed', last_checked_at=now(), check_attempts=$2, next_check_at=NULL, updated_at=now()
			WHERE id=$1`, id, attempts)
		return err
	}
	_, err := s.Pool.Exec(ctx, `
		UPDATE domains SET last_checked_at=now(), check_attempts=$2, next_check_at=$3, updated_at=now() WHERE id=$1`,
		id, attempts, next)
	return err
}

func backoff(attempts int) time.Duration {
	d := time.Duration(1<<min(attempts, 8)) * time.Second
	if d > 30*time.Minute {
		return 30 * time.Minute
	}
	return d
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type QRRow struct {
	ID, DomainID int64
	Code, Format, EC, FG, BG, ObjectKey string
	Size, Margin                        int
	HasLogo                             bool
}

func (s *Store) QRPending(ctx context.Context, limit int) ([]QRRow, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, domain_id, code, format::text, size, margin, ec_level, fg_color, bg_color, has_logo, COALESCE(object_key,'')
		FROM qr_codes WHERE object_key IS NULL OR object_key = '' LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []QRRow
	for rows.Next() {
		var q QRRow
		if err := rows.Scan(&q.ID, &q.DomainID, &q.Code, &q.Format, &q.Size, &q.Margin, &q.EC, &q.FG, &q.BG, &q.HasLogo, &q.ObjectKey); err != nil {
			return nil, err
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

func (s *Store) QRSetKey(ctx context.Context, id int64, key string) error {
	_, err := s.Pool.Exec(ctx, `UPDATE qr_codes SET object_key=$2, updated_at=now() WHERE id=$1`, id, key)
	return err
}

func (s *Store) RetentionPolicies(ctx context.Context) (map[string]int, error) {
	rows, err := s.Pool.Query(ctx, `SELECT data_class, retention_days FROM retention_policies`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]int{}
	for rows.Next() {
		var k string
		var d int
		if err := rows.Scan(&k, &d); err != nil {
			return nil, err
		}
		m[k] = d
	}
	return m, rows.Err()
}

func (s *Store) PurgeDeletedAccounts(ctx context.Context, older time.Duration) (int64, error) {
	tag, err := s.Pool.Exec(ctx, `
		DELETE FROM users WHERE deleted_at IS NOT NULL AND deleted_at < $1`, time.Now().Add(-older))
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}

type DSARRow struct {
	ID, WorkspaceID, UserID int64
	Type, Email             string
}

func (s *Store) OpenDSARs(ctx context.Context, limit int) ([]DSARRow, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT id, COALESCE(workspace_id,0), COALESCE(subject_user_id,0), type::text, subject_email
		FROM data_subject_requests WHERE status IN ('received','verifying','in_progress')
		ORDER BY requested_at ASC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DSARRow
	for rows.Next() {
		var d DSARRow
		if err := rows.Scan(&d.ID, &d.WorkspaceID, &d.UserID, &d.Type, &d.Email); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) CompleteDSAR(ctx context.Context, id int64) error {
	_, err := s.Pool.Exec(ctx, `UPDATE data_subject_requests SET status='completed', completed_at=now() WHERE id=$1`, id)
	return err
}

func (s *Store) EraseUser(ctx context.Context, userID int64) error {
	tx, err := s.Pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)
	_, _ = tx.Exec(ctx, `DELETE FROM webauthn_credentials WHERE user_id=$1`, userID)
	_, _ = tx.Exec(ctx, `UPDATE users SET email='erased+'||id::string||'@invalid.invalid', password_hash=NULL,
	                     mfa_totp_secret=NULL, mfa_enabled=false, status='deleted', deleted_at=now(), updated_at=now()
	                     WHERE id=$1`, userID)
	return tx.Commit(ctx)
}

func (s *Store) SetGPC(ctx context.Context, userID int64, v bool) error {
	_, err := s.Pool.Exec(ctx, `UPDATE users SET gpc_optout=$2, updated_at=now() WHERE id=$1`, userID, v)
	return err
}

func (s *Store) RestrictUser(ctx context.Context, userID int64) error {
	_, err := s.Pool.Exec(ctx, `UPDATE users SET status='suspended', updated_at=now() WHERE id=$1`, userID)
	return err
}

type Delivery struct {
	ID, EndpointID int64
	URL            string
	Secret         []byte
	Payload        []byte
	Attempts       int
}

func (s *Store) DueDeliveries(ctx context.Context, limit int) ([]Delivery, error) {
	rows, err := s.Pool.Query(ctx, `
		SELECT d.id, d.endpoint_id, e.url, e.secret, d.payload, d.attempts
		FROM webhook_deliveries d JOIN webhook_endpoints e ON e.id = d.endpoint_id
		WHERE d.delivered_at IS NULL AND d.next_attempt_at <= now() AND e.active
		ORDER BY d.next_attempt_at ASC LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Delivery
	for rows.Next() {
		var d Delivery
		if err := rows.Scan(&d.ID, &d.EndpointID, &d.URL, &d.Secret, &d.Payload, &d.Attempts); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (s *Store) MarkDelivered(ctx context.Context, id int64) error {
	_, err := s.Pool.Exec(ctx, `UPDATE webhook_deliveries SET delivered_at=now() WHERE id=$1`, id)
	return err
}

func (s *Store) MarkDeliveryFail(ctx context.Context, id int64, attempts int, msg string) error {
	_, err := s.Pool.Exec(ctx, `
		UPDATE webhook_deliveries SET attempts=$2, last_error=$3, next_attempt_at=now() + INTERVAL '1 minute' * $2
		WHERE id=$1`, id, attempts, msg)
	return err
}
