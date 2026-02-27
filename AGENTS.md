<!-- Generated: 2026-02-19 | Updated: 2026-02-19 -->

# terratest

## Purpose
Automated test pipeline for a Terraform EKS module using Terratest (Go) and GitHub Actions. Provides a reusable EKS cluster module with comprehensive integration testing, static analysis, security scanning, and parallel multi-version EKS validation.

## Key Files

| File | Description |
|------|-------------|
| `README.md` | Project documentation with setup, usage, and troubleshooting |
| `Taskfile.yml` | Task runner automation for all dev workflows (lint, test, deploy, cleanup) |
| `.cloud-nuke-config.template.yml` | Cloud-nuke config template for orphaned AWS resource cleanup via Pipeline + RunID tags |
| `.tflint.hcl` | TFLint configuration with AWS ruleset |
| `.trivyignore` | Trivy security scan exceptions |
| `.gitignore` | Git ignore rules |

## Subdirectories

| Directory | Purpose |
|-----------|---------|
| `modules/` | Reusable Terraform modules (see `modules/AGENTS.md`) |
| `examples/` | Terraform example fixtures for testing (see `examples/AGENTS.md`) |
| `test/` | Go test suite with unit and integration tests (see `test/AGENTS.md`) |
| `.github/` | GitHub Actions CI/CD pipeline (see `.github/AGENTS.md`) |
| `docs/` | Project documentation (ADRs, features, security) |
| `.claude/` | Claude Code session context and documentation |

## For AI Agents

### Working In This Directory
- Run `task setup` after cloning to initialize Terraform and Go dependencies
- Use `task --list` to see all available commands
- Region `us-west-1` is the default in Taskfile.yml
- All Terraform resources are tagged with `Pipeline`, `RunID`, and `Environment` for cleanup traceability (injected by `getPipelineTags` in Go test helpers)
- Minimum supported EKS version is 1.31 (enforced in module validation)

### Testing Requirements
- Unit tests: `task test-unit` (no AWS, ~3s, 48 tests)
- Static analysis: `task lint` (fmt + validate + tflint + trivy)
- Integration tests: `task test-integration` (real AWS, ~25min, ~$0.20)
- Parallel version tests: `task test-integration` (deploys shared VPC, tests multiple EKS versions)
- CI pipeline: `task ci` (runs all non-AWS checks locally)

### Common Patterns
- Terraform modules use `terraform-aws-modules` community modules as upstream sources
- All resources tagged with `Environment`, `Terraform`, `Test`, `Owned=terratest`, `Pipeline`, `RunID`
- Integration tests use Terratest's `terraform.InitAndApply` / `terraform.Destroy` lifecycle
- Version validation uses regex: `^1\.(3[1-9]|[4-9][0-9])$`

## Dependencies

### External
- Terraform >= 1.6.0
- Go >= 1.21
- AWS provider >= 5.40.0
- `terraform-aws-modules/eks/aws` ~> 20.8
- `terraform-aws-modules/vpc/aws` ~> 5.5
- Terratest (gruntwork-io/terratest)
- Task (taskfile.dev) for command automation
- TFLint, Trivy for static analysis and security scanning
- cloud-nuke (gruntwork-io) for orphaned resource cleanup

<!-- MANUAL: -->
