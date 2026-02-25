<!-- Parent: ../AGENTS.md -->
<!-- Generated: 2026-02-19 | Updated: 2026-02-19 -->

# unit

## Purpose
Pure unit tests for validation and helper functions. No AWS credentials required. Runs in ~3 seconds with comprehensive table-driven test coverage.

## Key Files

| File | Description |
|------|-------------|
| `validation.go` | Validation functions: cluster name, K8s version, subnet count, tags, instance types, node group sizing |
| `validation_test.go` | Table-driven tests for all validation functions (48 test cases) |

## For AI Agents

### Working In This Directory
- Package is `unit` (separate from integration tests)
- All functions are pure Go with no external service dependencies
- Run: `cd test && go test -v ./unit/...`

### Testing Requirements
- Tests must pass without AWS credentials
- Add table-driven test cases for any new validation function
- Use `testify/assert` for assertions (not `require` - let all cases run)
- Follow existing pattern: struct with `name`, `input`, `wantError`, `errorMsg`

### Validation Functions Available
- `ValidateClusterName(name)` - EKS naming rules (1-100 chars, alphanumeric + hyphens)
- `ValidateKubernetesVersion(version)` - Format `1.XX`
- `ValidateSubnetCount(subnets, min)` - Minimum subnet count with empty-string checks
- `ValidateTags(tags, required)` - Required tag presence validation
- `ValidateInstanceTypes(types)` - EC2 instance type format (e.g., `t3.medium`)
- `ValidateNodeGroupSize(min, max, desired)` - ASG sizing constraints
- `GenerateClusterTags(default, custom)` - Tag merging with custom override

### Common Patterns
- Table-driven tests with `for _, tt := range tests { t.Run(...) }`
- Error message substring matching via `assert.Contains(t, err.Error(), ...)`
- Boundary testing (empty strings, nil maps, negative numbers, edge limits)

## Dependencies

### External
- `github.com/stretchr/testify/assert` - Test assertions

<!-- MANUAL: -->
