variable "project" {
  type    = string
  default = "linkspire"
}
variable "region" {
  type    = string
  default = "us-east-1"
}
variable "account_suffix" {
  description = "A unique suffix (for example the AWS account id) to make the bucket name globally unique."
  type        = string
}
