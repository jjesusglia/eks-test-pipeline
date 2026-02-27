# Terraform Module Testing Template

Copy-and-customize template for testing AWS Terraform modules with **Terratest (Go)**, **Task** runner, and **GitHub Actions**.

EKS is included as a reference implementation — replace it with your module.

## Requirements

- [Task](https://taskfile.dev/installation/) >= 3.20 (task runner)
- Terraform >= 1.6
- Go >= 1.21
- [TFLint](https://github.com/terraform-linters/tflint)
- [Trivy](https://github.com/aquasecurity/trivy)
- [jq](https://jqlang.github.io/jq/)
- AWS account with credentials configured
- [cloud-nuke](https://github.com/gruntwork-io/cloud-nuke) (for fallback cleanup)

## Quick Start

### 1. Copy this repo

```bash
git clone <this-repo> my-terraform-module-tests
cd my-terraform-module-tests
```

### 2. Replace the module

Delete `modules/eks-cluster/` and add your own Terraform module under `modules/`.
See [docs/template_guide.md](docs/template_guide.md) for detailed examples (S3, RDS, Lambda) and a checklist for new projects.

### 3. Create test fixtures

Create directories under `examples/` for your test infrastructure. Each fixture needs a `pipeline_tags` variable for resource tagging:

```hcl
# examples/<your-fixture>/variables.tf
variable "pipeline_tags" {
  description = "Tags for resource identification and cleanup"
  type        = map(string)
  default     = {}
}

# examples/<your-fixture>/main.tf
module "my_module" {
  source = "../../modules/my-module"
  tags   = merge(var.pipeline_tags, { Environment = "test" })
}
```

### 4. Update variables

Edit vars in `Taskfile.yml`:

| Variable | Description | Example |
|----------|-------------|---------|
| `PROJECT_NAME` | Identifies resources in AWS tags | `s3-bucket` |
| `AWS_REGION` | AWS region for deployments | `us-east-1` |
| `VALIDATE_PATHS` | Terraform dirs to validate | `"modules/s3 examples/basic"` |
| `INTEGRATION_TEST_TIMEOUT` | Test timeout | `10m` |

Edit vars in `.github/workflows/test.yml`:

| Variable | Description |
|----------|-------------|
| `AWS_ROLE_ARN` | IAM role for OIDC auth |
| `AWS_REGION` | AWS region |

### 5. Write tests

Follow `test/integration/eks_version_test.go` as a pattern:

```go
func TestMyModule(t *testing.T) {
    terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
        TerraformDir: filepath.Join("..", "..", "examples", "basic"),
    })
    defer terraform.Destroy(t, terraformOptions)
    terraform.InitAndApply(t, terraformOptions)
    // Validate outputs and infrastructure
}
```

### 6. Run

```bash
task setup    # Initialize terraform + go deps
task test     # Lint + unit tests (no AWS)
```

## Test Infrastructure Lifecycle

Go tests manage the full infrastructure lifecycle — no external scripts or CI steps needed.

The pattern used by `eks_version_test.go`:

1. Deploy shared VPC via `terraform.InitAndApply`
2. Pass VPC outputs (`vpc_id`, `private_subnets`) to parallel EKS subtests
3. Each EKS subtest gets its own temp directory (avoids state lock conflicts)
4. `defer terraform.Destroy` ensures cleanup in correct order (EKS first, then VPC)

```
TestEksClusterVersionMatrix
  ├── Deploy VPC (once)
  ├── Discover EKS versions from AWS
  ├── t.Run("group", ...)
  │   ├── EKS 1.31 (parallel) → deploy, validate, defer destroy
  │   └── EKS 1.32 (parallel) → deploy, validate, defer destroy
  └── Destroy VPC (after all subtests complete)
```

## Pipeline Tags

Every resource is automatically tagged by Go test helpers (`getPipelineTags` in `helpers_test.go`):

| Tag | Value | Purpose |
|-----|-------|---------|
| `Pipeline` | `PROJECT_NAME` from env | Identifies which project created it |
| `RunID` | GitHub run ID or `local-YYYYMMDD-HHMMSS` | Identifies the specific run |
| `Environment` | `ci` or `local` | Distinguishes CI from developer runs |

### Cleanup

```bash
task cleanup-fallback     # Delete resources matching Pipeline + RunID tags
task cleanup-project      # Delete ALL resources for this project (dry-run only)
```

## Task Commands

### Static Analysis

| Command | Purpose |
|---------|---------|
| `task fmt` | Check Terraform formatting |
| `task fmt-fix` | Fix formatting |
| `task validate-tf` | Validate Terraform syntax |
| `task tflint` | Run linter |
| `task security-scan` | Security scanning (use `-- json` for JSON output) |
| `task lint` | Run all checks |

### Testing

| Command | Purpose | AWS Required |
|---------|---------|:---:|
| `task test-unit` | Unit tests (48 tests, ~3s) | No |
| `task test` | Fmt + validate-tf + lint + unit tests | No |
| `task test-integration` | Deploy → test → destroy | Yes |
| `task test-all` | Lint + unit + integration | Yes |

### Utilities

| Command | Purpose |
|---------|---------|
| `task setup` | Initialize dev environment (includes pre-commit hooks) |
| `task ci` | Run CI pipeline locally |
| `task clean` | Clean terraform state + go cache |
| `task validate-tf -- <dir>` | Validate a single directory |
| `pre-commit run -a` | Run all pre-commit hooks manually |

## CI/CD

The GitHub Actions workflow (`.github/workflows/test.yml`) runs the same `task` commands as local development.

### Pipeline stages

```
Static analysis (parallel) → Unit tests → Integration tests → Cleanup fallback
```

Integration tests can be triggered on PRs by commenting:
- `/test-integration` — runs only integration tests (skips static analysis and unit tests since they already passed on push)
- `/test-all` — runs the full pipeline: static analysis, unit tests, and integration tests

Integration tests handle the full lifecycle internally: deploy VPC, discover EKS versions, deploy parallel EKS clusters, validate, and destroy everything via `defer`.

## Project Structure

```
├── modules/
│   └── eks-cluster/               # REFERENCE: Replace with your module
├── examples/
│   ├── vpc/                       # Layer: VPC infrastructure
│   └── eks/                       # Layer: EKS cluster (version testing)
├── test/
│   ├── integration/
│   │   ├── eks_version_test.go    # REFERENCE: Version matrix testing
│   │   └── helpers_test.go        # Shared test helpers
│   └── unit/
│       ├── validation.go          # Validation functions
│       └── validation_test.go     # Unit tests
├── scripts/
│   └── clean.sh                   # Deep clean utility (state + cache)
├── Taskfile.yml                   # Task runner configuration
├── .github/workflows/test.yml    # CI/CD pipeline
├── .cloud-nuke-config.template.yml  # Cleanup config template
└── docs/                          # Documentation
```

Files marked `REFERENCE` contain EKS-specific code — replace with your module's implementation.

## Troubleshooting

### Tests timeout

```bash
# Increase timeout in Taskfile.yml:
# INTEGRATION_TEST_TIMEOUT: 60m
```

### Orphaned AWS resources

```bash
task cleanup-fallback     # Clean up resources from last run
task cleanup-project      # See ALL resources for this project (dry-run)
```

### Terraform state corrupted

```bash
task clean && task setup
```
