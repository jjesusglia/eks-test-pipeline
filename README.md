# Terraform EKS Module Test Pipeline

Automated test pipeline for a Terraform EKS module using **Terratest (Go)** and **GitHub Actions**.

## Quick Start

### Prerequisites

- Terraform >= 1.6.0
- Go >= 1.21
- AWS account with credentials configured
- [Task](https://taskfile.dev/installation/) - task runner for simplified commands

### Setup

```bash
# Initialize Terraform and download dependencies
task setup
```

This runs:
- `terraform init` for both modules and examples
- `go mod download` and `go mod tidy` for test dependencies
- `tflint --init` to download linter plugins

## Running Tests

### With Task

```bash
# Show all available tasks
task --list

# Quick feedback loop (no AWS, ~3 seconds)
task test-unit              # Pure unit tests (validation, helpers)

# Complete local validation (no AWS, ~3 minutes)
task test                   # Static analysis + unit tests

# Integration/E2E tests (real AWS, ~25 min, ~$0.20)
task test-integration       # Full deployment with validation
task test-smoke             # Alias for integration tests
task test-e2e               # Alias for integration tests

# Run everything
task test-all               # Static + unit + integration

# Individual checks
task fmt           # Check formatting
task fmt-fix       # Fix formatting
task validate      # Validate Terraform
task tflint        # Run TFLint
task trivy         # Run security & vulnerability scan
```

## Project Structure

```
terratest/
├── modules/
│   └── eks-cluster/              # EKS wrapper module
├── examples/
│   └── complete/                 # Full test fixture
├── test/
│   └── eks_cluster_test.go        # Integration tests
├── .github/workflows/
│   └── test.yml                   # GitHub Actions pipeline
├── docs/
│   ├── features.md                # Features overview
│   ├── architecture.md            # Architecture decisions
│   ├── testing-strategy.md        # Testing approach
│   └── security.md                # Security considerations
├── .tflint.hcl                    # TFLint config
├── .trivyignore                   # Trivy exceptions
├── Taskfile.yml                   # Task automation
└── README.md                       # This file
```

## Available Commands

### Static Analysis

| Command | Purpose |
|---------|---------|
| `task fmt` | Check Terraform formatting |
| `task fmt-fix` | Fix formatting issues |
| `task validate` | Validate Terraform syntax |
| `task tflint` | Run linter checks |
| `task trivy` | Security & vulnerability scanning |
| `task trivy-detailed` | Detailed scan with JSON output |
| `task lint` | Run all checks |

### Testing

| Command | Purpose | Time | AWS | Cost |
|---------|---------|------|-----|------|
| `task test-unit` | Pure unit tests (48 tests) | ~3s | No | $0 |
| `task test` | Static + unit tests | ~3min | No | $0 |
| `task test-integration` | E2E/smoke tests (real deployment) | ~25min | Yes | ~$0.20 |
| `task test-e2e` | Alias for integration | ~25min | Yes | ~$0.20 |
| `task test-smoke` | Alias for integration | ~25min | Yes | ~$0.20 |
| `task test-all` | Everything (static + unit + integration) | ~30min | Yes | ~$0.20 |

### Utilities

| Command | Purpose |
|---------|---------|
| `task setup` | Initialize dev environment |
| `task init` | Initialize Terraform |
| `task deps` | Download Go dependencies |
| `task clean` | Clean local Terraform/Go cache |
| `task cleanup-aws` | List orphaned AWS resources |
| `task version-check` | Verify tool versions |
| `task ci` | Run CI pipeline locally |

## Documentation

- **[Features](docs/features.md)** - Module features and test pipeline
- **[Architecture](docs/architecture.md)** - System design and data flow
- **[Testing Strategy](docs/testing-strategy.md)** - Test approach and debugging
- **[Security](docs/security.md)** - Security considerations and setup

View documentation:

```bash
task docs-features
task docs-architecture
task docs-testing
task docs-security
```

## GitHub Actions Setup

### 1. Create OIDC Provider

```bash
aws iam create-open-id-connect-provider \
  --url https://token.actions.githubusercontent.com \
  --client-id-list sts.amazonaws.com \
  --thumbprint-list 6938fd4d98bab03faadb97b34396831e3780aea1
```

### 2. Create IAM Role

```bash
aws iam create-role \
  --role-name github-actions-terratest \
  --assume-role-policy-document file://trust-policy.json
```

Trust policy (`trust-policy.json`):

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Federated": "arn:aws:iam::ACCOUNT_ID:oidc-provider/token.actions.githubusercontent.com"
      },
      "Action": "sts:AssumeRoleWithWebIdentity",
      "Condition": {
        "StringEquals": {
          "token.actions.githubusercontent.com:aud": "sts.amazonaws.com"
        },
        "StringLike": {
          "token.actions.githubusercontent.com:sub": "repo:YOUR_ORG/YOUR_REPO:*"
        }
      }
    }
  ]
}
```

### 3. Add Required Permissions

```bash
aws iam attach-role-policy \
  --role-name github-actions-terratest \
  --policy-arn arn:aws:iam::aws:policy/AmazonEKSFullAccess

# Add VPC, IAM, CloudWatch permissions as needed
```

### 4. Add GitHub Secret

In your GitHub repository:

1. Go to **Settings** → **Secrets and variables** → **Actions**
2. Create secret `AWS_ROLE_ARN` with the role ARN from step 2

## Local Testing

### Cost Optimization

Integration tests deploy real AWS resources:

- **Complete test**: ~$0.15-0.20 per run (20-30 min)
- **Minimal test**: ~$0.08-0.10 per run (15-20 min)

**Tips:**
- Use `task test-integration-minimal` for faster feedback
- Integration tests only run on `main` branch or with explicit label in CI
- Always verify cleanup with `task cleanup-aws`

### Debugging

```bash
# Enable Terraform debug logging
export TF_LOG=DEBUG
export TF_LOG_PATH=/tmp/terraform.log
task test-integration-minimal

# View logs
tail -f /tmp/terraform.log

# Check for orphaned resources
task cleanup-aws
```

### Pre-commit Hook

Automatically run tests before commits:

```bash
# Install
task hook-install

# Uninstall
task hook-uninstall
```

## Troubleshooting

### "Module not installed"

```bash
task init
```

### Tests timeout

Increase timeout in Taskfile or command:

```bash
cd test
go test -v -timeout 60m ./...
```

### Terraform state corrupted

```bash
task clean
task setup
```

### Orphaned AWS resources

```bash
task cleanup-aws

# Manual cleanup
aws eks delete-cluster --name terratest-xxx
aws ec2 delete-vpc --vpc-id vpc-xxx
```

## Quick Commands

```bash
# Everything (like GitHub Actions CI)
task ci

# Quick feedback loop
task quick-test

# Watch files and auto-lint
task watch
```

## Contributing

1. Format code: `task fmt-fix`
2. Run tests: `task test`
3. Add feature docs to `docs/features.md`
4. Update architecture if needed

## Tools

- **Terraform**: Infrastructure as code
- **Terratest**: Infrastructure testing framework (Go)
- **TFLint**: Terraform linter
- **Trivy**: Security and vulnerability scanner (IaC, containers, config)
- **GitHub Actions**: CI/CD pipeline
- **Task**: Command task runner

## License

See LICENSE file for details.

## Support

- Issues: Check docs and existing issues
- Security: Review `docs/security.md`
- Architecture: Review `docs/architecture.md`
- Testing: Review `docs/testing-strategy.md`
