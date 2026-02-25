<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-19 | Updated: 2026-02-19 -->

# vpc

## Purpose
Standalone VPC fixture deployed as shared infrastructure for parallel EKS version testing. Creates a VPC with private, public, and intra subnets suitable for EKS clusters.

## Key Files

| File | Description |
|------|-------------|
| `main.tf` | Deploys VPC via `terraform-aws-modules/vpc/aws ~> 5.5` with 2 AZs, single NAT gateway |
| `variables.tf` | Inputs: `vpc_name`, `vpc_cidr` (default 10.0.0.0/16), `aws_region`, `pipeline_tags` |
| `outputs.tf` | Exposes `vpc_id`, `private_subnets_json`, subnet IDs for downstream EKS fixtures |
| `versions.tf` | Provider requirements |

## For AI Agents

### Working In This Directory
- This VPC is deployed once and shared by multiple parallel EKS tests
- In CI, Terraform state is uploaded as an artifact and downloaded by the cleanup job
- For local testing, use `task vpc-deploy` and `task vpc-destroy`
- Outputs are passed to EKS fixtures via environment variables or CI job outputs
- VPC state must be preserved between deploy and destroy steps

### Testing Requirements
- Not directly tested; validated indirectly through EKS integration tests
- Deployed by CI `deploy-vpc` job or `task vpc-deploy` locally
- Must always be destroyed after tests complete (`task vpc-destroy` or CI `cleanup-vpc`)

### Common Patterns
- 2 AZs via `data.aws_availability_zones.available`
- Subnet CIDRs computed with `cidrsubnet()` from `vpc_cidr`
- Kubernetes subnet tags: `kubernetes.io/role/elb` (public), `kubernetes.io/role/internal-elb` (private)
- All resources tagged with `Pipeline` + `RunID` for cloud-nuke cleanup

## Dependencies

### External
- `terraform-aws-modules/vpc/aws` ~> 5.5
- AWS provider

<!-- MANUAL: -->
