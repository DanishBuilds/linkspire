// services/api/internal/repo/repo.go
package repo

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var ErrNotFound = errors.New("not found")
var ErrConflict = errors.New("conflict")

type Repo struct{ Pool *pgxpool.Pool }

func New(ctx context.Context, dsn string, maxConns int32) (*Repo, error) {
	cfg, err := pgxpool.ParseConfig(dsn)
	if err != nil {
		return nil, err
	}
	if maxConns > 0 {
		cfg.MaxConns = maxConns
	}
	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}
	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, err
	}
	return &Repo{Pool: pool}, nil
}

func (r *Repo) Close() { r.Pool.Close() }
func (r *Repo) Ping(ctx context.Context) error { return r.Pool.Ping(ctx) }

type User struct {
	ID           int64
	Email        string
	PasswordHash *string
	MFAEnabled   bool
	MFASecret    []byte
	Status       string
	GPCOptOut    bool
	HomeRegion   string
}

type Membership struct {
	WorkspaceID int64
	UserID      int64
	Role        string
	WSName      string
	HomeRegion  string
}

func (r *Repo) CreateUserWithWorkspace(ctx context.Context, email, hash, wsName, region string) (userID, wsID int64, err error) {
	tx, err := r.Pool.Begin(ctx)
	if err != nil {
		return 0, 0, err
	}
	defer tx.Rollback(ctx)
	err = tx.QueryRow(ctx, `
		INSERT INTO users (email, password_hash, home_region)
		VALUES ($1, $2, $3) RETURNING id`, email, hash, region).Scan(&userID)
	if err != nil {
		return 0, 0, mapErr(err)
	}
	slug := slugify(wsName) + "-" + rand4()
	err = tx.QueryRow(ctx, `
		INSERT INTO workspaces (name, slug, home_region) VALUES ($1, $2, $3) RETURNING id`,
		wsName, slug, region).Scan(&wsID)
	if err != nil {
		return 0, 0, mapErr(err)
	}
	_, err = tx.Exec(ctx, `INSERT INTO memberships (workspace_id, user_id, role) VALUES ($1,$2,'owner')`, wsID, userID)
	if err != nil {
		return 0, 0, err
	}
	return userID, wsID, tx.Commit(ctx)
}

func (r *Repo) GetUserByEmail(ctx context.Context, email string) (User, error) {
	var u User
	err := r.Pool.QueryRow(ctx, `
		SELECT id, email, password_hash, mfa_enabled, COALESCE(mfa_totp_secret, ''::bytes),
		       status::text, gpc_optout, home_region
		FROM users WHERE email_normalized = lower($1) AND deleted_at IS NULL`, email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.MFAEnabled, &u.MFASecret, &u.Status, &u.GPCOptOut, &u.HomeRegion)
	return u, mapErr(err)
}

func (r *Repo) GetUser(ctx context.Context, id int64) (User, error) {
	var u User
	err := r.Pool.QueryRow(ctx, `
		SELECT id, email, password_hash, mfa_enabled, COALESCE(mfa_totp_secret, ''::bytes),
		       status::text, gpc_optout, home_region
		FROM users WHERE id = $1 AND deleted_at IS NULL`, id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.MFAEnabled, &u.MFASecret, &u.Status, &u.GPCOptOut, &u.HomeRegion)
	return u, mapErr(err)
}

func (r *Repo) SetTOTP(ctx context.Context, userID int64, secret []byte, enabled bool) error {
	_, err := r.Pool.Exec(ctx, `UPDATE users SET mfa_totp_secret=$2, mfa_enabled=$3, updated_at=now() WHERE id=$1`,
		userID, secret, enabled)
	return err
}

func (r *Repo) SetGPC(ctx context.Context, userID int64, v bool) error {
	_, err := r.Pool.Exec(ctx, `UPDATE users SET gpc_optout=$2, updated_at=now() WHERE id=$1`, userID, v)
	return err
}

func (r *Repo) UpdateEmail(ctx context.Context, userID int64, email string) error {
	_, err := r.Pool.Exec(ctx, `UPDATE users SET email=$2, updated_at=now() WHERE id=$1`, userID, email)
	return mapErr(err)
}

func (r *Repo) Membership(ctx context.Context, wsID, userID int64) (Membership, error) {
	var m Membership
	err := r.Pool.QueryRow(ctx, `
		SELECT m.workspace_id, m.user_id, m.role::text, w.name, w.home_region
		FROM memberships m JOIN workspaces w ON w.id = m.workspace_id
		WHERE m.workspace_id=$1 AND m.user_id=$2`, wsID, userID).Scan(
		&m.WorkspaceID, &m.UserID, &m.Role, &m.WSName, &m.HomeRegion)
	return m, mapErr(err)
}

func (r *Repo) FirstMembership(ctx context.Context, userID int64) (Membership, error) {
	var m Membership
	err := r.Pool.QueryRow(ctx, `
		SELECT m.workspace_id, m.user_id, m.role::text, w.name, w.home_region
		FROM memberships m JOIN workspaces w ON w.id = m.workspace_id
		WHERE m.user_id=$1 ORDER BY m.created_at ASC LIMIT 1`, userID).Scan(
		&m.WorkspaceID, &m.UserID, &m.Role, &m.WSName, &m.HomeRegion)
	return m, mapErr(err)
}

func (r *Repo) ListMemberships(ctx context.Context, userID int64) ([]Membership, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT m.workspace_id, m.user_id, m.role::text, w.name, w.home_region
		FROM memberships m JOIN workspaces w ON w.id = m.workspace_id
		WHERE m.user_id=$1 ORDER BY w.name`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Membership
	for rows.Next() {
		var m Membership
		if err := rows.Scan(&m.WorkspaceID, &m.UserID, &m.Role, &m.WSName, &m.HomeRegion); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Repo) ListWorkspaceMembers(ctx context.Context, wsID int64) ([]Membership, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT workspace_id, user_id, role::text FROM memberships WHERE workspace_id=$1`, wsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Membership
	for rows.Next() {
		var m Membership
		if err := rows.Scan(&m.WorkspaceID, &m.UserID, &m.Role); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (r *Repo) UpsertMember(ctx context.Context, wsID, userID int64, role string) error {
	_, err := r.Pool.Exec(ctx, `
		INSERT INTO memberships (workspace_id, user_id, role) VALUES ($1,$2,$3::member_role)
		ON CONFLICT (workspace_id, user_id) DO UPDATE SET role = excluded.role`, wsID, userID, role)
	return err
}

type APIKeyRow struct {
	ID, WorkspaceID int64
	Name, Prefix    string
	Hash            []byte
	Scopes          []string
	RevokedAt       *time.Time
	ExpiresAt       *time.Time
}

func (r *Repo) InsertAPIKey(ctx context.Context, wsID, creator int64, name, prefix string, hash []byte, scopes []string) (int64, error) {
	var id int64
	err := r.Pool.QueryRow(ctx, `
		INSERT INTO api_keys (workspace_id, name, key_prefix, key_hash, scopes, created_by)
		VALUES ($1,$2,$3,$4, COALESCE($5::STRING[], ARRAY[]::STRING[]), $6) RETURNING id`,
		wsID, name, prefix, hash, scopes, creator).Scan(&id)
	return id, mapErr(err)
}

func (r *Repo) APIKeyByHash(ctx context.Context, hash []byte) (APIKeyRow, error) {
	var k APIKeyRow
	err := r.Pool.QueryRow(ctx, `
		SELECT id, workspace_id, name, key_prefix, key_hash, scopes, revoked_at, expires_at
		FROM api_keys WHERE key_hash = $1`, hash).Scan(
		&k.ID, &k.WorkspaceID, &k.Name, &k.Prefix, &k.Hash, &k.Scopes, &k.RevokedAt, &k.ExpiresAt)
	return k, mapErr(err)
}

func (r *Repo) ListAPIKeys(ctx context.Context, wsID int64) ([]APIKeyRow, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, workspace_id, name, key_prefix, key_hash, scopes, revoked_at, expires_at
		FROM api_keys WHERE workspace_id=$1 ORDER BY created_at DESC`, wsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []APIKeyRow
	for rows.Next() {
		var k APIKeyRow
		if err := rows.Scan(&k.ID, &k.WorkspaceID, &k.Name, &k.Prefix, &k.Hash, &k.Scopes, &k.RevokedAt, &k.ExpiresAt); err != nil {
			return nil, err
		}
		k.Hash = nil // never return hash to callers
		out = append(out, k)
	}
	return out, rows.Err()
}

func (r *Repo) RevokeAPIKey(ctx context.Context, wsID, id int64) error {
	tag, err := r.Pool.Exec(ctx, `UPDATE api_keys SET revoked_at=now() WHERE id=$1 AND workspace_id=$2 AND revoked_at IS NULL`, id, wsID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) TouchAPIKey(ctx context.Context, id int64) {
	_, _ = r.Pool.Exec(ctx, `UPDATE api_keys SET last_used_at=now() WHERE id=$1`, id)
}

type Link struct {
	DomainID, WorkspaceID, CreatedBy int64
	Code, Destination, Status, Verdict string
	RedirectType                     int16
	PasswordProtected                bool
	Title, Description, Image        *string
	ExpiresAt                        *time.Time
	ClickLimit                       *int64
	ClickCount                       int64
	CreatedAt                        time.Time
}

func (r *Repo) InsertLink(ctx context.Context, l Link) error {
	_, err := r.Pool.Exec(ctx, `
		INSERT INTO links (domain_id, code, workspace_id, destination_url, status, redirect_type,
		                   scan_verdict, is_password_protected, title_override, description_override,
		                   image_override, expires_at, click_limit, created_by)
		VALUES ($1,$2,$3,$4,$5::link_status,$6,$7::scan_verdict,$8,$9,$10,$11,$12,$13,$14)`,
		l.DomainID, l.Code, l.WorkspaceID, l.Destination, l.Status, l.RedirectType,
		l.Verdict, l.PasswordProtected, l.Title, l.Description, l.Image, l.ExpiresAt, l.ClickLimit, l.CreatedBy)
	return mapErr(err)
}

func (r *Repo) SetLinkPassword(ctx context.Context, domainID int64, code, phc string) error {
	_, err := r.Pool.Exec(ctx, `
		INSERT INTO link_passwords (domain_id, code, password_hash, algo_params)
		VALUES ($1,$2,$3,'{"m":65536,"t":3,"p":2}'::jsonb)
		ON CONFLICT (domain_id, code) DO UPDATE SET password_hash=excluded.password_hash, updated_at=now()`,
		domainID, code, phc)
	return err
}

func (r *Repo) GetLink(ctx context.Context, wsID, domainID int64, code string) (Link, error) {
	var l Link
	err := r.Pool.QueryRow(ctx, `
		SELECT domain_id, code, workspace_id, destination_url, status::text, redirect_type,
		       scan_verdict::text, is_password_protected, title_override, description_override,
		       image_override, expires_at, click_limit, click_count, created_at, COALESCE(created_by,0)
		FROM links WHERE workspace_id=$1 AND domain_id=$2 AND code=$3 AND deleted_at IS NULL`,
		wsID, domainID, code).Scan(
		&l.DomainID, &l.Code, &l.WorkspaceID, &l.Destination, &l.Status, &l.RedirectType,
		&l.Verdict, &l.PasswordProtected, &l.Title, &l.Description, &l.Image, &l.ExpiresAt,
		&l.ClickLimit, &l.ClickCount, &l.CreatedAt, &l.CreatedBy)
	return l, mapErr(err)
}

func (r *Repo) ListLinks(ctx context.Context, wsID int64, limit int) ([]Link, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	rows, err := r.Pool.Query(ctx, `
		SELECT domain_id, code, workspace_id, destination_url, status::text, redirect_type,
		       scan_verdict::text, is_password_protected, title_override, description_override,
		       image_override, expires_at, click_limit, click_count, created_at, COALESCE(created_by,0)
		FROM links WHERE workspace_id=$1 AND deleted_at IS NULL
		ORDER BY created_at DESC LIMIT $2`, wsID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Link
	for rows.Next() {
		var l Link
		if err := rows.Scan(&l.DomainID, &l.Code, &l.WorkspaceID, &l.Destination, &l.Status, &l.RedirectType,
			&l.Verdict, &l.PasswordProtected, &l.Title, &l.Description, &l.Image, &l.ExpiresAt,
			&l.ClickLimit, &l.ClickCount, &l.CreatedAt, &l.CreatedBy); err != nil {
			return nil, err
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

func (r *Repo) UpdateLink(ctx context.Context, wsID, domainID int64, code, dest, status string, rtype int16) error {
	tag, err := r.Pool.Exec(ctx, `
		UPDATE links SET destination_url=COALESCE(NULLIF($4,''), destination_url),
		                 status=COALESCE(NULLIF($5,'')::link_status, status),
		                 redirect_type=CASE WHEN $6=0 THEN redirect_type ELSE $6 END,
		                 updated_at=now()
		WHERE workspace_id=$1 AND domain_id=$2 AND code=$3 AND deleted_at IS NULL`,
		wsID, domainID, code, dest, status, rtype)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) SoftDeleteLink(ctx context.Context, wsID, domainID int64, code string) error {
	tag, err := r.Pool.Exec(ctx, `
		UPDATE links SET deleted_at=now(), status='disabled', updated_at=now()
		WHERE workspace_id=$1 AND domain_id=$2 AND code=$3 AND deleted_at IS NULL`, wsID, domainID, code)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) DomainOwned(ctx context.Context, wsID, domainID int64) (hostname string, ok bool, err error) {
	err = r.Pool.QueryRow(ctx, `SELECT hostname FROM domains WHERE id=$1 AND workspace_id=$2`, domainID, wsID).Scan(&hostname)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", false, nil
	}
	return hostname, err == nil, err
}

func (r *Repo) DomainByHost(ctx context.Context, host string) (id, wsID int64, state string, err error) {
	err = r.Pool.QueryRow(ctx, `SELECT id, workspace_id, state::text FROM domains WHERE hostname=$1`, host).Scan(&id, &wsID, &state)
	return id, wsID, state, mapErr(err)
}

type Domain struct {
	ID, WorkspaceID                                          int64
	Hostname, State, VerifyToken, CNAME                      string
	IsApex, HSTS                                             bool
	RootRedirect, Custom404                                  *string
	VerifiedAt, LastChecked, NextCheck                       *time.Time
}

func (r *Repo) InsertDomain(ctx context.Context, wsID int64, host, token, cname string, apex bool) (int64, error) {
	var id int64
	err := r.Pool.QueryRow(ctx, `
		INSERT INTO domains (workspace_id, hostname, verify_token, cname_target, is_apex, next_check_at)
		VALUES ($1,$2,$3,$4,$5, now()) RETURNING id`, wsID, host, token, cname, apex).Scan(&id)
	return id, mapErr(err)
}

func (r *Repo) GetDomain(ctx context.Context, wsID, id int64) (Domain, error) {
	var d Domain
	err := r.Pool.QueryRow(ctx, `
		SELECT id, workspace_id, hostname, state::text, verify_token, cname_target, is_apex,
		       root_redirect_url, custom_404_url, hsts_enabled, verified_at, last_checked_at, next_check_at
		FROM domains WHERE id=$1 AND workspace_id=$2`, id, wsID).Scan(
		&d.ID, &d.WorkspaceID, &d.Hostname, &d.State, &d.VerifyToken, &d.CNAME, &d.IsApex,
		&d.RootRedirect, &d.Custom404, &d.HSTS, &d.VerifiedAt, &d.LastChecked, &d.NextCheck)
	return d, mapErr(err)
}

func (r *Repo) ListDomains(ctx context.Context, wsID int64) ([]Domain, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, workspace_id, hostname, state::text, verify_token, cname_target, is_apex,
		       root_redirect_url, custom_404_url, hsts_enabled, verified_at, last_checked_at, next_check_at
		FROM domains WHERE workspace_id=$1 ORDER BY created_at DESC`, wsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Domain
	for rows.Next() {
		var d Domain
		if err := rows.Scan(&d.ID, &d.WorkspaceID, &d.Hostname, &d.State, &d.VerifyToken, &d.CNAME, &d.IsApex,
			&d.RootRedirect, &d.Custom404, &d.HSTS, &d.VerifiedAt, &d.LastChecked, &d.NextCheck); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func (r *Repo) UpdateDomainSettings(ctx context.Context, wsID, id int64, root, c404 *string, hsts bool) error {
	tag, err := r.Pool.Exec(ctx, `
		UPDATE domains SET root_redirect_url=$3, custom_404_url=$4, hsts_enabled=$5, updated_at=now()
		WHERE id=$1 AND workspace_id=$2`, id, wsID, root, c404, hsts)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) ReservedSlugs(ctx context.Context) (map[string]struct{}, error) {
	rows, err := r.Pool.Query(ctx, `SELECT slug FROM reserved_slugs`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	m := map[string]struct{}{}
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			return nil, err
		}
		m[s] = struct{}{}
	}
	return m, rows.Err()
}

func (r *Repo) IdemGet(ctx context.Context, wsID int64, key string) (status int, linkRef string, ok bool, err error) {
	err = r.Pool.QueryRow(ctx, `
		SELECT COALESCE(response_status,0), COALESCE(link_ref,'')
		FROM idempotency_keys WHERE workspace_id=$1 AND idempotency_key=$2`, wsID, key).Scan(&status, &linkRef)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", false, nil
	}
	return status, linkRef, err == nil, err
}

func (r *Repo) IdemPut(ctx context.Context, wsID int64, key, ref string, status int, reqHash []byte) error {
	_, err := r.Pool.Exec(ctx, `
		INSERT INTO idempotency_keys (workspace_id, idempotency_key, request_hash, response_status, link_ref)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (workspace_id, idempotency_key) DO NOTHING`, wsID, key, reqHash, status, ref)
	return err
}

func (r *Repo) InsertQR(ctx context.Context, domainID int64, code, format, ec string, size, margin int, fg, bg string, hasLogo bool, styleHash, objectKey string) (int64, error) {
	var id int64
	err := r.Pool.QueryRow(ctx, `
		INSERT INTO qr_codes (domain_id, code, format, size, margin, ec_level, fg_color, bg_color, has_logo, style_hash, object_key)
		VALUES ($1,$2,$3::qr_format,$4,$5,$6,$7,$8,$9,$10,$11)
		ON CONFLICT (domain_id, code, format, style_hash) DO UPDATE SET object_key=excluded.object_key, updated_at=now()
		RETURNING id`, domainID, code, format, size, margin, ec, fg, bg, hasLogo, []byte(styleHash), objectKey).Scan(&id)
	return id, err
}

func (r *Repo) EnqueueWebhook(ctx context.Context, endpointID, wsID int64, event string, payload any) error {
	b, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	_, err = r.Pool.Exec(ctx, `
		INSERT INTO webhook_deliveries (endpoint_id, workspace_id, event, payload) VALUES ($1,$2,$3,$4)`,
		endpointID, wsID, event, b)
	return err
}

func (r *Repo) ActiveWebhookIDs(ctx context.Context, wsID int64, event string) ([]int64, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id FROM webhook_endpoints WHERE workspace_id=$1 AND active AND $2 = ANY (events)`, wsID, event)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	return ids, rows.Err()
}

type WebhookEP struct {
	ID     int64
	URL    string
	Events []string
	Active bool
}

func (r *Repo) InsertWebhook(ctx context.Context, wsID int64, rawURL string, secret []byte, events []string) (int64, error) {
	var id int64
	err := r.Pool.QueryRow(ctx, `
		INSERT INTO webhook_endpoints (workspace_id, url, secret, events) VALUES ($1,$2,$3,$4) RETURNING id`,
		wsID, rawURL, secret, events).Scan(&id)
	return id, err
}

func (r *Repo) ListWebhooks(ctx context.Context, wsID int64) ([]WebhookEP, error) {
	rows, err := r.Pool.Query(ctx, `SELECT id, url, events, active FROM webhook_endpoints WHERE workspace_id=$1`, wsID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []WebhookEP
	for rows.Next() {
		var w WebhookEP
		if err := rows.Scan(&w.ID, &w.URL, &w.Events, &w.Active); err != nil {
			return nil, err
		}
		out = append(out, w)
	}
	return out, rows.Err()
}

func (r *Repo) DeleteWebhook(ctx context.Context, wsID, id int64) error {
	tag, err := r.Pool.Exec(ctx, `DELETE FROM webhook_endpoints WHERE id=$1 AND workspace_id=$2`, id, wsID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

func (r *Repo) Audit(ctx context.Context, wsID, userID, keyID int64, action, targetType, targetID string) {
	_, _ = r.Pool.Exec(ctx, `
		INSERT INTO audit_log (workspace_id, actor_user_id, actor_api_key_id, action, target_type, target_id)
		VALUES ($1,$2,$3,$4,$5,$6)`, wsID, nptr(userID), nptr(keyID), action, targetType, targetID)
}

func nptr(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

type DSAR struct {
	ID           int64
	Type, Status, Regime, Email string
	SLADue       time.Time
	RequestedAt  time.Time
}

func (r *Repo) InsertDSAR(ctx context.Context, wsID, userID int64, email, typ, regime string, sla time.Time) (int64, error) {
	var id int64
	err := r.Pool.QueryRow(ctx, `
		INSERT INTO data_subject_requests (workspace_id, subject_user_id, subject_email, type, regime, sla_due_at)
		VALUES ($1,$2,$3,$4::dsar_type,$5,$6) RETURNING id`, wsID, userID, email, typ, regime, sla).Scan(&id)
	return id, err
}

func (r *Repo) GetDSAR(ctx context.Context, id, userID int64) (DSAR, error) {
	var d DSAR
	err := r.Pool.QueryRow(ctx, `
		SELECT id, type::text, status::text, regime, subject_email, sla_due_at, requested_at
		FROM data_subject_requests WHERE id=$1 AND subject_user_id=$2`, id, userID).Scan(
		&d.ID, &d.Type, &d.Status, &d.Regime, &d.Email, &d.SLADue, &d.RequestedAt)
	return d, mapErr(err)
}

func (r *Repo) ListDSAR(ctx context.Context, userID int64) ([]DSAR, error) {
	rows, err := r.Pool.Query(ctx, `
		SELECT id, type::text, status::text, regime, subject_email, sla_due_at, requested_at
		FROM data_subject_requests WHERE subject_user_id=$1 ORDER BY requested_at DESC`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []DSAR
	for rows.Next() {
		var d DSAR
		if err := rows.Scan(&d.ID, &d.Type, &d.Status, &d.Regime, &d.Email, &d.SLADue, &d.RequestedAt); err != nil {
			return nil, err
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

func mapErr(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	// Unique violation (Postgres 23505 / CRDB).
	if isUnique(err) {
		return ErrConflict
	}
	return err
}

func isUnique(err error) bool {
	s := err.Error()
	return contains(s, "duplicate") || contains(s, "unique") || contains(s, "23505")
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (indexOf(s, sub) >= 0)
}
func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}

func slugify(s string) string {
	b := make([]byte, 0, len(s))
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c >= 'A' && c <= 'Z' {
			c += 32
		}
		if c >= 'a' && c <= 'z' || c >= '0' && c <= '9' {
			b = append(b, c)
		} else if len(b) > 0 && b[len(b)-1] != '-' {
			b = append(b, '-')
		}
	}
	if len(b) == 0 {
		return "ws"
	}
	if len(b) > 24 {
		b = b[:24]
	}
	return string(b)
}

func rand4() string {
	return time.Now().Format("150405")
}
