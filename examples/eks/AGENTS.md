<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-19 | Updated: 2026-02-19 -->

# eks

## Purpose
EKS-only test fixture that deploys an EKS cluster into a pre-existing VPC. Designed for parallel version testing where a shared VPC is deployed once and multiple EKS clusters test different Kubernetes versions simultaneously.

## Key Files

| File | Description |
|------|-------------|
| `main.tf` | Deploys EKS cluster module into provided VPC/subnets with Kubernetes provider |
| `variables.tf` | Inputs include `vpc_id`, `private_subnet_ids`, `cluster_version`, `pipeline_tags` |
| `outputs.tf` | Cluster endpoint, CA data, name, version for test assertions |
| `versions.tf` | Provider requirements matching the module |

## For AI Agents

### Working In This Directory
- This fixture does NOT create a VPC; it requires `vpc_id` and `private_subnet_ids` as inputs
- The `ClusterVersion` tag is added for version-specific resource identification
- Used by parallel CI matrix jobs where each job tests a different EKS version
- Environment variables `TF_VAR_*` are used to pass VPC outputs from the deploy-vpc CI stage

### Testing Requirements
- Tested by `test/integration/eks_version_test.go::TestEksClusterVersioned`
- Requires pre-deployed VPC (from `examples/vpc/` or CI `deploy-vpc` job)
- Each version test is independent and can run in parallel

### Common Patterns
- Cluster name includes normalized version: `terratest-eks-1-31-{uniqueID}`
- Single node group with `t3.small`, min/max/desired = 1 for cost efficiency
- Tags include all standard fields plus `ClusterVersion` for version tracking

## Dependencies

### Internal
- `../../modules/eks-cluster` - EKS cluster module
- `../vpc/` - Shared VPC must be deployed first

### External
- AWS provider, Kubernetes provider

<!-- MANUAL: -->
