# Development rules

## File hierarchy
  ┌──────────────────────────────────┬──────────────────────────────────────────────────────────┐
  │               Path               │                       Description                        │
  ├──────────────────────────────────┼──────────────────────────────────────────────────────────┤
  │ modules/eks-cluster/main.tf      │ EKS wrapper module using terraform-aws-modules/eks v20.x │
  ├──────────────────────────────────┼──────────────────────────────────────────────────────────┤
  │ modules/eks-cluster/variables.tf │ Input variables for the module                           │
  ├──────────────────────────────────┼──────────────────────────────────────────────────────────┤
  │ modules/eks-cluster/outputs.tf   │ Output values exposed by the module                      │
  ├──────────────────────────────────┼──────────────────────────────────────────────────────────┤
  │ modules/eks-cluster/versions.tf  │ Provider version constraints                             │
  ├──────────────────────────────────┼──────────────────────────────────────────────────────────┤
  │ examples/complete/main.tf        │ Test fixture deploying VPC + EKS                         │
  ├──────────────────────────────────┼──────────────────────────────────────────────────────────┤
  │ examples/complete/variables.tf   │ Example configuration variables                          │
  ├──────────────────────────────────┼──────────────────────────────────────────────────────────┤
  │ examples/complete/outputs.tf     │ Test outputs for validation                              │
  ├──────────────────────────────────┼──────────────────────────────────────────────────────────┤
  │ examples/complete/versions.tf    │ Example provider requirements                            │
  ├──────────────────────────────────┼──────────────────────────────────────────────────────────┤
  │ test/eks_cluster_test.go         │ Terratest integration tests                              │
  ├──────────────────────────────────┼──────────────────────────────────────────────────────────┤
  │ test/go.mod                      │ Go module definition                                     │
  ├──────────────────────────────────┼──────────────────────────────────────────────────────────┤
  │ .github/workflows/test.yml       │ GitHub Actions CI pipeline                               │
  ├──────────────────────────────────┼──────────────────────────────────────────────────────────┤
  │ .tflint.hcl                      │ TFLint configuration with AWS ruleset                    │
  ├──────────────────────────────────┼──────────────────────────────────────────────────────────┤
  │ .tfsec.yml                       │ TFSec security scanning config                           │
  ├──────────────────────────────────┼──────────────────────────────────────────────────────────┤
  │ docs/features.md                 │ Feature documentation                                    │
  ├──────────────────────────────────┼──────────────────────────────────────────────────────────┤
  │ docs/architecture.md             │ Architecture diagrams and decisions                      │
  ├──────────────────────────────────┼──────────────────────────────────────────────────────────┤
  │ docs/testing-strategy.md         │ Testing approach and debugging                           │
  ├──────────────────────────────────┼──────────────────────────────────────────────────────────┤
  │ docs/security.md                 │ Security considerations and setup                        │
  └──────────────────────────────────┴──────────────────────────────────────────────────────────┘
