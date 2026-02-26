# Template Guide: Using This Pipeline for Simpler Projects

This guide shows how to adapt this template for projects that are simpler than EKS — no version matrix, fewer layers, or no layers at all.

## Quick Reference

| Project type | LAYERS | Tests deploy infra? | Example |
|-------------|--------|:---:|---------|
| Self-contained | `""` | Yes (in Go test) | S3 bucket, IAM role |
| Single layer | `"my-module"` | No (layers handle it) | RDS, Lambda |
| Multi-layer | `"vpc rds"` | No (layers handle it) | EKS, ECS |

---

## Example 1: S3 Bucket Module (no layers)

The simplest case. Your Go test manages its own terraform.

### 1. Project structure

```
modules/s3-bucket/          # Your module
examples/basic/             # Test fixture
  main.tf
  variables.tf
  outputs.tf
test/
  integration/
    s3_bucket_test.go       # Deploys and validates S3 bucket
  unit/
    validation_test.go
```

### 2. Taskfile changes

```yaml
vars:
  PROJECT_NAME: s3-bucket
  AWS_REGION: us-east-1
  AWS_PROFILE: sandbox
  LAYERS: ""                              # No layers needed
  VALIDATE_PATHS: "modules/s3-bucket examples/basic"
  INTEGRATION_TEST_PATTERN: TestS3Bucket
  INTEGRATION_TEST_TIMEOUT: 10m
```

### 3. Test fixture

```hcl
# examples/basic/variables.tf
variable "pipeline_tags" {
  type    = map(string)
  default = {}
}

variable "pipeline_run_hash" {
  type    = string
  default = ""
}

variable "bucket_name" {
  type    = string
  default = "terratest-s3"
}

# examples/basic/main.tf
module "s3" {
  source      = "../../modules/s3-bucket"
  bucket_name = var.pipeline_run_hash != "" ? "${var.bucket_name}-${var.pipeline_run_hash}" : var.bucket_name
  tags        = merge(var.pipeline_tags, { Environment = "test" })
}
```

### 4. Go test (self-contained, like TestEksClusterComplete)

```go
func TestS3Bucket(t *testing.T) {
    terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
        TerraformDir: filepath.Join("..", "..", "examples", "basic"),
    })
    defer terraform.Destroy(t, terraformOptions)
    terraform.InitAndApply(t, terraformOptions)

    bucketID := terraform.Output(t, terraformOptions, "bucket_id")
    assert.NotEmpty(t, bucketID)
    // ... more validations
}
```

### 5. Workflow

```bash
task test          # lint + unit (no AWS)
task test-run      # deploy is a no-op, runs TestS3Bucket which deploys its own infra
```

Since `LAYERS: ""`, `deploy` and `destroy` skip automatically. The Go test handles everything.

### 6. CI changes

```yaml
env:
  ENABLE_VERSION_TESTING: "false"
  AWS_ROLE_ARN: "arn:aws:iam::ACCOUNT:role/s3-test-pipeline"
  AWS_REGION: "us-east-1"
```

---

## Example 2: RDS Module (single layer — needs VPC)

Your module needs a VPC deployed first, but there's only one version to test.

### 1. Project structure

```
modules/rds-postgres/       # Your module
examples/
  vpc/                      # Layer 1: shared VPC
    main.tf
    variables.tf
    outputs.tf
  rds/                      # Layer 2: RDS in the VPC
    main.tf
    variables.tf
    outputs.tf
test/
  integration/
    rds_test.go             # Validates RDS against deployed layers
```

### 2. Taskfile changes

```yaml
vars:
  PROJECT_NAME: rds-postgres
  AWS_REGION: eu-west-1
  AWS_PROFILE: sandbox
  LAYERS: "vpc rds"
  VALIDATE_PATHS: "modules/rds-postgres examples/vpc examples/rds"
  INTEGRATION_TEST_PATTERN: TestRdsPostgres
  INTEGRATION_TEST_TIMEOUT: 20m
```

### 3. Go test (layer-based, like TestEksClusterVersioned)

```go
func TestRdsPostgres(t *testing.T) {
    // Read outputs from deployed layers (loaded by task test-integration)
    dbEndpoint := os.Getenv("TF_VAR_db_endpoint")
    if dbEndpoint == "" {
        t.Skip("TF_VAR_db_endpoint not set, deploy layers first")
    }

    // Validate the database
    assert.Contains(t, dbEndpoint, "rds.amazonaws.com")
    // ... connect and validate
}
```

### 4. Workflow

```bash
# Manual workflow (iterate fast)
task deploy                    # Deploys vpc, then rds
task test-integration          # Validates RDS — no deploy
task test-integration          # Run again (fast!)
task destroy                   # Cleanup

# Automated pipeline
task test-run                  # Deploy → test → destroy in one command
```

### 5. Output passing

VPC outputs (like `vpc_id`, `private_subnets`) are automatically saved to `.task/vpc.outputs.json` and loaded as `TF_VAR_*` when deploying the `rds` layer.

Your RDS fixture just declares matching variables:

```hcl
# examples/rds/variables.tf
variable "vpc_id" { type = string }
variable "private_subnets" { type = list(string) }
variable "pipeline_tags" { type = map(string) default = {} }
variable "pipeline_run_hash" { type = string default = "" }
```

---

## Example 3: Lambda Module (no VPC, no layers, fast tests)

Serverless modules are often the simplest.

### 1. Taskfile changes

```yaml
vars:
  PROJECT_NAME: lambda-api
  LAYERS: ""
  VALIDATE_PATHS: "modules/lambda-api examples/basic"
  INTEGRATION_TEST_PATTERN: TestLambda
  INTEGRATION_TEST_TIMEOUT: 5m
```

### 2. Go test

```go
func TestLambdaApi(t *testing.T) {
    terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
        TerraformDir: filepath.Join("..", "..", "examples", "basic"),
    })
    defer terraform.Destroy(t, terraformOptions)
    terraform.InitAndApply(t, terraformOptions)

    functionName := terraform.Output(t, terraformOptions, "function_name")
    // Invoke the function via AWS SDK and check response
}
```

### 3. That's it

```bash
task test-run    # Deploys lambda, tests, destroys (~2 min)
```

---

## What to Remove

When copying this template for a simple project:

| Remove if... | Files to delete |
|-------------|----------------|
| No version matrix testing | `scripts/test_versions.sh`, `discover-versions` task, `test-run-versions` task |
| No VPC dependency | `examples/vpc/` |
| No EKS | `examples/eks/`, `modules/eks-cluster/` |
| Self-contained tests only | Keep `examples/complete/` pattern, delete layer examples |

## What to Keep

Always keep these — they're the generic framework:

```
scripts/terraform.sh          # Dynamic executor + auto-tagging
scripts/load_layer_outputs.sh # Layer output passing
Taskfile.yml                  # All task commands
.github/workflows/test.yml   # CI pipeline
.cloud-nuke-config.template.yml  # Cleanup safety net
```

## Checklist for New Projects

1. [ ] Copy the repo
2. [ ] Delete `modules/eks-cluster/`, `examples/eks/`, `examples/vpc/`, `examples/complete/`
3. [ ] Add your module under `modules/`
4. [ ] Create test fixture(s) under `examples/` with `pipeline_tags` and `pipeline_run_hash` variables
5. [ ] Set `PROJECT_NAME`, `LAYERS`, `VALIDATE_PATHS`, `INTEGRATION_TEST_PATTERN` in Taskfile
6. [ ] Set `ENABLE_VERSION_TESTING: "false"`, `AWS_ROLE_ARN`, `AWS_REGION` in workflow
7. [ ] Write Go tests following one of the patterns above
8. [ ] `task setup && task test`
9. [ ] `task test-run` to verify the full pipeline
