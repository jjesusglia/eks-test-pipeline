# REFERENCE IMPLEMENTATION: Only needed if your module requires a VPC layer.
# Remove this fixture if your module doesn't depend on VPC infrastructure.

provider "aws" {
  region = var.aws_region
}

data "aws_availability_zones" "available" {
  state = "available"
}

locals {
  name = var.pipeline_run_hash != "" ? "${var.vpc_name}-${var.pipeline_run_hash}" : var.vpc_name
  azs  = slice(data.aws_availability_zones.available.names, 0, 2)

  tags = merge(var.pipeline_tags, {
    Environment = var.environment
    Terraform   = "true"
    Test        = "true"
    Owned       = "terratest"
  })
}

################################################################################
# VPC Module
################################################################################

module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "~> 5.5"

  name = local.name
  cidr = var.vpc_cidr

  azs             = local.azs
  private_subnets = [for k, v in local.azs : cidrsubnet(var.vpc_cidr, 4, k)]
  public_subnets  = [for k, v in local.azs : cidrsubnet(var.vpc_cidr, 8, k + 48)]
  intra_subnets   = [for k, v in local.azs : cidrsubnet(var.vpc_cidr, 8, k + 52)]

  enable_nat_gateway = true
  single_nat_gateway = true

  public_subnet_tags = {
    "kubernetes.io/role/elb" = 1
  }

  private_subnet_tags = {
    "kubernetes.io/role/internal-elb" = 1
  }

  tags = local.tags
}
