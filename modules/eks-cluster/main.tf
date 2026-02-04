module "eks" {
  source  = "terraform-aws-modules/eks/aws"
  version = "~> 20.8"

  cluster_name    = var.cluster_name
  cluster_version = var.cluster_version

  vpc_id     = var.vpc_id
  subnet_ids = var.subnet_ids

  # Cluster endpoint configuration
  cluster_endpoint_public_access       = var.cluster_endpoint_public_access
  cluster_endpoint_private_access      = var.cluster_endpoint_private_access
  cluster_endpoint_public_access_cidrs = var.cluster_endpoint_public_access_cidrs

  # Access management
  enable_cluster_creator_admin_permissions = var.enable_cluster_creator_admin_permissions

  # Cluster addons
  cluster_addons = var.cluster_addons

  # Managed node groups
  eks_managed_node_groups         = var.eks_managed_node_groups
  eks_managed_node_group_defaults = var.eks_managed_node_group_defaults

  # IRSA
  enable_irsa = var.enable_irsa

  # Logging
  cloudwatch_log_group_retention_in_days = var.cloudwatch_log_group_retention_in_days
  cluster_enabled_log_types              = var.cluster_enabled_log_types

  # Tags
  tags         = var.tags
  cluster_tags = var.cluster_tags
}
