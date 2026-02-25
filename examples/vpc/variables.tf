variable "aws_region" {
  description = "AWS region to deploy resources"
  type        = string
  default     = "us-west-1"
}

variable "vpc_name" {
  description = "Name of the VPC"
  type        = string
  default     = "terratest-vpc"
}

variable "vpc_cidr" {
  description = "CIDR block for the VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "environment" {
  description = "Environment name for tagging"
  type        = string
  default     = "test"
}

variable "pipeline_tags" {
  description = "Tags automatically injected by scripts/terraform.sh for resource identification and cleanup"
  type        = map(string)
  default     = {}
}

variable "pipeline_run_hash" {
  description = "Short unique hash from RunID, injected by scripts/terraform.sh for unique resource naming"
  type        = string
  default     = ""
}
