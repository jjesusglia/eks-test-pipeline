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

---

## Feature Log

### 2026-02-05: Integration Test Speed Optimization

**Title**: Optimize Terratest Integration Test Speed

**Summary**: Reduced integration test execution time from ~25-35 minutes to ~18-27 minutes through multiple optimizations targeting retry intervals, node count, Terraform parallelism, and Kubernetes client caching.

**Implementation Details**:

1. **Faster Retry Intervals** (`test/integration/eks_cluster_test.go:29-35`)
   - Changed `retryInterval` from 30s to 10s (`fastRetryInterval`)
   - Increased `maxRetries` from 20 to 30 to compensate
   - Applied to all `retry.DoWithRetryE` calls (cluster status, node ready, pod ready)
   - Time saved: ~1-2 minutes

2. **Reduced Node Count** (`test/integration/eks_cluster_test.go:54-65`)
   - Changed `node_desired_size` from 2 to 1
   - Changed `node_max_size` from 3 to 1
   - Changed instance type from `t3.medium` to `t3.small`
   - Single node sufficient for validating cluster functionality
   - Time saved: ~2-4 minutes

3. **Terraform Parallelism** (`test/integration/eks_cluster_test.go:64`)
   - Added `Parallelism: 20` to terraform options (default is 10)
   - Speeds up VPC resource creation (subnets, route tables, NAT gateway)
   - Time saved: ~30-60 seconds

4. **Kubernetes Client Caching** (`test/integration/eks_cluster_test.go:94-103`)
   - Create client once before k8s-based sub-tests
   - Refactored `validateNodesReady` → `validateNodesReadyWithClient`
   - Refactored `validateTestWorkload` → `validateTestWorkloadWithClient`
   - Eliminates duplicate IAM token generation and TLS connection setup
   - Time saved: ~10-15 seconds

**Total Estimated Savings**: 5-8 minutes per test run
