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

### 3. Create test fixtures

Create directories under `examples/` for your test infrastructure. Each fixture needs a `pipeline_tags` variable for automatic tagging:

```hcl
# examples/<your-fixture>/variables.tf
variable "pipeline_tags" {
  description = "Tags injected by scripts/terraform.sh for resource identification and cleanup"
  type        = map(string)
  default     = {}
}

# examples/<your-fixture>/main.tf
module "my_module" {
  source = "../../modules/my-module"
  tags   = merge(var.pipeline_tags, { Environment = "test" })
}
```

### 4. Configure layers

Set `LAYERS` in `Taskfile.yml` to define your deploy order:

```yaml
vars:
  PROJECT_NAME: my-module        # Used in Pipeline tag
  LAYERS: ""                     # No infra deps (tests manage their own terraform)
  # LAYERS: "vpc"                # Module needs VPC deployed first
  # LAYERS: "vpc eks"            # Multi-layer: VPC then EKS
```

### 5. Update variables

Edit ~10 vars in `Taskfile.yml`:

| Variable | Description | Example |
|----------|-------------|---------|
| `PROJECT_NAME` | Identifies resources in AWS tags | `s3-bucket` |
| `AWS_REGION` | AWS region for deployments | `us-east-1` |
| `AWS_PROFILE` | AWS CLI profile for local runs | `sandbox` |
| `LAYERS` | Deploy order (space-separated) | `""` or `"vpc"` |
| `VALIDATE_PATHS` | Terraform dirs to validate | `"modules/s3 examples/basic"` |
| `INTEGRATION_TEST_PATTERN` | Go test function match | `TestS3Bucket` |
| `INTEGRATION_TEST_TIMEOUT` | Test timeout | `10m` |

Edit ~3 vars in `.github/workflows/test.yml`:

| Variable | Description |
|----------|-------------|
| `ENABLE_VERSION_TESTING` | `"false"` for simple projects |
| `AWS_ROLE_ARN` | IAM role for OIDC auth |
| `AWS_REGION` | AWS region |

### 6. Write tests

Follow `test/integration/eks_cluster_test.go` as a pattern:

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

### 7. Run

```bash
task setup    # Initialize terraform + go deps
task test     # Lint + unit tests (no AWS)
```

## Layer System

Layers are infrastructure dependencies deployed in order before tests run.

| `LAYERS` value | Behavior |
|----------------|----------|
| `""` | No layers. Tests manage their own terraform. |
| `"account-setup"` | Single layer. Deploy → test → destroy. |
| `"vpc eks"` | Multi-layer. Deploy vpc, then eks. Destroy in reverse. |

### Commands

```bash
task deploy -- vpc        # Deploy a single layer
task deploy               # Deploy all layers (left to right)
task destroy -- vpc       # Destroy a single layer
task destroy              # Destroy all layers (right to left)
task plan -- vpc          # Plan a single layer
task plan                 # Plan all layers
```

### Output passing

When a layer is deployed, its terraform outputs are saved to `.task/<layer>.outputs.json`. The next layer automatically loads them as `TF_VAR_*` environment variables via `scripts/load_layer_outputs.sh`.

## Pipeline Tags

Every resource deployed by this template is automatically tagged:

| Tag | Value | Purpose |
|-----|-------|---------|
| `Pipeline` | `PROJECT_NAME` from Taskfile | Identifies which project created it |
| `RunID` | GitHub run ID or `local-YYYYMMDD-HHMMSS` | Identifies the specific run |
| `Environment` | `ci` or `local` | Distinguishes CI from developer runs |

Tags are injected by `scripts/terraform.sh` via `TF_VAR_pipeline_tags`. No manual tagging needed.

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
| `task test-all-versions` | Parallel version testing | Yes |

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

### Simple pipeline (default)

```
task fmt → task validate-tf → task tflint → task security-scan → task test-unit → task test-integration
```

### Version testing pipeline (opt-in)

Set `ENABLE_VERSION_TESTING: "true"` in the workflow to enable parallel matrix testing:

```
Static analysis → Unit tests → Discover versions → Deploy VPC
  → [per version] Deploy EKS → Go test → Destroy EKS
  → Destroy VPC → Cleanup fallback
```

## Project Structure

```
├── modules/
│   └── eks-cluster/               # REFERENCE: Replace with your module
├── examples/
│   ├── vpc/                       # Layer: VPC infrastructure
│   ├── eks/                       # Layer: EKS cluster (version testing)
│   └── complete/                  # Self-contained fixture (VPC + EKS)
├── test/
│   ├── integration/
│   │   ├── eks_cluster_test.go    # REFERENCE: Integration tests
│   │   ├── eks_version_test.go    # REFERENCE: Version testing
│   │   └── helpers_test.go        # Shared test helpers
│   └── unit/
│       ├── validation.go          # Validation functions
│       └── validation_test.go     # Unit tests
├── scripts/
│   ├── terraform.sh               # Dynamic terraform executor + auto-tagging
│   └── load_layer_outputs.sh      # Layer output → TF_VAR translation
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
