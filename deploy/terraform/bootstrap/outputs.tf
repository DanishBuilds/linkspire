output "state_bucket" { value = aws_s3_bucket.state.id }
output "lock_table" { value = aws_dynamodb_table.lock.name }
output "init_command" {
  value = "terraform init -backend-config=\"bucket=${aws_s3_bucket.state.id}\" -backend-config=\"key=${var.project}/prod.tfstate\" -backend-config=\"region=${var.region}\" -backend-config=\"dynamodb_table=${aws_dynamodb_table.lock.name}\" -backend-config=\"encrypt=true\""
}
