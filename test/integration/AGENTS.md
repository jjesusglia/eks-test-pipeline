<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-19 | Updated: 2026-02-19 -->

# integration

## Purpose
Integration tests that deploy real AWS infrastructure (EKS clusters) and validate the Terraform module works end-to-end. Tests use Terratest to manage Terraform lifecycle and Kubernetes client-go for cluster validation.

## Key Files

| File | Description |
|------|-------------|
| `eks_version_test.go` | `TestEksClusterVersionMatrix` - Parallel EKS version testing with pre-deployed VPC |

## For AI Agents

### Working In This Directory
- Package is `test` (not `integration`) since Go test packages match the module root
- Tests skip in `-short` mode for CI unit-test stages
- `TestEksClusterVersionMatrix` expects a pre-deployed VPC (layer-based approach)
- Tests use `defer terraform.Destroy()` for cleanup
- Kubernetes validation uses AWS IAM Authenticator for token generation

### Testing Requirements
- Requires AWS credentials (profile `sandbox` or CI OIDC role)
- Run: `cd test && go test -v -timeout 40m ./integration/...`
- Run specific test: `go test -v -timeout 35m -run TestEksClusterVersionMatrix ./integration/...`
- `TestEksClusterVersionMatrix` requires env vars: `TF_VAR_vpc_id`, `TF_VAR_private_subnet_ids`

### Test Structure

**TestEksClusterVersionMatrix** (eks_version_test.go):
- Deploys `examples/eks/` fixture into pre-deployed VPC
- Sub-tests: `ValidateClusterEndpoint`, `ValidateClusterWithAWSSdk`, `ValidateNodesReady`
- Verifies cluster version matches requested version via AWS SDK

### Common Patterns
- Retry loops with `terratest/modules/retry` (30 retries, 10s interval)
- Unique cluster names via `random.UniqueId()` with `terratest-` prefix
- `t.Parallel()` for concurrent test execution
- Shared helper functions: `getKubernetesClient`, `getEnvWithDefault`

## Dependencies

### Internal
- `../../examples/eks/` - EKS fixture for `TestEksClusterVersionMatrix`

### External
- `github.com/gruntwork-io/terratest` - Terraform lifecycle and retry utilities
- `github.com/stretchr/testify` - assert/require
- `github.com/aws/aws-sdk-go` - EKS DescribeCluster validation
- `k8s.io/client-go` - Kubernetes API client
- `sigs.k8s.io/aws-iam-authenticator` - Token generation for EKS auth

<!-- MANUAL: -->
