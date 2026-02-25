<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-19 | Updated: 2026-02-19 -->

# integration

## Purpose
Integration tests that deploy real AWS infrastructure (EKS clusters) and validate the Terraform module works end-to-end. Tests use Terratest to manage Terraform lifecycle and Kubernetes client-go for cluster validation.

## Key Files

| File | Description |
|------|-------------|
| `eks_cluster_test.go` | `TestEksClusterComplete` - Full VPC+EKS deployment with node and workload validation |
| `eks_version_test.go` | `TestEksClusterVersioned` - EKS-only test for parallel version testing with pre-deployed VPC |

## For AI Agents

### Working In This Directory
- Package is `test` (not `integration`) since Go test packages match the module root
- Tests skip in `-short` mode for CI unit-test stages
- `TestEksClusterComplete` creates its own VPC; `TestEksClusterVersioned` expects a pre-deployed VPC
- Both tests use `defer terraform.Destroy()` for cleanup
- Kubernetes validation uses AWS IAM Authenticator for token generation

### Testing Requirements
- Requires AWS credentials (profile `sandbox` or CI OIDC role)
- Run: `cd test && go test -v -timeout 40m ./integration/...`
- Run specific test: `go test -v -timeout 35m -run TestEksClusterVersioned ./integration/...`
- `TestEksClusterVersioned` requires env vars: `TF_VAR_vpc_id`, `TF_VAR_private_subnet_ids`, `TF_VAR_cluster_version`

### Test Structure

**TestEksClusterComplete** (eks_cluster_test.go):
- Deploys `examples/complete/` fixture
- Sub-tests: `ValidateClusterEndpoint`, `ValidateClusterWithAWSSdk`, `ValidateNodesReady`, `ValidateTestWorkload`
- Creates a Kubernetes client and deploys a test nginx pod

**TestEksClusterVersioned** (eks_version_test.go):
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
- `../../examples/complete/` - Full fixture for `TestEksClusterComplete`
- `../../examples/eks/` - EKS fixture for `TestEksClusterVersioned`

### External
- `github.com/gruntwork-io/terratest` - Terraform lifecycle and retry utilities
- `github.com/stretchr/testify` - assert/require
- `github.com/aws/aws-sdk-go` - EKS DescribeCluster validation
- `k8s.io/client-go` - Kubernetes API client
- `sigs.k8s.io/aws-iam-authenticator` - Token generation for EKS auth

<!-- MANUAL: -->
