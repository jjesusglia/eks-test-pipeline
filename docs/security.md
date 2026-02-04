# Security

## Overview

This document outlines security considerations for the Terraform EKS module and its test pipeline.

## AWS Authentication

### GitHub Actions OIDC

The pipeline uses OpenID Connect (OIDC) for AWS authentication instead of long-lived access keys.

#### Setup Requirements

1. **Create OIDC Identity Provider in AWS**:
   ```bash
   aws iam create-open-id-connect-provider \
     --url https://token.actions.githubusercontent.com \
     --client-id-list sts.amazonaws.com \
     --thumbprint-list 6938fd4d98bab03faadb97b34396831e3780aea1
   ```

2. **Create IAM Role with Trust Policy**:
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

3. **Add Required Permissions** to the IAM role:
   - EKS full access
   - VPC management
   - IAM role/policy management (for IRSA)
   - CloudWatch Logs
   - EC2 for node groups

4. **Store Role ARN** as GitHub secret `AWS_ROLE_ARN`

### Local Development

For local testing, use standard AWS credential methods:

```bash
# Option 1: AWS CLI profile
export AWS_PROFILE=your-profile

# Option 2: Environment variables
export AWS_ACCESS_KEY_ID=xxx
export AWS_SECRET_ACCESS_KEY=xxx
export AWS_REGION=us-west-2

# Option 3: AWS SSO
aws sso login --profile your-sso-profile
```

## EKS Cluster Security

### Endpoint Access

The module configures:

| Setting | Default | Purpose |
|---------|---------|---------|
| `cluster_endpoint_public_access` | `true` | Allow external access |
| `cluster_endpoint_private_access` | `true` | Allow VPC-internal access |
| `cluster_endpoint_public_access_cidrs` | `["0.0.0.0/0"]` | Restrict public access IPs |

**Recommendation for production**: Set `cluster_endpoint_public_access_cidrs` to specific IPs or disable public access entirely.

### Access Management

```hcl
enable_cluster_creator_admin_permissions = true
```

This grants the IAM principal creating the cluster admin access. For production:
- Use IAM Roles for Service Accounts (IRSA)
- Implement least-privilege access
- Use Kubernetes RBAC

### Control Plane Logging

Enabled log types:
- `api` - API server logs
- `audit` - Audit logs
- `authenticator` - Authentication logs

Retention: 7 days (configurable via `cloudwatch_log_group_retention_in_days`)

## Network Security

### VPC Configuration

The example deploys:

| Subnet Type | Purpose | Internet Access |
|-------------|---------|-----------------|
| Public | Load balancers | Direct |
| Private | Worker nodes, pods | Via NAT Gateway |
| Intra | Control plane ENIs | None |

### Security Groups

The EKS module creates:
- **Cluster security group**: Controls control plane access
- **Node security group**: Controls worker node communication

## Secrets Management

### In Code

- No secrets are hardcoded
- Sensitive outputs are marked `sensitive = true`
- CA certificate data is treated as sensitive

### In CI/CD

Required secrets:
- `AWS_ROLE_ARN`: IAM role ARN for OIDC authentication

**Never store**:
- AWS access keys
- Kubernetes kubeconfig files
- Cluster CA certificates

## Static Analysis Security Checks

### Trivy Security Scanning

Trivy performs comprehensive security scanning including:

#### Infrastructure Misconfigurations (IaC)

Key checks enforced:

| Check ID | Severity | Description | Status |
|----------|----------|-------------|--------|
| `AVD-AWS-0039` | HIGH | EKS public endpoint access | Allowed for testing* |
| `AVD-AWS-0041` | HIGH | EKS endpoint CIDR restrictions | Allowed for testing* |
| `AVD-AWS-0037` | HIGH | EKS secrets encryption with KMS | Allowed for testing* |
| `AVD-AWS-0057` | CRITICAL | IAM policy wildcards | Enforced |
| `AVD-AWS-0102` | HIGH | VPC security group 0.0.0.0/0 ingress | Enforced |

*Exceptions configured in `.trivyignore` for testing; should be enforced in production.

#### Additional Checks

Trivy also scans for:
- Known CVEs in container images
- License compliance issues
- Secrets in code (API keys, passwords)
- Kubernetes security best practices
- Compliance frameworks (CIS, PCI-DSS, etc.)

## Resource Tagging

All resources are tagged for identification and cost tracking:

```hcl
tags = {
  Environment = "test"
  Terraform   = "true"
  Test        = "terratest"
}
```

## Cleanup and Resource Lifecycle

### Automatic Cleanup

- Terratest defers `terraform destroy` on test completion
- GitHub Actions cleanup job checks for orphaned resources
- Resources tagged with `Test=terratest` can be identified for cleanup

### Manual Cleanup Procedure

If automatic cleanup fails:

```bash
# List terratest EKS clusters
aws eks list-clusters --query "clusters[?starts_with(@, 'terratest-')]"

# List tagged VPCs
aws ec2 describe-vpcs --filters "Name=tag:Test,Values=terratest"

# Delete cluster
aws eks delete-cluster --name terratest-xxx

# Delete VPC (after removing dependencies)
aws ec2 delete-vpc --vpc-id vpc-xxx
```

## Security Recommendations for Production

1. **Disable public endpoint** or restrict to known CIDRs
2. **Enable secrets encryption** with KMS
3. **Use IRSA** for pod-level AWS access
4. **Implement network policies** for pod-to-pod traffic
5. **Enable audit logging** with longer retention
6. **Use private node groups** in private subnets
7. **Implement GuardDuty** for threat detection
8. **Regular security scans** with Trivy, Checkov, or similar
