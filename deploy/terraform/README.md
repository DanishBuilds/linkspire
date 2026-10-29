<!-- deploy/terraform/README.md -->
# Linkspire infrastructure (Terraform, AWS)

Provisions the cloud substrate the Helm chart deploys onto:

| Module | Resources |
|--------|-----------|
| `network` | VPC, public/private subnets across AZs, NAT gateways, VPC flow logs |
| `eks` | EKS control plane (KMS-encrypted secrets, control-plane logs), IRSA OIDC provider, managed node group |
| `ecr` | Immutable, scan-on-push repositories for `redirect`, `api`, `workers`, `web` with a 30-image lifecycle policy |
| `data` | ElastiCache Redis (multi-AZ, encrypted in transit and at rest) and MSK Kafka (TLS, KMS) reachable only from the cluster |

## Not provisioned here (by design)

- **CockroachDB** (RPO=0 distributed SQL, ADR-002): run as **CockroachDB Cloud** (or self-managed)
  and VPC-peer it. Inject `PG_DSN` as a Kubernetes secret.
- **ClickHouse** (analytics, ADR-005): run as **ClickHouse Cloud** and inject `CLICKHOUSE_URL`.

Keeping the stateful, RPO-sensitive stores out of this Terraform avoids a `terraform destroy`
ever touching committed link data.

## Usage

```bash
cd deploy/terraform
terraform init
terraform plan  -var-file=terraform.tfvars    # copy terraform.tfvars.example first
terraform apply -var-file=terraform.tfvars

aws eks update-kubeconfig --region <region> --name <cluster>   # see outputs
```

Then build and push images to the ECR URLs (see `outputs "ecr_repository_urls"`), set
`global.image.registry` in the Helm values accordingly, and `helm upgrade --install`.

## Notes

- State is local by default. For real use, configure a remote backend (S3 + DynamoDB lock)
  in a `backend.tf`; it is intentionally omitted so this validates without cloud credentials.
- `terraform validate` and `terraform fmt -check` should pass; run them in CI (see
  `.github/workflows/`).
