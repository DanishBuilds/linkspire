# ElastiCache Redis (link cache + rate limiting) and MSK (click-event stream).
# Both are reachable only from the EKS node security group.

resource "aws_security_group" "redis" {
  name        = "${var.name}-redis"
  description = "Redis access from EKS nodes"
  vpc_id      = var.vpc_id

  ingress {
    description     = "Redis from cluster"
    from_port       = 6379
    to_port         = 6379
    protocol        = "tcp"
    security_groups = [var.node_security_group_id]
  }
  egress {
    description = "all egress"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_elasticache_subnet_group" "redis" {
  name       = "${var.name}-redis"
  subnet_ids = var.private_subnet_ids
}

resource "random_password" "redis_auth" {
  length  = 32
  special = false
}

resource "aws_elasticache_replication_group" "redis" {
  replication_group_id       = "${var.name}-redis"
  description                = "${var.name} link cache"
  engine                     = "redis"
  engine_version             = "7.1"
  node_type                  = var.redis_node_type
  num_cache_clusters         = 2
  automatic_failover_enabled = true
  multi_az_enabled           = true
  port                       = 6379
  subnet_group_name          = aws_elasticache_subnet_group.redis.name
  security_group_ids         = [aws_security_group.redis.id]
  at_rest_encryption_enabled = true
  transit_encryption_enabled = true
  auth_token                 = random_password.redis_auth.result
  snapshot_retention_limit   = 7
}

# MSK (Kafka) for the click-event pipeline.
resource "aws_security_group" "kafka" {
  name        = "${var.name}-kafka"
  description = "Kafka access from EKS nodes"
  vpc_id      = var.vpc_id

  ingress {
    description     = "Kafka TLS from cluster"
    from_port       = 9094
    to_port         = 9094
    protocol        = "tcp"
    security_groups = [var.node_security_group_id]
  }
  egress {
    description = "all egress"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_kms_key" "kafka" {
  description             = "${var.name} MSK encryption key"
  enable_key_rotation     = true
  deletion_window_in_days = 14
}

resource "aws_cloudwatch_log_group" "kafka" {
  name              = "/aws/msk/${var.name}"
  retention_in_days = 90
}

resource "aws_msk_cluster" "this" {
  cluster_name           = "${var.name}"
  kafka_version          = "3.6.0"
  number_of_broker_nodes = var.kafka_broker_count

  broker_node_group_info {
    instance_type   = "kafka.m5.large"
    client_subnets  = var.private_subnet_ids
    security_groups = [aws_security_group.kafka.id]
    storage_info {
      ebs_storage_info {
        volume_size = 100
      }
    }
  }

  encryption_info {
    encryption_at_rest_kms_key_arn = aws_kms_key.kafka.arn
    encryption_in_transit {
      client_broker = "TLS"
      in_cluster    = true
    }
  }

  logging_info {
    broker_logs {
      cloudwatch_logs {
        enabled   = true
        log_group = aws_cloudwatch_log_group.kafka.name
      }
    }
  }
}
