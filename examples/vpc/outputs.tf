################################################################################
# VPC Outputs
################################################################################

output "vpc_id" {
  description = "ID of the VPC"
  value       = module.vpc.vpc_id
}

output "vpc_cidr_block" {
  description = "CIDR block of the VPC"
  value       = module.vpc.vpc_cidr_block
}

output "private_subnets" {
  description = "List of IDs of private subnets"
  value       = module.vpc.private_subnets
}

output "private_subnets_json" {
  description = "JSON-encoded list of private subnet IDs (for passing to other fixtures)"
  value       = jsonencode(module.vpc.private_subnets)
}

output "public_subnets" {
  description = "List of IDs of public subnets"
  value       = module.vpc.public_subnets
}

output "intra_subnets" {
  description = "List of IDs of intra subnets"
  value       = module.vpc.intra_subnets
}

output "azs" {
  description = "Availability zones used"
  value       = local.azs
}

################################################################################
# KMS Outputs
################################################################################

output "kms_key_arn" {
  description = "ARN of the shared KMS key for EKS cluster encryption"
  value       = aws_kms_key.eks.arn
}
