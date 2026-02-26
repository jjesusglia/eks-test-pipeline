<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-19 | Updated: 2026-02-19 -->

# examples

## Purpose
Terraform example configurations used as test fixtures. Each example deploys real AWS infrastructure and is referenced by integration tests and the CI pipeline.

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `eks/` | EKS-only fixture using pre-deployed VPC (see `eks/AGENTS.md`) |
| `vpc/` | Standalone VPC fixture for shared infrastructure (see `vpc/AGENTS.md`) |

## For AI Agents

### Working In This Directory
- Each example must have `main.tf`, `variables.tf`, `outputs.tf`, and `versions.tf`
- Examples reference modules via relative path `../../modules/eks-cluster`
- All resources must include `pipeline_tags` for cleanup traceability (Pipeline, RunID, Environment)
- State files (`terraform.tfstate*`) are gitignored but may exist locally

<!-- MANUAL: -->
