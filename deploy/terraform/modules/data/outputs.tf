output "redis_primary_endpoint" {
  value = aws_elasticache_replication_group.redis.primary_endpoint_address
}
output "kafka_bootstrap_brokers" {
  value = aws_msk_cluster.this.bootstrap_brokers_tls
}
