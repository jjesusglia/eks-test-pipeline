variable "cluster_name" {
  description = "Name of the EKS cluster"
  type        = string
}

variable "cluster_version" {
  description = "Kubernetes version for the EKS cluster"
  type        = string
  default     = "1.29"
}

variable "vpc_id" {
  description = "ID of the VPC where the cluster will be deployed"
  type        = string
}

variable "subnet_ids" {
  description = "List of subnet IDs for the EKS cluster (control plane ENIs)"
  type        = list(string)
}

variable "cluster_endpoint_public_access" {
  description = "Enable public access to the cluster endpoint"
  type        = bool
  default     = true
}

variable "cluster_endpoint_private_access" {
  description = "Enable private access to the cluster endpoint"
  type        = bool
  default     = true
}

variable "cluster_endpoint_public_access_cidrs" {
  description = "List of CIDR blocks allowed to access the public endpoint"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "enable_cluster_creator_admin_permissions" {
  description = "Grant cluster admin permissions to the IAM principal creating the cluster"
  type        = bool
  default     = true
}

variable "cluster_addons" {
  description = "Map of cluster addon configurations"
  type        = any
  default = {
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
}

variable "eks_managed_node_groups" {
  description = "Map of EKS managed node group definitions"
  type        = any
  default     = {}
}

variable "eks_managed_node_group_defaults" {
  description = "Default configurations for EKS managed node groups"
  type        = any
  default = {
    ami_type       = "AL2023_x86_64_STANDARD"
    instance_types = ["t3.medium"]
  }
}

variable "tags" {
  description = "Tags to apply to all resources"
  type        = map(string)
  default     = {}
}

variable "cluster_tags" {
  description = "Additional tags for the EKS cluster"
  type        = map(string)
  default     = {}
}

variable "enable_irsa" {
  description = "Enable IAM Roles for Service Accounts"
  type        = bool
  default     = true
}

variable "cloudwatch_log_group_retention_in_days" {
  description = "Number of days to retain log events in CloudWatch"
  type        = number
  default     = 7
}

variable "cluster_enabled_log_types" {
  description = "List of control plane log types to enable"
  type        = list(string)
  default     = ["api", "audit", "authenticator"]
}
