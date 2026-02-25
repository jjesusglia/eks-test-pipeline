<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-19 | Updated: 2026-02-19 -->

# eks-cluster

## Purpose
Reusable Terraform module wrapping `terraform-aws-modules/eks/aws` v20.x. Provides an opinionated EKS cluster configuration with managed node groups, IRSA, CloudWatch logging, and standard addons (CoreDNS, kube-proxy, VPC-CNI).

## Key Files

| File | Description |
|------|-------------|
| `main.tf` | Module definition using `terraform-aws-modules/eks/aws ~> 20.8` |
| `variables.tf` | Input variables with validation (cluster_version >= 1.31) |
| `outputs.tf` | 17 outputs including cluster ID/ARN/endpoint, OIDC, IAM roles, node groups |
| `versions.tf` | Provider requirements: AWS >= 5.40.0, Kubernetes >= 2.27.0, TLS >= 4.0.0 |

## For AI Agents

### Working In This Directory
- This is a thin wrapper module; most logic is delegated to the upstream community module
- The `cluster_version` variable has a validation rule enforcing >= 1.31
- Default addons: `coredns`, `kube-proxy`, `vpc-cni` (all set to `most_recent = true`)
- Default node group AMI: `AL2023_x86_64_STANDARD` with `t3.medium` instances
- IRSA is enabled by default
- CloudWatch log retention defaults to 7 days

### Testing Requirements
- Unit validation: `cd test && go test -v ./unit/...` (validates input formats)
- Integration: `cd test && go test -v -timeout 40m ./integration/...` (deploys real cluster)
- Any variable changes must be reflected in example fixtures under `examples/`

### Common Patterns
- All variables have sensible defaults except `cluster_name`, `vpc_id`, `subnet_ids` (required)
- Tags are passed through to all resources via `tags` and `cluster_tags`
- Outputs mirror the upstream module's output interface

## Dependencies

### Internal
- Referenced by `examples/complete/`, `examples/eks/` via relative path `../../modules/eks-cluster`

### External
- `terraform-aws-modules/eks/aws` ~> 20.8 (upstream EKS module)
- Terraform >= 1.6.0
- AWS provider >= 5.40.0
- Kubernetes provider >= 2.27.0

<!-- MANUAL: -->
