// src/lib/api.ts — server-side client for the Phase 4 management API.
// Responsibilities:
//   * attach the JWT from the httpOnly session cookie (never exposed to browser)
//   * normalize the API's mixed snake_case / PascalCase JSON into clean models
//   * surface RFC 9457 problem+json as a typed ApiError
import { env } from './env';
import { getToken, getWorkspaceId } from './session';
import type {
  Me, Membership, Link, DomainRecord, NewDomainInstructions, ApiKey, Webhook,
  DsarRequest, AnalyticsResult, AnalyticsRow, Role, TotpSetup, AbuseReport, QrAsset, Passkey,
} from './types';

type R = Record<string, any>;

export class ApiError extends Error {
  status: number;
  title: string;
  detail: string;
  constructor(status: number, title: string, detail: string) {
    super(detail || title || `HTTP ${status}`);
    this.name = 'ApiError';
    this.status = status;
    this.title = title;
    this.detail = detail;
  }
}

// first defined value among candidate keys (handles snake_case + PascalCase)
function pick(o: R, ...keys: string[]): any {
  for (const k of keys) {
    if (o != null && o[k] !== undefined) return o[k];
  }
  return undefined;
}
const asNum = (v: any): number => (typeof v === 'number' ? v : Number(v ?? 0)) || 0;
const asStr = (v: any): string => (v == null ? '' : String(v));
const asBool = (v: any): boolean => v === true || v === 'true';
const asStrOrNull = (v: any): string | null => (v == null || v === '' ? null : String(v));
const asArr = (v: any): string[] => (Array.isArray(v) ? v.map(String) : []);

interface Options {
  method?: string;
  body?: unknown;
  headers?: Record<string, string>;
  token?: string | null;      // explicit token (login flow passes none; most read cookie)
  auth?: boolean;             // default true: pull token from session cookie
}

async function request<T = any>(path: string, opts: Options = {}): Promise<T> {
  const { method = 'GET', body, headers = {}, auth = true } = opts;
  const h: Record<string, string> = { Accept: 'application/json', ...headers };
  if (body !== undefined) h['Content-Type'] = 'application/json';

  let token = opts.token ?? null;
  if (auth && token == null) token = await getToken();
  if (token) h['Authorization'] = `Bearer ${token}`;
  if (auth) {
    const ws = await getWorkspaceId();
    if (ws) h['X-Workspace-Id'] = ws;
  }

  let res: Response;
  try {
    res = await fetch(`${env.apiUrl}${path}`, {
      method,
      headers: h,
      body: body === undefined ? undefined : JSON.stringify(body),
      cache: 'no-store',
    });
  } catch (e) {
    throw new ApiError(0, 'Network error', `Could not reach the Linkspire API at ${env.apiUrl}.`);
  }

  if (res.status === 204) return undefined as T;

  const text = await res.text();
  let data: any = null;
  if (text) {
    try { data = JSON.parse(text); } catch { data = text; }
  }

  if (!res.ok) {
    if (data && typeof data === 'object') {
      throw new ApiError(res.status, asStr(data.title) || 'Request failed', asStr(data.detail) || asStr(data.note) || '');
    }
    throw new ApiError(res.status, 'Request failed', typeof data === 'string' ? data : '');
  }
  return data as T;
}

/* ---------- normalizers ---------- */
function normMembership(o: R): Membership {
  return {
    workspaceId: asNum(pick(o, 'workspace_id', 'WorkspaceID')),
    userId: asNum(pick(o, 'user_id', 'UserID')),
    role: (asStr(pick(o, 'role', 'Role')) || 'viewer') as Role,
    workspaceName: asStr(pick(o, 'workspace_name', 'WSName', 'name')),
    homeRegion: asStr(pick(o, 'home_region', 'HomeRegion')),
  };
}
function normLink(o: R): Link {
  return {
    domainId: asNum(pick(o, 'domain_id', 'DomainID')),
    code: asStr(pick(o, 'code', 'Code')),
    destination: asStr(pick(o, 'destination', 'Destination')),
    status: asStr(pick(o, 'status', 'Status')),
    scanVerdict: asStr(pick(o, 'scan_verdict', 'Verdict')) || 'pending',
    redirectType: asNum(pick(o, 'redirect_type', 'RedirectType')) || 302,
    passwordProtected: asBool(pick(o, 'password_protected', 'PasswordProtected')),
    shortUrl: asStr(pick(o, 'short_url', 'ShortURL')),
    expiresAt: asStrOrNull(pick(o, 'expires_at', 'ExpiresAt')),
    clickLimit: (() => { const v = pick(o, 'click_limit', 'ClickLimit'); return v == null ? null : asNum(v); })(),
    createdAt: asStr(pick(o, 'created_at', 'CreatedAt')),
  };
}
function normDomain(o: R): DomainRecord {
  return {
    id: asNum(pick(o, 'id', 'ID')),
    hostname: asStr(pick(o, 'hostname', 'Hostname')),
    state: asStr(pick(o, 'state', 'State')),
    verifyToken: asStr(pick(o, 'verify_token', 'VerifyToken')),
    cnameTarget: asStr(pick(o, 'cname_target', 'CNAME', 'cname')),
    isApex: asBool(pick(o, 'is_apex', 'IsApex', 'apex')),
    hsts: asBool(pick(o, 'hsts_enabled', 'HSTS')),
    rootRedirect: asStrOrNull(pick(o, 'root_redirect_url', 'RootRedirect')),
    custom404: asStrOrNull(pick(o, 'custom_404_url', 'Custom404')),
    verifiedAt: asStrOrNull(pick(o, 'verified_at', 'VerifiedAt')),
    lastChecked: asStrOrNull(pick(o, 'last_checked_at', 'LastChecked')),
    nextCheck: asStrOrNull(pick(o, 'next_check_at', 'NextCheck')),
  };
}
function normKey(o: R): ApiKey {
  return {
    id: asNum(pick(o, 'id', 'ID')),
    name: asStr(pick(o, 'name', 'Name')),
    prefix: asStr(pick(o, 'prefix', 'Prefix')),
    scopes: asArr(pick(o, 'scopes', 'Scopes')),
    revokedAt: asStrOrNull(pick(o, 'revoked_at', 'RevokedAt')),
    expiresAt: asStrOrNull(pick(o, 'expires_at', 'ExpiresAt')),
  };
}
function normWebhook(o: R): Webhook {
  return {
    id: asNum(pick(o, 'id', 'ID')),
    url: asStr(pick(o, 'url', 'URL')),
    events: asArr(pick(o, 'events', 'Events')),
    active: asBool(pick(o, 'active', 'Active')),
  };
}
function normDsar(o: R): DsarRequest {
  return {
    id: asNum(pick(o, 'id', 'ID')),
    type: asStr(pick(o, 'type', 'Type')),
    status: asStr(pick(o, 'status', 'Status')) || 'received',
    regime: asStr(pick(o, 'regime', 'Regime')),
    email: asStr(pick(o, 'email', 'Email')),
    slaDue: asStr(pick(o, 'sla_due_at', 'SLADue')),
    requestedAt: asStr(pick(o, 'requested_at', 'RequestedAt')),
  };
}
function listOf(res: any): R[] {
  const d = res?.data ?? res;
  return Array.isArray(d) ? d : [];
}

/* ---------- auth ---------- */
export interface LoginResult { token: string; userId: number; mfaRequired: boolean; }

export async function register(input: {
  email: string; password: string; workspace?: string; home_region?: 'us' | 'eu';
}): Promise<{ token: string; userId: number; workspaceId: number }> {
  const r = await request('/v1/auth/register', { method: 'POST', auth: false, body: input });
  return { token: asStr(r.token), userId: asNum(r.user_id), workspaceId: asNum(r.workspace_id) };
}

export async function login(input: { email: string; password: string; totp?: string }): Promise<LoginResult> {
  const r = await request('/v1/auth/login', { method: 'POST', auth: false, body: input });
  return { token: asStr(r.token), userId: asNum(r.user_id), mfaRequired: asBool(r.mfa_required) };
}

export async function me(): Promise<Me> {
  const r = await request('/v1/auth/me');
  return {
    id: asNum(r.id),
    email: asStr(r.email),
    mfaEnabled: asBool(r.mfa_enabled),
    gpcOptOut: asBool(r.gpc_optout),
    homeRegion: asStr(r.home_region),
    isStaff: asBool(r.is_staff),
    workspaces: Array.isArray(r.workspaces) ? r.workspaces.map(normMembership) : [],
  };
}

export async function totpSetup(): Promise<TotpSetup> {
  const r = await request('/v1/auth/mfa/totp/setup', { method: 'POST' });
  return { secret: asStr(r.secret), otpauthUrl: asStr(r.otpauth_url) };
}
export async function totpEnable(code: string): Promise<void> {
  await request('/v1/auth/mfa/totp/enable', { method: 'POST', body: { code } });
}

/* ---------- workspaces ---------- */
export async function workspaces(): Promise<Membership[]> {
  const r = await request('/v1/workspaces');
  return listOf(r).map(normMembership);
}

export async function listMembers(workspaceId: number): Promise<Membership[]> {
  const r = await request(`/v1/workspaces/${workspaceId}/members`);
  return listOf(r).map(normMembership);
}

export async function setMemberRole(workspaceId: number, userId: number, role: string): Promise<void> {
  await request(`/v1/workspaces/${workspaceId}/members/${userId}`, { method: 'PUT', body: { role } });
}

/* ---------- links ---------- */
export async function listLinks(limit = 100): Promise<Link[]> {
  const r = await request(`/v1/links?limit=${limit}`);
  return listOf(r).map(normLink);
}
export async function getLink(domainId: number, code: string): Promise<Link> {
  return normLink(await request(`/v1/links/${domainId}/${encodeURIComponent(code)}`));
}
export interface CreateLinkInput {
  destination: string;
  domain_id?: number;
  alias?: string;
  redirect_type?: number;
  password?: string;
  expires_at?: string;
  click_limit?: number;
  title?: string;
  description?: string;
  image?: string;
}
export async function createLink(input: CreateLinkInput, idempotencyKey?: string): Promise<Link> {
  const headers: Record<string, string> = {};
  if (idempotencyKey) headers['Idempotency-Key'] = idempotencyKey;
  return normLink(await request('/v1/links', { method: 'POST', body: input, headers }));
}
export async function bulkCreate(items: CreateLinkInput[]): Promise<{ ok: Link[]; errors: string[] }> {
  const r = await request('/v1/links/bulk', { method: 'POST', body: { items } });
  const out = listOf(r);
  const ok: Link[] = [];
  const errors: string[] = [];
  for (const item of out) {
    if (item && item.error) errors.push(asStr(item.error));
    else ok.push(normLink(item));
  }
  return { ok, errors };
}
export async function patchLink(
  domainId: number, code: string,
  patch: { destination?: string; status?: string; redirect_type?: number },
): Promise<Link> {
  return normLink(await request(`/v1/links/${domainId}/${encodeURIComponent(code)}`, { method: 'PATCH', body: patch }));
}
export async function deleteLink(domainId: number, code: string): Promise<void> {
  await request(`/v1/links/${domainId}/${encodeURIComponent(code)}`, { method: 'DELETE' });
}
export async function requestQR(
  domainId: number, code: string,
  body: { format?: 'png' | 'svg'; size?: number; ec_level?: string; fg_color?: string; bg_color?: string; logo?: boolean },
): Promise<{ id: number; status: string }> {
  const r = await request(`/v1/links/${domainId}/${encodeURIComponent(code)}/qr`, { method: 'POST', body });
  return { id: asNum(r.id), status: asStr(r.status) };
}

/* ---------- analytics ---------- */
export async function analytics(domainId: number, code: string): Promise<AnalyticsResult> {
  const r = await request(`/v1/analytics/links/${domainId}/${encodeURIComponent(code)}`);
  const rows: AnalyticsRow[] = listOf(r).map((o: R) => ({
    date: asStr(pick(o, 'date', 'event_date')),
    clicks: asNum(pick(o, 'clicks')),
    uniqueVisitors: asNum(pick(o, 'unique_visitors', 'uniques')),
    countryCode: asStr(pick(o, 'country_code')) || undefined,
    deviceClass: asStr(pick(o, 'device_class')) || undefined,
    browserFamily: asStr(pick(o, 'browser_family')) || undefined,
    osFamily: asStr(pick(o, 'os_family')) || undefined,
    referrerDomain: asStr(pick(o, 'referrer_domain')) || undefined,
    source: asStr(pick(o, 'source')) || undefined,
  }));
  return {
    configured: !(r && r.note && String(r.note).includes('not configured')),
    rows,
    note: r?.note ? asStr(r.note) : undefined,
    privacy: r?.privacy ? asStr(r.privacy) : undefined,
  };
}

/* ---------- domains ---------- */
export async function listDomains(): Promise<DomainRecord[]> {
  const r = await request('/v1/domains');
  return listOf(r).map(normDomain);
}
export async function getDomain(id: number): Promise<DomainRecord> {
  return normDomain(await request(`/v1/domains/${id}`));
}
export async function createDomain(hostname: string, apex: boolean): Promise<NewDomainInstructions> {
  const r = await request('/v1/domains', { method: 'POST', body: { hostname, apex } });
  return {
    id: asNum(r.id), hostname: asStr(r.hostname), state: asStr(r.state),
    cnameTarget: asStr(r.cname_target), txtName: asStr(r.txt_name), txtValue: asStr(r.txt_value),
    apex: asBool(r.apex), apexNote: r.apex_note ? asStr(r.apex_note) : undefined,
  };
}
export async function patchDomain(
  id: number, body: { root_redirect_url?: string; custom_404_url?: string; hsts_enabled?: boolean },
): Promise<DomainRecord> {
  return normDomain(await request(`/v1/domains/${id}`, { method: 'PATCH', body }));
}

/* ---------- api keys ---------- */
export async function listKeys(): Promise<ApiKey[]> {
  const r = await request('/v1/api-keys');
  return listOf(r).map(normKey);
}
export async function createKey(name: string, scopes: string[]): Promise<{ id: number; prefix: string; key: string }> {
  const r = await request('/v1/api-keys', { method: 'POST', body: { name, scopes } });
  return { id: asNum(r.id), prefix: asStr(r.prefix), key: asStr(r.key) };
}
export async function revokeKey(id: number): Promise<void> {
  await request(`/v1/api-keys/${id}`, { method: 'DELETE' });
}

/* ---------- webhooks ---------- */
export async function listWebhooks(): Promise<Webhook[]> {
  const r = await request('/v1/webhooks');
  return listOf(r).map(normWebhook);
}
export async function createWebhook(url: string, events: string[]): Promise<{ id: number; secretHex: string }> {
  const r = await request('/v1/webhooks', { method: 'POST', body: { url, events } });
  return { id: asNum(r.id), secretHex: asStr(r.secret_hex) };
}
export async function deleteWebhook(id: number): Promise<void> {
  await request(`/v1/webhooks/${id}`, { method: 'DELETE' });
}

/* ---------- privacy / DSAR ---------- */
export async function listDsar(): Promise<DsarRequest[]> {
  const r = await request('/v1/privacy/requests');
  return listOf(r).map(normDsar);
}
export async function createDsar(type: string, regime: string): Promise<{ id: number; slaDays: number }> {
  const r = await request('/v1/privacy/requests', { method: 'POST', body: { type, regime } });
  return { id: asNum(r.id), slaDays: asNum(r.sla_days) };
}
export async function doNotSell(): Promise<void> {
  await request('/v1/privacy/do-not-sell', { method: 'POST', body: {} });
}

/* ---------- public abuse report (no auth) ---------- */
export async function reportAbuse(shortUrl: string, reason: string): Promise<void> {
  await request('/v1/abuse/reports', { method: 'POST', auth: false, body: { short_url: shortUrl, reason } });
}

/* ---------- QR assets ---------- */
export async function listQR(domainId: number, code: string): Promise<QrAsset[]> {
  const r = await request(`/v1/links/${domainId}/${encodeURIComponent(code)}/qr`);
  return listOf(r).map((o: R) => ({
    id: asNum(pick(o, 'id')),
    format: asStr(pick(o, 'format')),
    ecLevel: asStr(pick(o, 'ec_level')),
    size: asNum(pick(o, 'size')),
    status: asStr(pick(o, 'status')),
    url: asStr(pick(o, 'url')),
    createdAt: asStr(pick(o, 'created_at')),
  }));
}

/* ---------- admin abuse queue ---------- */
export async function listAbuse(status?: string): Promise<AbuseReport[]> {
  const qs = status ? `?status=${encodeURIComponent(status)}` : '';
  const r = await request(`/v1/admin/abuse${qs}`);
  return listOf(r).map((o: R) => ({
    id: asNum(pick(o, 'id')),
    shortUrl: asStr(pick(o, 'short_url')),
    reason: asStr(pick(o, 'reason')),
    status: asStr(pick(o, 'status')),
    reporterEmail: asStrOrNull(pick(o, 'reporter_email')),
    createdAt: asStr(pick(o, 'created_at')),
    resolvedAt: asStrOrNull(pick(o, 'resolved_at')),
  }));
}
export async function resolveAbuse(id: number, status: string): Promise<void> {
  await request(`/v1/admin/abuse/${id}/resolve`, { method: 'POST', body: { status } });
}

/* ---------- webauthn passthrough (raw ceremony objects) ---------- */
export async function waRegisterBegin(): Promise<any> {
  return request('/v1/auth/webauthn/register/begin', { method: 'POST' });
}
export async function waRegisterFinish(credential: unknown, label: string): Promise<any> {
  return request(`/v1/auth/webauthn/register/finish?label=${encodeURIComponent(label)}`, { method: 'POST', body: credential });
}
export async function waLoginBegin(email: string): Promise<any> {
  return request('/v1/auth/webauthn/login/begin', { method: 'POST', auth: false, body: { email } });
}
export async function waLoginFinish(credential: unknown): Promise<{ token: string }> {
  const r = await request('/v1/auth/webauthn/login/finish', { method: 'POST', auth: false, body: credential });
  return { token: asStr(r.token) };
}
export async function waList(): Promise<Passkey[]> {
  const r = await request('/v1/auth/webauthn/credentials');
  return listOf(r).map((o: R) => ({ id: asNum(pick(o, 'id')), label: asStrOrNull(pick(o, 'label')) }));
}
export async function waDelete(id: number): Promise<void> {
  await request(`/v1/auth/webauthn/credentials/${id}`, { method: 'DELETE' });
}
