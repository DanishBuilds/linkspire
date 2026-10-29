# Root module: wires network -> eks + ecr + data.
# CockroachDB (RPO=0 distributed SQL) and ClickHouse are expected as managed
# external services (CockroachDB Cloud, ClickHouse Cloud) peered to this VPC;
# their connection strings are injected as Kubernetes secrets, not provisioned here.

provider "aws" {
  region = var.region
  default_tags {
    tags = merge({
      Project     = var.project
      Environment = var.environment
      ManagedBy   = "terraform"
    }, var.tags)
  }
}

locals {
  name = "${var.project}-${var.environment}"
}

module "network" {
  source   = "./modules/network"
  name     = local.name
  vpc_cidr = var.vpc_cidr
  az_count = var.az_count
}

module "ecr" {
  source = "./modules/ecr"
  name   = var.project
  images = ["redirect", "api", "workers", "web"]
}

module "eks" {
  source             = "./modules/eks"
  name               = local.name
  kubernetes_version = var.eks_version
  vpc_id             = module.network.vpc_id
  private_subnet_ids = module.network.private_subnet_ids
  instance_types     = var.node_instance_types
  node_min           = var.node_min
  node_max           = var.node_max
  node_desired       = var.node_desired
}

module "data" {
  source             = "./modules/data"
  name               = local.name
  vpc_id             = module.network.vpc_id
  private_subnet_ids = module.network.private_subnet_ids
  node_security_group_id = module.eks.node_security_group_id
}
