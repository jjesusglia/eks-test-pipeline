# Testing Strategy

## Overview

This document outlines the testing approach for the Terraform EKS module, covering static analysis, unit tests, and integration tests.

## Testing Pyramid

```
                    ┌─────────────────────┐
                    │  Integration Tests  │  (slow, expensive, high confidence)
                    │  ~20-30 minutes     │
                    │  Real AWS resources │
                    └─────────┬───────────┘
                              │
              ┌───────────────┴───────────────┐
              │         Unit Tests            │  (fast, isolated)
              │         ~1-2 minutes          │
              │    Helper function tests      │
              └───────────────┬───────────────┘
                              │
  ┌───────────────────────────┴───────────────────────────┐
  │                   Static Analysis                      │  (instant feedback)
  │                   ~2-3 minutes                         │
  │    Format, Lint, Security Scan, Validation            │
  └───────────────────────────────────────────────────────┘
```

## Static Analysis

### Purpose

Catch issues early without deploying infrastructure.

### Tools

| Tool | What it checks | Config |
|------|----------------|--------|
| `terraform fmt` | Code formatting consistency | N/A |
| `terraform validate` | Configuration syntax and internal consistency | N/A |
| TFLint | Best practices, naming conventions, AWS-specific rules | `.tflint.hcl` |
| Trivy | Security misconfigurations, vulnerabilities, CVEs | `.trivyignore` |

### Running Locally

```bash
# Format check
terraform fmt -check -recursive -diff

# Validate
cd modules/eks-cluster && terraform init -backend=false && terraform validate

# TFLint
tflint --init
tflint --recursive

# Trivy (security + vulnerabilities)
trivy config . --severity CRITICAL,HIGH,MEDIUM
```

## Unit Tests

### Purpose

Fast validation of helper functions and logic **without deploying infrastructure**.

**Speed**: ~3 seconds for 48 tests
**Cost**: $0 (no AWS resources)
**When**: Every commit, pre-commit hooks, CI pipeline

### Structure

```
test/
├── unit/                    # Pure unit tests (FAST)
│   ├── validation.go        # Helper functions
│   └── validation_test.go   # Unit tests (48 tests)
└── integration/             # E2E tests (SLOW)
    └── eks_cluster_test.go  # Terratest integration
```

### What's Tested

**Location**: `test/unit/`

**Functions tested**:
1. `ValidateClusterName()` - EKS cluster name validation
2. `ValidateKubernetesVersion()` - Version format check
3. `ValidateSubnetCount()` - Minimum subnet requirements
4. `ValidateTags()` - Required tags validation
5. `ValidateInstanceTypes()` - EC2 instance type format
6. `GenerateClusterTags()` - Tag merging logic
7. `ValidateNodeGroupSize()` - Min/max/desired validation

**Test coverage**: 48 test cases

### Running Unit Tests

```bash
# Via Task (recommended)
task test-unit

# With coverage report
task test-unit-coverage
open test/coverage.html

# Direct
cd test
go test -v ./unit/...
```

### Example Unit Test

```go
func TestValidateClusterName(t *testing.T) {
    tests := []struct {
        name      string
        input     string
        wantError bool
    }{
        {"valid name", "my-cluster", false},
        {"empty name", "", true},
        {"name too long", string(make([]byte, 101)), true},
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            err := ValidateClusterName(tt.input)
            if tt.wantError {
                assert.Error(t, err)
            } else {
                assert.NoError(t, err)
            }
        })
    }
}
```

### Adding New Unit Tests

1. Add helper function to `test/unit/validation.go`
2. Add tests to `test/unit/validation_test.go`
3. Run `task test-unit` to verify

Example:
```go
// In validation.go
func ValidateClusterName(name string) error {
    if name == "" {
        return fmt.Errorf("cluster name cannot be empty")
    }
    return nil
}

// In validation_test.go
func TestValidateClusterName(t *testing.T) {
    err := ValidateClusterName("")
    assert.Error(t, err)
}

## Integration Tests (E2E/Smoke)

### Purpose

Validate the module works in a real AWS environment with actual resource deployment.

**Speed**: ~25 minutes
**Cost**: ~$0.20 per run
**When**: Before releases, on main branch, explicit PR requests

### Structure

```
test/
├── unit/                    # Unit tests (FAST)
└── integration/             # Integration/E2E/Smoke (SLOW)
    └── eks_cluster_test.go  # Full deployment test
```

### Test Cases

#### TestEksClusterComplete

Full deployment test with 2 worker nodes:

1. **Deploy VPC**: Create VPC with public/private subnets
2. **Deploy EKS**: Create cluster using the wrapper module
3. **Validate Outputs**: Check cluster endpoint, CA data, version
4. **Validate AWS SDK**: Confirm cluster is ACTIVE via AWS API
5. **Validate Nodes**: Wait for worker nodes to become Ready
6. **Validate Workload**: Deploy nginx pod and verify it runs
7. **Cleanup**: Destroy all resources

#### TestEksClusterMinimal

Cost-optimized test with single small node:
- Same validation flow
- Single t3.small instance
- Lower cost footprint

### Running Locally

Prerequisites:
- AWS credentials configured
- Terraform installed
- Go installed

```bash
cd test

# Run all tests
go test -v -timeout 40m ./...

# Run specific test
go test -v -timeout 40m -run TestEksClusterComplete ./...

# Run minimal test only (faster/cheaper)
go test -v -timeout 30m -run TestEksClusterMinimal ./...
```

### Test Timeouts

| Phase | Expected Duration |
|-------|-------------------|
| VPC Creation | 2-3 minutes |
| EKS Cluster Creation | 10-15 minutes |
| Node Group Ready | 5-8 minutes |
| Pod Deployment | 1-2 minutes |
| Cleanup (Destroy) | 5-10 minutes |
| **Total** | **20-35 minutes** |

## Debugging Failed Tests

### 1. Check Terraform State

If tests fail mid-execution, resources may be left behind:

```bash
cd examples/complete
terraform state list
```

### 2. Manual Cleanup

```bash
# Force destroy
cd examples/complete
terraform destroy -auto-approve

# Or manually check AWS console for:
# - EKS clusters starting with "terratest-"
# - VPCs tagged with "Test=terratest"
```

### 3. View Test Logs

In CI:
- Check workflow run logs
- Download artifacts from failed runs

Locally:
```bash
go test -v -timeout 40m ./... 2>&1 | tee test.log
```

### 4. Common Issues

| Issue | Cause | Solution |
|-------|-------|----------|
| Node not ready | Insufficient instance quota | Request quota increase |
| Timeout on apply | EKS creation slow | Increase timeout |
| Auth failure | OIDC misconfigured | Check IAM trust policy |
| Cleanup failure | Resources stuck | Manual deletion |

## CI/CD Integration

### When Tests Run

| Trigger | Static Analysis | Unit Tests | Integration Tests |
|---------|-----------------|------------|-------------------|
| Push to main | Yes | Yes | Yes |
| Pull Request | Yes | Yes | No* |
| PR with label | Yes | Yes | Yes |

*PRs can opt-in to integration tests by adding the `run-integration-tests` label.

### Cost Control

Integration tests deploy real AWS resources:

| Resource | Hourly Cost (approx) |
|----------|---------------------|
| EKS Control Plane | $0.10 |
| t3.medium (x2) | $0.08 |
| NAT Gateway | $0.045 |
| **Total** | **~$0.25/hour** |

Per test run (~30 min): **$0.12-0.15**

### Preventing Runaway Costs

1. **Timeout limits**: 45-minute max on integration job
2. **Concurrency control**: Cancel previous runs
3. **Cleanup job**: Checks for orphaned resources
4. **Branch restrictions**: Only main runs integration by default

## Adding New Tests

### Test File Structure

```go
func TestNewFeature(t *testing.T) {
    t.Parallel()  // Run concurrently with other tests

    uniqueID := strings.ToLower(random.UniqueId())
    clusterName := fmt.Sprintf("terratest-%s", uniqueID)

    terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
        TerraformDir: "../examples/complete",
        Vars: map[string]interface{}{
            "cluster_name": clusterName,
            // ... other vars
        },
    })

    // ALWAYS defer destroy
    defer terraform.Destroy(t, terraformOptions)

    // Deploy
    terraform.InitAndApply(t, terraformOptions)

    // Validate
    // ... assertions

    // Cleanup happens via deferred Destroy
}
```

### Best Practices

1. **Always use unique names** with `random.UniqueId()`
2. **Always defer destroy** before any operations
3. **Use subtests** for related validations
4. **Add meaningful assertions** with descriptive messages
5. **Set appropriate timeouts** based on resource creation time
