###############################################################################
# Viola MSK Module
# Creates an Amazon MSK (Managed Kafka) cluster for event streaming.
###############################################################################

variable "name" {
  description = "Cluster name"
  type        = string
}

variable "environment" {
  description = "Environment (dev, staging, prod)"
  type        = string
}

variable "vpc_id" {
  description = "VPC ID"
  type        = string
}

variable "private_subnet_ids" {
  description = "Private subnet IDs (one per AZ, need at least 2)"
  type        = list(string)
}

variable "allowed_security_group_ids" {
  description = "Security group IDs allowed to connect to Kafka"
  type        = list(string)
  default     = []
}

variable "kafka_version" {
  description = "Kafka version"
  type        = string
  default     = "3.7.x.kraft"
}

variable "broker_instance_type" {
  description = "Broker instance type"
  type        = string
  default     = "kafka.m5.large"
}

variable "broker_count" {
  description = "Number of brokers (must be multiple of AZ count)"
  type        = number
  default     = 3
}

variable "broker_storage_gb" {
  description = "EBS volume size per broker in GB"
  type        = number
  default     = 500
}

variable "retention_hours" {
  description = "Log retention in hours"
  type        = number
  default     = 168 # 7 days
}

variable "tags" {
  description = "Additional tags"
  type        = map(string)
  default     = {}
}

locals {
  common_tags = merge(var.tags, {
    Environment = var.environment
    Project     = "viola"
    ManagedBy   = "terraform"
  })

  is_prod = var.environment == "prod"
}

# ── Security Group ─────────────────────────────────────────────────────────

resource "aws_security_group" "msk" {
  name_prefix = "${var.name}-msk-"
  vpc_id      = var.vpc_id

  # Broker-to-broker communication
  ingress {
    from_port = 9092
    to_port   = 9098
    protocol  = "tcp"
    self      = true
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(local.common_tags, {
    Name = "${var.name}-msk-sg"
  })

  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_security_group_rule" "msk_ingress" {
  count = length(var.allowed_security_group_ids)

  type                     = "ingress"
  from_port                = 9092
  to_port                  = 9098
  protocol                 = "tcp"
  source_security_group_id = var.allowed_security_group_ids[count.index]
  security_group_id        = aws_security_group.msk.id
}

# ── KMS Key ────────────────────────────────────────────────────────────────

resource "aws_kms_key" "msk" {
  description             = "MSK encryption key for ${var.name}"
  deletion_window_in_days = 7
  enable_key_rotation     = true

  tags = local.common_tags
}

# ── CloudWatch Log Group ──────────────────────────────────────────────────

resource "aws_cloudwatch_log_group" "msk" {
  name              = "/viola/${var.environment}/msk"
  retention_in_days = local.is_prod ? 30 : 7

  tags = local.common_tags
}

# ── MSK Configuration ─────────────────────────────────────────────────────

resource "aws_msk_configuration" "main" {
  name              = "${var.name}-config"
  kafka_versions    = [var.kafka_version]

  server_properties = <<-PROPERTIES
    auto.create.topics.enable=false
    default.replication.factor=3
    min.insync.replicas=2
    num.partitions=6
    log.retention.hours=${var.retention_hours}
    log.segment.bytes=1073741824
    unclean.leader.election.enable=false
    message.max.bytes=10485760
  PROPERTIES
}

# ── MSK Cluster ────────────────────────────────────────────────────────────

resource "aws_msk_cluster" "main" {
  cluster_name           = var.name
  kafka_version          = var.kafka_version
  number_of_broker_nodes = local.is_prod ? var.broker_count : 2

  configuration_info {
    arn      = aws_msk_configuration.main.arn
    revision = aws_msk_configuration.main.latest_revision
  }

  broker_node_group_info {
    instance_type  = local.is_prod ? var.broker_instance_type : "kafka.t3.small"
    client_subnets = var.private_subnet_ids
    security_groups = [aws_security_group.msk.id]

    storage_info {
      ebs_storage_info {
        volume_size = local.is_prod ? var.broker_storage_gb : 100
      }
    }

    connectivity_info {
      public_access {
        type = "DISABLED"
      }
    }
  }

  encryption_info {
    encryption_at_rest_kms_key_arn = aws_kms_key.msk.arn

    encryption_in_transit {
      client_broker = "TLS"
      in_cluster    = true
    }
  }

  logging_info {
    broker_logs {
      cloudwatch_logs {
        enabled   = true
        log_group = aws_cloudwatch_log_group.msk.name
      }
    }
  }

  open_monitoring {
    prometheus {
      jmx_exporter {
        enabled_in_broker = true
      }
      node_exporter {
        enabled_in_broker = true
      }
    }
  }

  tags = local.common_tags
}

# ── Outputs ─────────────────────────────────────────────────────────────────

output "bootstrap_brokers_tls" {
  value = aws_msk_cluster.main.bootstrap_brokers_tls
}

output "zookeeper_connect" {
  value = aws_msk_cluster.main.zookeeper_connect_string
}

output "cluster_arn" {
  value = aws_msk_cluster.main.arn
}

output "security_group_id" {
  value = aws_security_group.msk.id
}
