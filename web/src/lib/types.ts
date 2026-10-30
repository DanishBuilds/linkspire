// src/lib/types.ts — clean TypeScript models for the dashboard.
// The Phase 4 API is inconsistent: handler-level DTOs use snake_case, while
// endpoints that serialize Go repo structs emit PascalCase (no json tags).
// All of that is normalized away in lib/api.ts; the UI only sees these.

export type Role = 'owner' | 'admin' | 'member' | 'viewer';

export const ROLE_RANK: Record<Role, number> = { viewer: 1, member: 2, admin: 3, owner: 4 };

export interface Membership {
  workspaceId: number;
  userId: number;
  role: Role;
  workspaceName: string;
  homeRegion: string;
}

export interface Me {
  id: number;
  email: string;
  mfaEnabled: boolean;
  gpcOptOut: boolean;
  homeRegion: string;
  isStaff: boolean;
  workspaces: Membership[];
}

export type LinkStatus = 'active' | 'password_protected' | 'disabled' | 'expired' | string;
export type ScanVerdict = 'pending' | 'clean' | 'suspicious' | 'malicious' | string;

export interface Link {
  domainId: number;
  code: string;
  destination: string;
  status: LinkStatus;
  scanVerdict: ScanVerdict;
  redirectType: number;
  passwordProtected: boolean;
  shortUrl: string;
  expiresAt: string | null;
  clickLimit: number | null;
  createdAt: string;
}

export interface DomainRecord {
  id: number;
  hostname: string;
  state: string; // pending | verified | active | failed
  verifyToken: string;
  cnameTarget: string;
  isApex: boolean;
  hsts: boolean;
  rootRedirect: string | null;
  custom404: string | null;
  verifiedAt: string | null;
  lastChecked: string | null;
  nextCheck: string | null;
}

export interface NewDomainInstructions {
  id: number;
  hostname: string;
  state: string;
  cnameTarget: string;
  txtName: string;
  txtValue: string;
  apex: boolean;
  apexNote?: string;
}

export interface ApiKey {
  id: number;
  name: string;
  prefix: string;
  scopes: string[];
  revokedAt: string | null;
  expiresAt: string | null;
}

export interface Webhook {
  id: number;
  url: string;
  events: string[];
  active: boolean;
}

export type DsarType =
  | 'access' | 'rectification' | 'erasure' | 'restriction' | 'objection' | 'do_not_sell';

export interface DsarRequest {
  id: number;
  type: DsarType | string;
  status: string;
  regime: string;
  email: string;
  slaDue: string;
  requestedAt: string;
}

export interface AnalyticsRow {
  date: string;
  clicks: number;
  uniqueVisitors: number;
  countryCode?: string;
  deviceClass?: string;
  browserFamily?: string;
  osFamily?: string;
  referrerDomain?: string;
  source?: string;
}

export interface AnalyticsResult {
  configured: boolean;
  status: 'ok' | 'unconfigured' | 'error';
  rows: AnalyticsRow[];
  note?: string;
  privacy?: string;
}

export interface TotpSetup {
  secret: string;
  otpauthUrl: string;
}

export interface AbuseReport {
  id: number;
  shortUrl: string;
  reason: string;
  status: string;
  reporterEmail: string | null;
  createdAt: string;
  resolvedAt: string | null;
}

export interface QrAsset {
  id: number;
  format: string;
  ecLevel: string;
  size: number;
  status: string;
  url: string;
  createdAt: string;
}

export interface Passkey {
  id: number;
  label: string | null;
}

