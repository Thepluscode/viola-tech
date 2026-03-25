###############################################################################
# Viola XDR — Staging Environment
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
    key            = "staging/terraform.tfstate"
    region         = "us-east-1"
    dynamodb_table = "viola-terraform-locks"
    encrypt        = true
  }
}

provider "aws" {
  region = "us-east-1"

  default_tags {
    tags = {
      Environment = "staging"
      Project     = "viola"
      ManagedBy   = "terraform"
    }
  }
}

locals {
  name        = "viola-staging"
  environment = "staging"
}

module "vpc" {
  source      = "../../modules/vpc"
  name        = local.name
  environment = local.environment
  vpc_cidr    = "10.20.0.0/16"
  azs         = ["us-east-1a", "us-east-1b", "us-east-1c"]
  private_subnets  = ["10.20.1.0/24", "10.20.2.0/24", "10.20.3.0/24"]
  public_subnets   = ["10.20.101.0/24", "10.20.102.0/24", "10.20.103.0/24"]
  database_subnets = ["10.20.201.0/24", "10.20.202.0/24", "10.20.203.0/24"]
}

module "eks" {
  source              = "../../modules/eks"
  name                = local.name
  environment         = local.environment
  vpc_id              = module.vpc.vpc_id
  private_subnet_ids  = module.vpc.private_subnet_ids
  kubernetes_version  = "1.31"
  node_instance_types = ["m6i.large"]
  node_desired_size   = 2
  node_min_size       = 2
  node_max_size       = 6
}

module "rds" {
  source                     = "../../modules/rds"
  name                       = local.name
  environment                = local.environment
  vpc_id                     = module.vpc.vpc_id
  db_subnet_group_name       = module.vpc.db_subnet_group_name
  allowed_security_group_ids = [module.eks.cluster_security_group_id]
  instance_class             = "db.r6g.large"
  allocated_storage          = 50
  max_allocated_storage      = 200
  multi_az                   = true
  backup_retention_period    = 7
}

module "msk" {
  source                     = "../../modules/msk"
  name                       = local.name
  environment                = local.environment
  vpc_id                     = module.vpc.vpc_id
  private_subnet_ids         = module.vpc.private_subnet_ids
  allowed_security_group_ids = [module.eks.cluster_security_group_id]
  broker_count               = 3
  broker_storage_gb          = 200
  retention_hours            = 72
}

module "iam" {
  source            = "../../modules/iam"
  name              = local.name
  environment       = local.environment
  oidc_provider_arn = module.eks.oidc_provider_arn
  oidc_provider_url = module.eks.oidc_provider_url
  rds_secret_arn    = module.rds.master_user_secret_arn
  msk_cluster_arn   = module.msk.cluster_arn
}

output "eks_cluster_name" {
  value = module.eks.cluster_name
}

output "rds_endpoint" {
  value = module.rds.endpoint
}

output "msk_bootstrap_brokers" {
  value = module.msk.bootstrap_brokers_tls
}
