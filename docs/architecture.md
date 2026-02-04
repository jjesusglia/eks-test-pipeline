# Architecture

## Overview

This project implements an automated test pipeline for a Terraform EKS module using Terratest (Go) and GitHub Actions.

```
┌─────────────────────────────────────────────────────────────────────────────┐
│                           GitHub Actions Pipeline                            │
├─────────────────────────────────────────────────────────────────────────────┤
│                                                                              │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ Stage 1: Static Analysis (parallel)                    ~2-3 min     │    │
│  │ ┌──────────────┐ ┌──────────────┐ ┌──────────────┐ ┌─────────────┐  │    │
│  │ │ terraform    │ │ terraform    │ │   TFLint     │ │   Trivy     │  │    │
│  │ │    fmt       │ │  validate    │ │              │ │ Security    │  │    │
│  │ └──────────────┘ └──────────────┘ └──────────────┘ └─────────────┘  │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                     │                                        │
│                                     ▼                                        │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ Stage 2: Unit Tests                                    ~1-2 min     │    │
│  │ ┌──────────────────────────────────────────────────────────────┐    │    │
│  │ │              Go Unit Tests (go test -short)                  │    │    │
│  │ └──────────────────────────────────────────────────────────────┘    │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                     │                                        │
│                                     ▼ (main branch or label)                 │
│  ┌─────────────────────────────────────────────────────────────────────┐    │
│  │ Stage 3: Integration Tests                            ~20-30 min    │    │
│  │ ┌──────────────────────────────────────────────────────────────┐    │    │
│  │ │  1. Configure AWS (OIDC)                                     │    │    │
│  │ │  2. Terratest: Deploy VPC + EKS                              │    │    │
│  │ │  3. Validate cluster connectivity                            │    │    │
│  │ │  4. Validate nodes ready                                     │    │    │
│  │ │  5. Deploy test workload                                     │    │    │
│  │ │  6. Cleanup (terraform destroy)                              │    │    │
│  │ └──────────────────────────────────────────────────────────────┘    │    │
│  └─────────────────────────────────────────────────────────────────────┘    │
│                                                                              │
└─────────────────────────────────────────────────────────────────────────────┘
```

## Project Structure

```
terratest/
├── modules/
│   └── eks-cluster/              # EKS wrapper module
│       ├── main.tf               # Module implementation
│       ├── variables.tf          # Input variables
│       ├── outputs.tf            # Output values
│       └── versions.tf           # Provider requirements
│
├── examples/
│   └── complete/                 # Full example for testing
│       ├── main.tf               # VPC + EKS deployment
│       ├── variables.tf          # Example variables
│       ├── outputs.tf            # Test outputs
│       └── versions.tf           # Provider requirements
│
├── test/
│   ├── eks_cluster_test.go       # Terratest integration tests
│   ├── go.mod                    # Go module definition
│   └── go.sum                    # Dependency checksums
│
├── .github/
│   └── workflows/
│       └── test.yml              # GitHub Actions pipeline
│
├── docs/
│   ├── features.md               # Feature documentation
│   ├── architecture.md           # This file
│   ├── testing-strategy.md       # Testing approach
│   └── security.md               # Security considerations
│
├── .tflint.hcl                   # TFLint configuration
└── .trivyignore                  # Trivy exceptions
```

## Module Architecture

### EKS Wrapper Module

The `modules/eks-cluster` module is a thin wrapper around `terraform-aws-modules/eks`:

```
┌─────────────────────────────────────────────────────────────────────┐
│                        modules/eks-cluster                           │
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │                    Input Variables                              │ │
│  │  cluster_name, cluster_version, vpc_id, subnet_ids             │ │
│  │  eks_managed_node_groups, cluster_addons, tags                 │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                              │                                       │
│                              ▼                                       │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │              terraform-aws-modules/eks v20.x                    │ │
│  │                                                                 │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │ │
│  │  │ EKS Cluster │  │ Node Groups │  │ IAM Roles / IRSA        │ │ │
│  │  └─────────────┘  └─────────────┘  └─────────────────────────┘ │ │
│  │  ┌─────────────┐  ┌─────────────┐  ┌─────────────────────────┐ │ │
│  │  │ Addons      │  │ Security    │  │ CloudWatch Logs         │ │ │
│  │  │             │  │ Groups      │  │                         │ │ │
│  │  └─────────────┘  └─────────────┘  └─────────────────────────┘ │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                              │                                       │
│                              ▼                                       │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │                    Output Values                                │ │
│  │  cluster_endpoint, cluster_ca_data, cluster_name               │ │
│  │  oidc_provider_arn, node_security_group_id                     │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### Test Infrastructure

The `examples/complete` fixture deploys a full test environment:

```
┌─────────────────────────────────────────────────────────────────────┐
│                           AWS Account                                │
│                                                                      │
│  ┌────────────────────────────────────────────────────────────────┐ │
│  │                         VPC (10.0.0.0/16)                       │ │
│  │                                                                 │ │
│  │  ┌─────────────────────────────────────────────────────────┐   │ │
│  │  │                    Public Subnets                        │   │ │
│  │  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐│   │ │
│  │  │  │  AZ-a       │ │  AZ-b       │ │  AZ-c               ││   │ │
│  │  │  │  10.0.48.x  │ │  10.0.49.x  │ │  10.0.50.x          ││   │ │
│  │  │  └─────────────┘ └─────────────┘ └─────────────────────┘│   │ │
│  │  └─────────────────────────────────────────────────────────┘   │ │
│  │                              │                                  │ │
│  │                         NAT Gateway                             │ │
│  │                              │                                  │ │
│  │  ┌─────────────────────────────────────────────────────────┐   │ │
│  │  │                    Private Subnets                       │   │ │
│  │  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐│   │ │
│  │  │  │  AZ-a       │ │  AZ-b       │ │  AZ-c               ││   │ │
│  │  │  │  10.0.0.x   │ │  10.0.16.x  │ │  10.0.32.x          ││   │ │
│  │  │  └──────┬──────┘ └──────┬──────┘ └──────────┬──────────┘│   │ │
│  │  │         │               │                    │           │   │ │
│  │  │         └───────────────┼────────────────────┘           │   │ │
│  │  │                         │                                │   │ │
│  │  │  ┌──────────────────────┴────────────────────────────┐  │   │ │
│  │  │  │                   EKS Cluster                      │  │   │ │
│  │  │  │  ┌─────────────────────────────────────────────┐  │  │   │ │
│  │  │  │  │             Control Plane                    │  │  │   │ │
│  │  │  │  │  (AWS Managed)                              │  │  │   │ │
│  │  │  │  └─────────────────────────────────────────────┘  │  │   │ │
│  │  │  │  ┌─────────────────────────────────────────────┐  │  │   │ │
│  │  │  │  │          Managed Node Group                  │  │  │   │ │
│  │  │  │  │  ┌───────────┐  ┌───────────┐               │  │  │   │ │
│  │  │  │  │  │ t3.medium │  │ t3.medium │               │  │  │   │ │
│  │  │  │  │  │  Node 1   │  │  Node 2   │               │  │  │   │ │
│  │  │  │  │  └───────────┘  └───────────┘               │  │  │   │ │
│  │  │  │  └─────────────────────────────────────────────┘  │  │   │ │
│  │  │  └───────────────────────────────────────────────────┘  │   │ │
│  │  └─────────────────────────────────────────────────────────┘   │ │
│  └────────────────────────────────────────────────────────────────┘ │
│                                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

## Data Flow

### Test Execution Flow

```
┌────────────────┐
│   Developer    │
│   pushes code  │
└───────┬────────┘
        │
        ▼
┌───────────────────────────────────────────────────────────────┐
│                    GitHub Actions                              │
│                                                                │
│  1. Checkout code                                              │
│  2. Run static analysis (parallel)                             │
│     ├── terraform fmt                                          │
│     ├── terraform validate                                     │
│     ├── tflint                                                 │
│     └── trivy (security + vulnerabilities)                     │
│  3. Run unit tests                                             │
│  4. (if main branch) Run integration tests                     │
│     a. OIDC auth to AWS                                        │
│     b. Terratest runs tests                                    │
│        ├── terraform init                                      │
│        ├── terraform apply                                     │
│        ├── validate outputs                                    │
│        ├── kubernetes tests                                    │
│        └── terraform destroy                                   │
│  5. Cleanup check                                              │
│  6. Report results                                             │
└───────────────────────────────────────────────────────────────┘
```

## Key Design Decisions

### 1. Module Wrapper Pattern

Using a thin wrapper around the community EKS module allows:
- Organizational defaults to be enforced
- Simplified interface for users
- Easier testing of specific configurations

### 2. OIDC for AWS Authentication

- No long-lived credentials stored in GitHub secrets
- Follows AWS security best practices
- Session-scoped permissions

### 3. Parallel Static Analysis

- Faster feedback loop
- Independent checks don't block each other
- Fails fast on any issue

### 4. Integration Test Gating

- Only runs on main branch or with explicit label
- Prevents excessive AWS costs on every PR
- Can be triggered manually when needed

### 5. Terratest for Integration Tests

- Real AWS infrastructure validation
- Kubernetes cluster connectivity testing
- Workload deployment verification
- Automatic cleanup on test completion
