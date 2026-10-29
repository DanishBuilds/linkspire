output "region" {
  value = var.region
}

output "vpc_id" {
  value = module.network.vpc_id
}

output "eks_cluster_name" {
  value = module.eks.cluster_name
}

output "eks_cluster_endpoint" {
  value = module.eks.cluster_endpoint
}

output "ecr_repository_urls" {
  value = module.ecr.repository_urls
}

output "redis_primary_endpoint" {
  value = module.data.redis_primary_endpoint
}

output "kafka_bootstrap_brokers" {
  value     = module.data.kafka_bootstrap_brokers
  sensitive = true
}

output "kubeconfig_command" {
  description = "Run this to configure kubectl."
  value       = "aws eks update-kubeconfig --region ${var.region} --name ${module.eks.cluster_name}"
}
