<!-- docs/runbooks/dsar.md -->
# Runbook: data subject request (DSAR) fulfillment

**Severity:** SEV3 operationally, but **time-bound**: 30 days (GDPR) or 45 days (CCPA/CPRA) from a
verified request. Missing the window is a compliance failure.

## Background

Requests are filed from the dashboard (Settings, Privacy) or by email and recorded in
`data_subject_requests` with a type, regime, and SLA due date. The DSAR worker processes open
requests. Types: access, rectification, erasure, restriction, objection, do-not-sell.

## Procedure

1. **Verify identity.** Confirm the requester controls the account or email in question before
   acting. Do not disclose or delete on an unverified request.
2. **Classify.** Confirm the type and regime; the due date is set from the regime (30 vs 45 days).
   Prioritize by due date.
3. **Fulfill by type:**
   - **Access / portability:** export the subject's account data (profile, workspaces, links, keys
     metadata) in a commonly used format. Note that click analytics are already non-identifying
     aggregates with no per-subject rows, so there is nothing subject-specific to export from them.
   - **Rectification:** correct the stored field(s); most are self-service in the dashboard.
   - **Erasure:** delete account, workspace, link, and key data within the window. Analytics need no
     action: they contain no identifier for the subject and expire on their TTLs (raw 30 days,
     aggregates ~25 months). Record the deletion.
   - **Restriction / objection:** flag the account so processing beyond storage is paused as
     required.
   - **Do-not-sell / GPC:** set the account's opt-out flag. Note we do not sell or share data, so
     this is a preference record, not a change in data flows.
4. **Close out.** Mark the request completed with a timestamp; keep the record to evidence
   compliance, then let it expire on schedule.

## Verify

- The data is actually gone or exported as applicable (spot-check the relevant tables), the request
  status is `completed`, and completion is within the SLA.

## Escalate

- If a request cannot be completed in time (for example a legal hold or a technical blocker), inform
  the requester before the deadline with the lawful reason and the extension, and consult counsel.

## Follow up

- Track SLA adherence. If manual steps recur, automate them in the DSAR worker. Never satisfy a
  request by logging or exporting data the platform deliberately does not hold (raw IPs, user
  agents); their absence is the correct answer.
