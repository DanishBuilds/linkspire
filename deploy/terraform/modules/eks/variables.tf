variable "name" { type = string }
variable "kubernetes_version" { type = string }
variable "vpc_id" { type = string }
variable "private_subnet_ids" { type = list(string) }
variable "instance_types" { type = list(string) }
variable "node_min" { type = number }
variable "node_max" { type = number }
variable "node_desired" { type = number }
