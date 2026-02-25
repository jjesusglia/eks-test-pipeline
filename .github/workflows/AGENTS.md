<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-19 | Updated: 2026-02-19 -->

# workflows

## Purpose
GitHub Actions CI/CD pipeline definitions for automated testing, security scanning, and resource cleanup.

## Key Files

| File | Description |
|------|-------------|
| `test.yml` | Main CI pipeline with 7-stage workflow for EKS module testing |

## For AI Agents

### Working In This Directory
- The pipeline triggers on push to `master` and pull requests targeting `master`
- Integration tests only run on `master` or when PR has `run-integration-tests` label
- Concurrency group prevents duplicate runs on the same branch

### Pipeline Stages (test.yml)
1. **Static Analysis** (parallel): terraform-fmt, terraform-validate (matrix), tflint, trivy
2. **Unit Tests**: Go unit tests with `-short` flag
3. **Version Discovery**: Queries AWS for supported EKS versions >= 1.31
4. **Deploy Shared VPC**: Single VPC for all parallel EKS tests, state uploaded as artifact
5. **Parallel EKS Tests**: Matrix strategy over discovered versions, each deploys EKS into shared VPC
6. **Cleanup VPC**: Terraform destroy using downloaded state artifact
7. **Fallback Cleanup**: cloud-nuke with Pipeline + RunID tag filtering as safety net

### Key Configuration
- AWS auth via OIDC (`role-to-assume` with `id-token: write` permission)
- Terraform 1.6.6, Go 1.21, TFLint v0.50.3
- Region: `us-west-1`
- Minimum EKS version: `1.31` (env var `MIN_EKS_VERSION`)
- VPC state passed between jobs via `actions/upload-artifact` / `actions/download-artifact`

### Common Patterns
- Matrix strategy for parallel validation and version testing
- `fail-fast: false` on integration tests so all versions complete
- `if: always()` on cleanup jobs to ensure resources are destroyed
- Trivy results uploaded to GitHub Security tab via SARIF format

## Dependencies

### External
- `actions/checkout@v4`
- `hashicorp/setup-terraform@v3`
- `actions/setup-go@v5`
- `terraform-linters/setup-tflint@v4`
- `aquasecurity/trivy-action@master`
- `aws-actions/configure-aws-credentials@v4`
- `github/codeql-action/upload-sarif@v3`
- cloud-nuke v0.46.0

<!-- MANUAL: -->
