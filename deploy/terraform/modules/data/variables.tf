variable "name" { type = string }
variable "vpc_id" { type = string }
variable "private_subnet_ids" { type = list(string) }
variable "node_security_group_id" { type = string }
variable "redis_node_type" {
  type    = string
  default = "cache.m6g.large"
}
variable "kafka_broker_count" {
  type    = number
  default = 3
}
