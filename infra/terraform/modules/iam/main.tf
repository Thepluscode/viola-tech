###############################################################################
# Viola IAM Module
# Creates IRSA (IAM Roles for Service Accounts) for Viola services.
###############################################################################

variable "name" {
  description = "Name prefix"
  type        = string
}

variable "environment" {
  description = "Environment (dev, staging, prod)"
  type        = string
}

variable "oidc_provider_arn" {
  description = "EKS OIDC provider ARN"
  type        = string
}

variable "oidc_provider_url" {
  description = "EKS OIDC provider URL (without https://)"
  type        = string
}

variable "namespace" {
  description = "Kubernetes namespace"
  type        = string
  default     = "viola"
}

variable "rds_secret_arn" {
  description = "ARN of the RDS master user secret"
  type        = string
}

variable "msk_cluster_arn" {
  description = "ARN of the MSK cluster"
  type        = string
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

  oidc_issuer = replace(var.oidc_provider_url, "https://", "")
}

# ── Gateway API Service Role ───────────────────────────────────────────────
# Needs: RDS read/write, MSK produce/consume, Secrets Manager

resource "aws_iam_role" "gateway_api" {
  name = "${var.name}-gateway-api"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Federated = var.oidc_provider_arn
      }
      Action = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "${local.oidc_issuer}:sub" = "system:serviceaccount:${var.namespace}:viola-gateway-api"
          "${local.oidc_issuer}:aud" = "sts.amazonaws.com"
        }
      }
    }]
  })

  tags = local.common_tags
}

resource "aws_iam_role_policy" "gateway_api" {
  name = "${var.name}-gateway-api-policy"
  role = aws_iam_role.gateway_api.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "SecretsAccess"
        Effect = "Allow"
        Action = [
          "secretsmanager:GetSecretValue",
        ]
        Resource = [var.rds_secret_arn]
      },
      {
        Sid    = "MSKAccess"
        Effect = "Allow"
        Action = [
          "kafka-cluster:Connect",
          "kafka-cluster:DescribeTopic",
          "kafka-cluster:WriteData",
          "kafka-cluster:ReadData",
          "kafka-cluster:DescribeGroup",
          "kafka-cluster:AlterGroup",
        ]
        Resource = ["${var.msk_cluster_arn}/*"]
      },
    ]
  })
}

# ── Detection Service Role ─────────────────────────────────────────────────

resource "aws_iam_role" "detection" {
  name = "${var.name}-detection"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Federated = var.oidc_provider_arn
      }
      Action = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "${local.oidc_issuer}:sub" = "system:serviceaccount:${var.namespace}:viola-detection"
          "${local.oidc_issuer}:aud" = "sts.amazonaws.com"
        }
      }
    }]
  })

  tags = local.common_tags
}

resource "aws_iam_role_policy" "detection" {
  name = "${var.name}-detection-policy"
  role = aws_iam_role.detection.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Sid    = "MSKAccess"
      Effect = "Allow"
      Action = [
        "kafka-cluster:Connect",
        "kafka-cluster:DescribeTopic",
        "kafka-cluster:WriteData",
        "kafka-cluster:ReadData",
        "kafka-cluster:DescribeGroup",
        "kafka-cluster:AlterGroup",
      ]
      Resource = ["${var.msk_cluster_arn}/*"]
    }]
  })
}

# ── Workers Service Role ───────────────────────────────────────────────────

resource "aws_iam_role" "workers" {
  name = "${var.name}-workers"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect = "Allow"
      Principal = {
        Federated = var.oidc_provider_arn
      }
      Action = "sts:AssumeRoleWithWebIdentity"
      Condition = {
        StringEquals = {
          "${local.oidc_issuer}:sub" = "system:serviceaccount:${var.namespace}:viola-workers"
          "${local.oidc_issuer}:aud" = "sts.amazonaws.com"
        }
      }
    }]
  })

  tags = local.common_tags
}

resource "aws_iam_role_policy" "workers" {
  name = "${var.name}-workers-policy"
  role = aws_iam_role.workers.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "SecretsAccess"
        Effect = "Allow"
        Action = ["secretsmanager:GetSecretValue"]
        Resource = [var.rds_secret_arn]
      },
      {
        Sid    = "MSKAccess"
        Effect = "Allow"
        Action = [
          "kafka-cluster:Connect",
          "kafka-cluster:DescribeTopic",
          "kafka-cluster:WriteData",
          "kafka-cluster:ReadData",
          "kafka-cluster:DescribeGroup",
          "kafka-cluster:AlterGroup",
        ]
        Resource = ["${var.msk_cluster_arn}/*"]
      },
    ]
  })
}

# ── Outputs ─────────────────────────────────────────────────────────────────

output "gateway_api_role_arn" {
  value = aws_iam_role.gateway_api.arn
}

output "detection_role_arn" {
  value = aws_iam_role.detection.arn
}

output "workers_role_arn" {
  value = aws_iam_role.workers.arn
}
