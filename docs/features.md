# Features

## EKS Cluster Module (`modules/eks-cluster`)

### Core Features

| Feature | Description | Status |
|---------|-------------|--------|
| EKS Cluster | Wrapper around terraform-aws-modules/eks v20.x | Implemented |
| Managed Node Groups | Support for EKS managed node groups with customizable instance types | Implemented |
| Cluster Addons | CoreDNS, kube-proxy, VPC-CNI addon management | Implemented |
| IRSA | IAM Roles for Service Accounts support | Implemented |
| Endpoint Access | Configurable public/private endpoint access | Implemented |
| Logging | CloudWatch control plane logging | Implemented |

### Configuration Options

- **Cluster Version**: Supports Kubernetes versions 1.27, 1.28, 1.29
- **Instance Types**: Configurable via `eks_managed_node_groups`
- **Scaling**: Min/max/desired size configuration
- **Network**: Custom VPC and subnet configuration
- **Security**: Cluster endpoint CIDR restrictions

## Test Pipeline

### Static Analysis (Stage 1)

| Tool | Purpose | Config File |
|------|---------|-------------|
| terraform fmt | Code formatting validation | N/A |
| terraform validate | Configuration validation | N/A |
| TFLint | Terraform linting with AWS ruleset | `.tflint.hcl` |
| Trivy | Security & vulnerability scanning (IaC misconfigurations, CVEs) | `.trivyignore` |

### Unit Tests (Stage 2)

- Go unit tests for helper functions
- Short-running tests (`-short` flag)
- No AWS resources deployed

### Integration Tests (Stage 3)

| Test | Description |
|------|-------------|
| `TestEksClusterComplete` | Full VPC + EKS deployment with 2 nodes |
| `TestEksClusterMinimal` | Minimal deployment with 1 small node |

#### Integration Test Validations

1. **Cluster Deployment**: Terraform apply completes successfully
2. **Endpoint Validation**: Cluster endpoint is HTTPS and valid EKS endpoint
3. **AWS SDK Validation**: Cluster status is ACTIVE
4. **Node Ready Check**: At least one worker node reaches Ready state
5. **Workload Validation**: Test nginx pod runs successfully

## GitHub Actions Workflow

### Triggers

- Push to `master` branches
- Pull requests to `master` branches

### Cost Control Features

- Concurrency limiting (cancels previous runs)
- Integration tests require explicit label or main branch
- 45-minute timeout on integration tests
- Cleanup job for orphaned resources

### Security Features

- OIDC authentication (no long-lived AWS keys)
- Artifact upload on failure for debugging
- Resource tagging for identification
