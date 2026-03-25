###############################################################################
# Viola XDR — Production Environment
###############################################################################

terraform {
  required_version = ">= 1.6"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }

  backend "s3" {
    bucket         = "viola-terraform-state"
    key            = "prod/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "viola-terraform-locks"
    encrypt        = true
  }
}

provider "aws" {
  region = "us-east-1"

  default_tags {
    tags = {
      Environment = "prod"
      Project     = "viola"
      ManagedBy   = "terraform"
    }
  }
}

locals {
  name        = "viola-prod"
  environment = "prod"
}

# ── VPC ──────────────────────────────────────────────────────────────────────

module "vpc" {
  source = "../../modules/vpc"

  name        = local.name
  environment = local.environment
  vpc_cidr    = "10.0.0.0/16"
  azs         = ["us-east-1a", "us-east-1b", "us-east-1c"]

  private_subnets  = ["10.0.1.0/24", "10.0.2.0/24", "10.0.3.0/24"]
  public_subnets   = ["10.0.101.0/24", "10.0.102.0/24", "10.0.103.0/24"]
  database_subnets = ["10.0.201.0/24", "10.0.202.0/24", "10.0.203.0/24"]
}

# ── EKS ──────────────────────────────────────────────────────────────────────

module "eks" {
  source = "../../modules/eks"

  name                = local.name
  environment         = local.environment
  vpc_id              = module.vpc.vpc_id
  private_subnet_ids  = module.vpc.private_subnet_ids
  kubernetes_version  = "1.31"
  node_instance_types = ["m6i.xlarge"]
  node_desired_size   = 3
  node_min_size       = 3
  node_max_size       = 15
}

# ── RDS ──────────────────────────────────────────────────────────────────────

module "rds" {
  source = "../../modules/rds"

  name                       = local.name
  environment                = local.environment
  vpc_id                     = module.vpc.vpc_id
  db_subnet_group_name       = module.vpc.db_subnet_group_name
  allowed_security_group_ids = [module.eks.cluster_security_group_id]
  instance_class             = "db.r6g.xlarge"
  allocated_storage          = 200
  max_allocated_storage      = 1000
  multi_az                   = true
  backup_retention_period    = 30
}

# ── MSK ──────────────────────────────────────────────────────────────────────

module "msk" {
  source = "../../modules/msk"

  name                       = local.name
  environment                = local.environment
  vpc_id                     = module.vpc.vpc_id
  private_subnet_ids         = module.vpc.private_subnet_ids
  allowed_security_group_ids = [module.eks.cluster_security_group_id]
  broker_instance_type       = "kafka.m5.xlarge"
  broker_count               = 3
  broker_storage_gb          = 1000
  retention_hours            = 168 # 7 days
}

# ── IAM (IRSA) ──────────────────────────────────────────────────────────────

module "iam" {
  source = "../../modules/iam"

  name              = local.name
  environment       = local.environment
  oidc_provider_arn = module.eks.oidc_provider_arn
  oidc_provider_url = module.eks.oidc_provider_url
  rds_secret_arn    = module.rds.master_user_secret_arn
  msk_cluster_arn   = module.msk.cluster_arn
}

# ── Outputs ──────────────────────────────────────────────────────────────────

output "eks_cluster_name" {
  value = module.eks.cluster_name
}

output "rds_endpoint" {
  value     = module.rds.endpoint
  sensitive = true
}

output "msk_bootstrap_brokers" {
  value = module.msk.bootstrap_brokers_tls
}
