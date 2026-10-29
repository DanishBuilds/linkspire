# deploy/terraform/backend.tf
# Remote state in S3 with a DynamoDB lock table. Partial configuration: create
# the bucket and lock table once with ./bootstrap, then init with -backend-config:
#
#   terraform init \
#     -backend-config="bucket=linkspire-tfstate-<account-id>" \
#     -backend-config="key=linkspire/prod.tfstate" \
#     -backend-config="region=us-east-1" \
#     -backend-config="dynamodb_table=linkspire-tflock" \
#     -backend-config="encrypt=true"
#
# Kept empty so `terraform validate` works without credentials or a live bucket.
terraform {
  backend "s3" {}
}
