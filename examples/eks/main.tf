# REFERENCE IMPLEMENTATION: Only needed for parallel version testing.
# Remove this fixture if your project doesn't test multiple versions.

provider "aws" {
  region = var.aws_region
}

locals {
  name = var.pipeline_run_hash != "" ? "${var.cluster_name}-${var.pipeline_run_hash}" : var.cluster_name

  tags = merge(var.pipeline_tags, {
    Environment    = var.environment
    Terraform      = "true"
    Test           = "true"
    Owned          = "terratest"
    ClusterVersion = var.cluster_version
  })
}

################################################################################
# EKS Cluster Module
################################################################################

module "eks" {
  source = "../../modules/eks-cluster"

  cluster_name    = local.name
  cluster_version = var.cluster_version

  vpc_id     = var.vpc_id
  subnet_ids = length(var.private_subnet_ids) > 0 ? var.private_subnet_ids : var.private_subnets

  cluster_endpoint_public_access  = true
  cluster_endpoint_private_access = true

  enable_cluster_creator_admin_permissions = true

  cluster_addons = {
    coredns = {
      most_recent = true
    }
    kube-proxy = {
      most_recent = true
    }
    vpc-cni = {
      most_recent = true
    }
  }

  eks_managed_node_group_defaults = {
    ami_type                 = "AL2023_x86_64_STANDARD"
    instance_types           = var.node_instance_types
    iam_role_use_name_prefix = false
  }

  eks_managed_node_groups = {
    default = {
      name           = "${local.name}-default"
      instance_types = var.node_instance_types

      min_size     = var.node_min_size
      max_size     = var.node_max_size
      desired_size = var.node_desired_size

      labels = {
        Environment = var.environment
        NodeGroup   = "default"
      }

      tags = local.tags
    }
  }

  tags = local.tags
}

################################################################################
# Kubernetes Provider Configuration
################################################################################

provider "kubernetes" {
  host                   = module.eks.cluster_endpoint
  cluster_ca_certificate = base64decode(module.eks.cluster_certificate_authority_data)

  exec {
    api_version = "client.authentication.k8s.io/v1beta1"
    command     = "aws"
    args        = ["eks", "get-token", "--cluster-name", module.eks.cluster_name]
  }
}
