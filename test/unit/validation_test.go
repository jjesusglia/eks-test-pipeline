package unit

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestValidateClusterName(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid simple name",
			input:     "my-cluster",
			wantError: false,
		},
		{
			name:      "valid name with numbers",
			input:     "eks-cluster-123",
			wantError: false,
		},
		{
			name:      "valid name starting with number",
			input:     "1-cluster",
			wantError: false,
		},
		{
			name:      "empty name",
			input:     "",
			wantError: true,
			errorMsg:  "cluster name cannot be empty",
		},
		{
			name:      "name too long",
			input:     string(make([]byte, 101)), // 101 characters
			wantError: true,
			errorMsg:  "cluster name cannot exceed 100 characters",
		},
		{
			name:      "name with invalid characters",
			input:     "cluster_with_underscores",
			wantError: true,
			errorMsg:  "must contain only alphanumeric characters and hyphens",
		},
		{
			name:      "name starting with hyphen",
			input:     "-invalid",
			wantError: true,
			errorMsg:  "must start with a letter or number",
		},
		{
			name:      "name with spaces",
			input:     "my cluster",
			wantError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateClusterName(tt.input)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateKubernetesVersion(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantError bool
	}{
		{"valid version 1.29", "1.29", false},
		{"valid version 1.28", "1.28", false},
		{"valid version 1.5", "1.5", false},
		{"empty version", "", true},
		{"invalid format - no minor", "1", true},
		{"invalid format - three parts", "1.29.0", true},
		{"invalid major version", "2.0", true},
		{"invalid characters", "1.2a", true},
		{"with v prefix", "v1.29", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateKubernetesVersion(tt.input)

			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateSubnetCount(t *testing.T) {
	tests := []struct {
		name        string
		subnets     []string
		minRequired int
		wantError   bool
		errorMsg    string
	}{
		{
			name:        "sufficient subnets",
			subnets:     []string{"subnet-1", "subnet-2", "subnet-3"},
			minRequired: 2,
			wantError:   false,
		},
		{
			name:        "exact minimum",
			subnets:     []string{"subnet-1", "subnet-2"},
			minRequired: 2,
			wantError:   false,
		},
		{
			name:        "insufficient subnets",
			subnets:     []string{"subnet-1"},
			minRequired: 2,
			wantError:   true,
			errorMsg:    "at least 2 subnets required",
		},
		{
			name:        "empty subnet in list",
			subnets:     []string{"subnet-1", "", "subnet-3"},
			minRequired: 2,
			wantError:   true,
			errorMsg:    "subnet at index 1 is empty",
		},
		{
			name:        "whitespace only subnet",
			subnets:     []string{"subnet-1", "   ", "subnet-3"},
			minRequired: 2,
			wantError:   true,
			errorMsg:    "subnet at index 1 is empty",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateSubnetCount(tt.subnets, tt.minRequired)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateTags(t *testing.T) {
	tests := []struct {
		name         string
		tags         map[string]string
		requiredTags []string
		wantError    bool
		errorMsg     string
	}{
		{
			name: "all required tags present",
			tags: map[string]string{
				"Environment": "production",
				"Team":        "platform",
				"Owner":       "ops",
			},
			requiredTags: []string{"Environment", "Team"},
			wantError:    false,
		},
		{
			name: "missing required tag",
			tags: map[string]string{
				"Environment": "production",
			},
			requiredTags: []string{"Environment", "Team"},
			wantError:    true,
			errorMsg:     "required tag 'Team' is missing",
		},
		{
			name:         "nil tags map",
			tags:         nil,
			requiredTags: []string{"Environment"},
			wantError:    true,
			errorMsg:     "tags map cannot be nil",
		},
		{
			name: "no required tags",
			tags: map[string]string{
				"Environment": "production",
			},
			requiredTags: []string{},
			wantError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateTags(tt.tags, tt.requiredTags)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestValidateInstanceTypes(t *testing.T) {
	tests := []struct {
		name      string
		input     []string
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid t3 instance",
			input:     []string{"t3.medium"},
			wantError: false,
		},
		{
			name:      "multiple valid instances",
			input:     []string{"t3.medium", "m5.large", "c5.xlarge"},
			wantError: false,
		},
		{
			name:      "empty list",
			input:     []string{},
			wantError: true,
			errorMsg:  "at least one instance type must be specified",
		},
		{
			name:      "invalid format - uppercase",
			input:     []string{"T3.MEDIUM"},
			wantError: true,
			errorMsg:  "invalid instance type format",
		},
		{
			name:      "invalid format - no dot",
			input:     []string{"t3medium"},
			wantError: true,
			errorMsg:  "invalid instance type format",
		},
		{
			name:      "invalid format - special characters",
			input:     []string{"t3_medium"},
			wantError: true,
			errorMsg:  "invalid instance type format",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateInstanceTypes(tt.input)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestGenerateClusterTags(t *testing.T) {
	tests := []struct {
		name        string
		defaultTags map[string]string
		customTags  map[string]string
		expected    map[string]string
	}{
		{
			name: "merge tags without conflicts",
			defaultTags: map[string]string{
				"ManagedBy": "terraform",
				"Project":   "eks",
			},
			customTags: map[string]string{
				"Environment": "production",
				"Team":        "platform",
			},
			expected: map[string]string{
				"ManagedBy":   "terraform",
				"Project":     "eks",
				"Environment": "production",
				"Team":        "platform",
			},
		},
		{
			name: "custom tags override defaults",
			defaultTags: map[string]string{
				"Environment": "development",
				"ManagedBy":   "terraform",
			},
			customTags: map[string]string{
				"Environment": "production",
			},
			expected: map[string]string{
				"Environment": "production",
				"ManagedBy":   "terraform",
			},
		},
		{
			name:        "nil default tags",
			defaultTags: nil,
			customTags: map[string]string{
				"Environment": "production",
			},
			expected: map[string]string{
				"Environment": "production",
			},
		},
		{
			name: "nil custom tags",
			defaultTags: map[string]string{
				"ManagedBy": "terraform",
			},
			customTags: nil,
			expected: map[string]string{
				"ManagedBy": "terraform",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GenerateClusterTags(tt.defaultTags, tt.customTags)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestValidateNodeGroupSize(t *testing.T) {
	tests := []struct {
		name      string
		min       int
		max       int
		desired   int
		wantError bool
		errorMsg  string
	}{
		{
			name:      "valid configuration",
			min:       1,
			max:       5,
			desired:   3,
			wantError: false,
		},
		{
			name:      "desired equals min",
			min:       2,
			max:       5,
			desired:   2,
			wantError: false,
		},
		{
			name:      "desired equals max",
			min:       1,
			max:       3,
			desired:   3,
			wantError: false,
		},
		{
			name:      "negative min",
			min:       -1,
			max:       5,
			desired:   3,
			wantError: true,
			errorMsg:  "min size cannot be negative",
		},
		{
			name:      "max less than min",
			min:       5,
			max:       3,
			desired:   4,
			wantError: true,
			errorMsg:  "max size (3) cannot be less than min size (5)",
		},
		{
			name:      "desired less than min",
			min:       3,
			max:       5,
			desired:   2,
			wantError: true,
			errorMsg:  "desired size (2) cannot be less than min size (3)",
		},
		{
			name:      "desired greater than max",
			min:       1,
			max:       3,
			desired:   5,
			wantError: true,
			errorMsg:  "desired size (5) cannot exceed max size (3)",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateNodeGroupSize(tt.min, tt.max, tt.desired)

			if tt.wantError {
				assert.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
