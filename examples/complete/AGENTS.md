<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-19 | Updated: 2026-02-19 -->

# complete

## Purpose
Full test fixture deploying both VPC and EKS cluster together. Used by `TestEksClusterComplete` integration test for end-to-end validation of the module in a self-contained environment.

## Key Files

| File | Description |
|------|-------------|
| `main.tf` | Deploys VPC (terraform-aws-modules/vpc ~> 5.5) + EKS cluster module with Kubernetes provider |
| `variables.tf` | Configuration inputs (cluster name, version, region, node sizing, GitHub run ID) |
| `outputs.tf` | Exposes cluster endpoint, CA data, version, IAM roles, VPC/subnet IDs for test assertions |
| `versions.tf` | Provider requirements matching the module |

## For AI Agents

### Working In This Directory
- This fixture creates its own VPC with 2 AZs, single NAT gateway
- CIDR: `10.0.0.0/16` with private, public, and intra subnets
- Resources tagged with `Environment`, `Terraform`, `Test`, `Owned=terratest`, `Pipeline`, `RunID`
- Kubernetes provider configured via `aws eks get-token` exec plugin
- State files may exist locally from manual test runs; they are gitignored

### Testing Requirements
- Tested by `test/integration/eks_cluster_test.go::TestEksClusterComplete`
- Deploys real AWS resources (~$0.20, ~25 min)
- Changes here must be validated with `task test-integration`

### Common Patterns
- Uses `data.aws_availability_zones.available` to discover AZs dynamically
- Subnet CIDRs computed with `cidrsubnet()` from VPC CIDR
- Subnet tags include Kubernetes ELB annotations for load balancer discovery

## Dependencies

### Internal
- `../../modules/eks-cluster` - EKS cluster module

### External
- `terraform-aws-modules/vpc/aws` ~> 5.5
- AWS provider, Kubernetes provider

<!-- MANUAL: -->
