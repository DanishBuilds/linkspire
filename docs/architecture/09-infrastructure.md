<!-- docs/architecture/09-infrastructure.md -->
# Phase 6 — Infrastructure, delivery, and tests

Everything needed to build, ship, run, load-test, and security-test the platform:
containers (`deploy/docker`), Kubernetes packaging (`deploy/helm`), cloud substrate
(`deploy/terraform`), load tests (`test/load`), a security suite (`test/security`), and
CI/CD (`.github/workflows`).

## 1. Containers

Each service builds from the repo root with a multi-stage, distroless, non-root image:

| Image | Base | Ports | Notes |
|-------|------|-------|-------|
| `redirect` | `distroless/static:nonroot` | 8080 (public), 9090 (admin) | static CGO-free Go binary |
| `api` | `distroless/static:nonroot` | 8081, 9091 (metrics) | static Go binary |
| `workers` | `distroless/static:nonroot` | 9092 (health/metrics) | role via `WORKER_ROLE` |
| `web` | `distroless/nodejs22:nonroot` | 3000 | Next.js standalone output |

All run as uid 65532, carry OCI labels, and use BuildKit cache mounts for Go modules and the
build cache. Phase 6 added a small health and metrics server to the workers process (`/healthz`,
`/readyz` with a DB ping, `/metrics`) so it is probeable in Kubernetes, and enabled Next.js
`output: "standalone"` for a minimal web image.

`deploy/docker/compose.yaml` brings up the whole stack locally: CockroachDB, Redis, Redpanda
(Kafka), ClickHouse (schema auto-loaded), a one-shot golang-migrate job, and the four services
wired together. `docker compose -f deploy/docker/compose.yaml up --build` from the repo root.

## 2. Kubernetes (Helm)

`deploy/helm/linkspire` is a single values-driven chart. `.Values.services` is a map; each entry
renders a Deployment, a Service (when it serves traffic), an HPA (when autoscaling is on), and a
PodDisruptionBudget. Defaults encode the capacity plan: redirect autoscales 3 to 50 on 60% CPU
with a 50% minAvailable PDB; the API 2 to 10; the dashboard 2 to 8; workers run as a single
replica with no Service.

Security defaults apply to every pod: `runAsNonRoot`, `readOnlyRootFilesystem` (with emptyDir
mounts for `/tmp` and the Next cache), all capabilities dropped, `seccompProfile:
RuntimeDefault`, and a ServiceAccount with no auto-mounted token. Go services load non-secret
config from a ConfigMap and secrets from a Secret (or, in production, an external secret
manager). Liveness and readiness probes hit `/healthz` and `/readyz`; an optional ServiceMonitor
wires Prometheus scraping.

## 3. Cloud substrate (Terraform, AWS)

`deploy/terraform` is modular: `network` (VPC, public/private subnets, NAT, flow logs), `eks`
(control plane with KMS-encrypted secrets and control-plane logs, an IRSA OIDC provider, and a
managed node group), `ecr` (immutable, scan-on-push repositories with a lifecycle policy), and
`data` (ElastiCache Redis and MSK Kafka, both encrypted and reachable only from the cluster
security group).

CockroachDB and ClickHouse are deliberately left out of Terraform and consumed as managed
services (CockroachDB Cloud, ClickHouse Cloud) peered to the VPC, so the RPO=0 link store is
never at the mercy of a `terraform destroy`. Their connection strings arrive as Kubernetes
secrets.

## 4. Load tests (k6)

`test/load` encodes SLOs as thresholds: `redirect_hotpath.js` ramps arrival rate and fails the
run if redirect p99 exceeds 30 ms or errors exceed 0.1%; `api_load.js` holds the management API
to p95 under 200 ms; `smoke.js` checks liveness. Summaries export to JSON for archiving under
`test/load/results/`.

## 5. Security tests

`test/security` is tagged `security` so it never runs in the default build. `ssrf_test.go`
asserts the IP policy blocks metadata, loopback, RFC1918, unspecified, and IPv4-mapped ranges and
denies non-http schemes. `decoder_fuzz_test.go` fuzzes the Base62 decoder (never panics) and the
Feistel round-trip (a bijection). `authz_integration_test.go` and `ratelimit_integration_test.go`
run against a live stack (gated on `LINKSPIRE_API_URL` / `REDIRECT_URL`) to prove tenant
isolation, authentication, malicious-destination rejection, and that spoofing `X-Forwarded-For`
does not bypass the rate limiter. Deeper unit coverage lives beside the code in `pkg/*/*_test.go`.

## 6. CI/CD (GitHub Actions)

| Workflow | What it does |
|----------|--------------|
| `ci.yml` | Go vet/test (race)/build + gofmt; dashboard lint/typecheck/build |
| `images.yml` | Build each image, Trivy scan (SARIF), Syft SBOM (SPDX), push to GHCR |
| `security.yml` | CodeQL (Go + JS/TS), govulncheck, gosec, checkov on `deploy/`, gitleaks |
| `deploy.yml` | Canary a reduced fleet, verify rollout, promote with Helm, clean up |

## 7. Verified this phase

Full `helm`, `terraform`, `docker`, and `k6` binaries are not available in the build sandbox
(locked-down egress), so machine validation used equivalent checks:

| Check | Result |
|-------|--------|
| `yamllint` (compose, chart metadata, all workflows) | clean |
| Terraform HCL parse (16 files) | clean |
| Helm template grammar parse (10 templates) | clean |
| k6 scripts (`node --check`) | clean |
| `gofmt` (workers change + security suite) | clean |

Run `helm lint` / `helm template`, `terraform init && terraform validate`, `docker build`, and
the k6 scripts in your own environment (and in CI) for the full checks; `ci.yml` and
`security.yml` do exactly this on every push.

## 8. Assumptions and open questions

- **Assumptions:** AWS is the target cloud; Redis and Kafka are AWS-managed while the SQL and
  analytics stores are external managed services; images publish to GHCR; the `deploy.yml` canary
  expects a `KUBECONFIG_B64` secret and a `production` environment with approvals.
- **Open questions:** (1) a remote Terraform backend (S3 + DynamoDB lock) is documented but left
  out so the config validates without credentials. (2) Progressive delivery could move from the
  manual Helm canary to Argo Rollouts or Flagger. (3) MSK vs a Redpanda operator in-cluster is a
  cost/ops tradeoff worth revisiting against the 100M events/day target.
