variable "aws_region" {
  description = "AWS region to deploy resources"
  type        = string
  default     = "us-west-1"
}

variable "cluster_name" {
  description = "Name of the EKS cluster"
  type        = string
  default     = "terratest-eks"
}

variable "cluster_version" {
  description = "Kubernetes version for the EKS cluster"
  type        = string
  default     = "1.31"
}

variable "vpc_id" {
  description = "ID of the VPC where EKS will be deployed"
  type        = string
}

variable "private_subnet_ids" {
  description = "List of private subnet IDs for EKS nodes"
  type        = list(string)
  default     = []
}

variable "private_subnets" {
  description = "Alias for private_subnet_ids (auto-loaded from VPC layer outputs)"
  type        = list(string)
  default     = []
}

variable "environment" {
  description = "Environment name for tagging"
  type        = string
  default     = "test"
}

variable "node_instance_types" {
  description = "Instance types for the managed node group"
  type        = list(string)
  default     = ["t3.small"]
}

variable "node_desired_size" {
  description = "Desired number of nodes in the managed node group"
  type        = number
  default     = 1
}

variable "node_min_size" {
  description = "Minimum number of nodes in the managed node group"
  type        = number
  default     = 1
}

variable "node_max_size" {
  description = "Maximum number of nodes in the managed node group"
  type        = number
  default     = 2
}

variable "pipeline_tags" {
  description = "Tags for resource identification and cleanup (injected by Go test helpers)"
  type        = map(string)
  default     = {}
}

variable "pipeline_run_hash" {
  description = "Short unique hash from RunID for unique resource naming"
  type        = string
  default     = ""
}
