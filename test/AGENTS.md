<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-19 | Updated: 2026-02-19 -->

# test

## Purpose
Go test suite for the Terraform EKS module. Contains unit tests (fast, no AWS) and integration tests (real AWS deployments with validation).

## Key Files

| File | Description |
|------|-------------|
| `go.mod` | Go module definition (`github.com/jjesusglia/terraform-eks-module/test`) |
| `go.sum` | Dependency checksums |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `integration/` | Integration tests deploying real AWS infrastructure (see `integration/AGENTS.md`) |
| `unit/` | Unit tests for validation functions, no AWS required (see `unit/AGENTS.md`) |

## For AI Agents

### Working In This Directory
- Go module root is this `test/` directory, not the repo root
- Run `go mod tidy` after adding/removing dependencies
- All test commands should be run from this directory: `cd test && go test ...`
- Package name for integration tests is `test` (same directory), unit tests use package `unit`

### Testing Requirements
- Unit tests: `go test -v ./unit/...` (~3s)
- Integration tests: `go test -v -timeout 40m ./integration/...` (requires AWS credentials)
- Short mode (`-short`) skips integration tests

### Common Patterns
- Table-driven tests with `testify/assert` and `testify/require`
- Terratest `terraform.WithDefaultRetryableErrors` for retryable Terraform operations
- `defer terraform.Destroy(t, opts)` for cleanup on all integration tests
- AWS SDK validation via `eks.DescribeCluster` with retry loops
- Kubernetes client validation via `aws-iam-authenticator` token generation

## Dependencies

### External
- `github.com/gruntwork-io/terratest` - Infrastructure testing framework
- `github.com/stretchr/testify` - Assertions (assert, require)
- `github.com/aws/aws-sdk-go` - AWS SDK for cluster validation
- `k8s.io/client-go` - Kubernetes client for node/pod validation
- `sigs.k8s.io/aws-iam-authenticator` - EKS authentication token generation

<!-- MANUAL: -->
