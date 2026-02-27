# Template Guide: Using This Pipeline for Simpler Projects

This guide shows how to adapt this template for projects simpler than EKS.

## Quick Reference

| Project type | Shared infra? | Example |
|-------------|:---:|---------|
| Self-contained | No | S3 bucket, IAM role, Lambda |
| Needs VPC | Yes (deploy in parent test) | RDS, ECS, EKS |

All infrastructure is managed by Go tests — no external deploy scripts needed.

---

## Example 1: S3 Bucket Module (self-contained)

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
  VALIDATE_PATHS: "modules/s3-bucket examples/basic"
  INTEGRATION_TEST_TIMEOUT: 10m
```

### 3. Test fixture

```hcl
# examples/basic/variables.tf
variable "pipeline_tags" {
  type    = map(string)
  default = {}
}

variable "bucket_name" {
  type    = string
  default = "terratest-s3"
}

# examples/basic/main.tf
module "s3" {
  source      = "../../modules/s3-bucket"
  bucket_name = var.bucket_name
  tags        = merge(var.pipeline_tags, { Environment = "test" })
}
```

### 4. Go test (self-contained)

```go
func TestS3Bucket(t *testing.T) {
    cfg := newTestConfig(t)

    terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
        TerraformDir: copyFixtureToTemp(t, "examples/basic"),
        Vars: map[string]interface{}{
            "bucket_name":   fmt.Sprintf("test-s3-%s", cfg.UniqueID),
            "pipeline_tags": cfg.PipelineTags,
        },
    })
    defer terraform.Destroy(t, terraformOptions)
    terraform.InitAndApply(t, terraformOptions)

    bucketID := terraform.Output(t, terraformOptions, "bucket_id")
    assert.NotEmpty(t, bucketID)
}
```

### 5. Workflow

```bash
task test              # lint + unit (no AWS)
task test-integration  # deploy → validate → destroy
```

---

## Example 2: RDS Module (needs shared VPC)

Your module needs a VPC. Deploy it in the parent test, share across subtests.

### 1. Project structure

```
modules/rds-postgres/       # Your module
examples/
  vpc/                      # VPC fixture (deployed once by Go test)
  rds/                      # RDS fixture (uses VPC outputs)
test/
  integration/
    rds_test.go             # Deploys VPC, then RDS subtests
    helpers_test.go         # Shared helpers
```

### 2. Taskfile changes

```yaml
vars:
  PROJECT_NAME: rds-postgres
  AWS_REGION: eu-west-1
  VALIDATE_PATHS: "modules/rds-postgres examples/vpc examples/rds"
  INTEGRATION_TEST_TIMEOUT: 20m
```

### 3. Go test (shared VPC pattern)

```go
func TestRdsPostgres(t *testing.T) {
    cfg := newTestConfig(t)

    // Deploy shared VPC once
    vpcDir := copyFixtureToTemp(t, "examples/vpc")
    vpcOpts := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
        TerraformDir: vpcDir,
        Vars: map[string]interface{}{
            "vpc_name":      fmt.Sprintf("test-vpc-%s", cfg.UniqueID),
            "aws_region":    cfg.AWSRegion,
            "pipeline_tags": cfg.PipelineTags,
        },
    })
    defer terraform.Destroy(t, vpcOpts)
    terraform.InitAndApply(t, vpcOpts)

    vpcID := terraform.Output(t, vpcOpts, "vpc_id")
    privateSubnets := terraform.OutputList(t, vpcOpts, "private_subnets")

    // Deploy RDS using VPC outputs
    rdsDir := copyFixtureToTemp(t, "examples/rds")
    rdsOpts := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
        TerraformDir: rdsDir,
        Vars: map[string]interface{}{
            "vpc_id":          vpcID,
            "private_subnets": privateSubnets,
            "pipeline_tags":   cfg.PipelineTags,
        },
    })
    defer terraform.Destroy(t, rdsOpts)
    terraform.InitAndApply(t, rdsOpts)

    dbEndpoint := terraform.Output(t, rdsOpts, "db_endpoint")
    assert.Contains(t, dbEndpoint, "rds.amazonaws.com")
}
```

### 4. Workflow

```bash
task test-integration  # Go test handles full lifecycle: VPC → RDS → validate → destroy
```

---

## Example 3: Lambda Module (no VPC, fast tests)

Serverless modules are the simplest.

### 1. Taskfile changes

```yaml
vars:
  PROJECT_NAME: lambda-api
  VALIDATE_PATHS: "modules/lambda-api examples/basic"
  INTEGRATION_TEST_TIMEOUT: 5m
```

### 2. Go test

```go
func TestLambdaApi(t *testing.T) {
    cfg := newTestConfig(t)

    terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
        TerraformDir: copyFixtureToTemp(t, "examples/basic"),
        Vars: map[string]interface{}{
            "pipeline_tags": cfg.PipelineTags,
        },
    })
    defer terraform.Destroy(t, terraformOptions)
    terraform.InitAndApply(t, terraformOptions)

    functionName := terraform.Output(t, terraformOptions, "function_name")
    // Invoke the function via AWS SDK and check response
}
```

### 3. That's it

```bash
task test-integration  # Deploys lambda, tests, destroys (~2 min)
```

---

## What to Remove

When copying this template for a simpler project:

| Remove if... | Files to delete |
|-------------|----------------|
| No VPC dependency | `examples/vpc/` |
| No EKS | `examples/eks/`, `modules/eks-cluster/` |

## What to Keep

Always keep these — they're the generic framework:

```
Taskfile.yml                     # All task commands (includes test runners inline)
ci/validate.sh                   # Terraform init + validate
ci/cleanup.sh                    # Cloud-nuke safety net
scripts/clean.sh                 # Deep clean utility
.github/workflows/test.yml      # CI pipeline
.cloud-nuke-config.template.yml # Cleanup safety net
```

## Checklist for New Projects

1. [ ] Copy the repo
2. [ ] Delete `modules/eks-cluster/`, `examples/eks/`, `examples/vpc/`
3. [ ] Add your module under `modules/`
4. [ ] Create test fixture(s) under `examples/` with `pipeline_tags` variable
5. [ ] Set `PROJECT_NAME`, `VALIDATE_PATHS`, `INTEGRATION_TEST_TIMEOUT` in Taskfile
6. [ ] Set `AWS_ROLE_ARN`, `AWS_REGION` in workflow
7. [ ] Write Go tests following one of the patterns above
8. [ ] `task setup && task test`
9. [ ] `task test-integration` to verify the full pipeline
