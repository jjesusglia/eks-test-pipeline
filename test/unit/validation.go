package unit

import (
	"fmt"
	"regexp"
	"strings"
)

// ValidateClusterName checks if a cluster name is valid for EKS
// EKS cluster names must be 1-100 characters, alphanumeric plus hyphens
func ValidateClusterName(name string) error {
	if name == "" {
		return fmt.Errorf("cluster name cannot be empty")
	}

	if len(name) > 100 {
		return fmt.Errorf("cluster name cannot exceed 100 characters, got %d", len(name))
	}

	// EKS cluster names: alphanumeric and hyphens only, must start with letter/number
	validNamePattern := regexp.MustCompile(`^[a-zA-Z0-9][a-zA-Z0-9-]*$`)
	if !validNamePattern.MatchString(name) {
		return fmt.Errorf("cluster name must contain only alphanumeric characters and hyphens, and must start with a letter or number")
	}

	return nil
}

// ValidateKubernetesVersion checks if a Kubernetes version is valid format
func ValidateKubernetesVersion(version string) error {
	if version == "" {
		return fmt.Errorf("kubernetes version cannot be empty")
	}

	// Version format: 1.XX
	validVersionPattern := regexp.MustCompile(`^1\.\d{1,2}$`)
	if !validVersionPattern.MatchString(version) {
		return fmt.Errorf("kubernetes version must be in format 1.XX (e.g., 1.29)")
	}

	return nil
}

// ValidateSubnetCount checks if the minimum subnet count requirement is met
func ValidateSubnetCount(subnets []string, minRequired int) error {
	if len(subnets) < minRequired {
		return fmt.Errorf("at least %d subnets required for high availability, got %d", minRequired, len(subnets))
	}

	// Check for empty subnet IDs
	for i, subnet := range subnets {
		if strings.TrimSpace(subnet) == "" {
			return fmt.Errorf("subnet at index %d is empty", i)
		}
	}

	return nil
}

// ValidateTags checks if required tags are present
func ValidateTags(tags map[string]string, requiredTags []string) error {
	if tags == nil {
		return fmt.Errorf("tags map cannot be nil")
	}

	for _, required := range requiredTags {
		if _, exists := tags[required]; !exists {
			return fmt.Errorf("required tag '%s' is missing", required)
		}
	}

	return nil
}

// ValidateInstanceTypes checks if instance types are valid AWS EC2 types
func ValidateInstanceTypes(instanceTypes []string) error {
	if len(instanceTypes) == 0 {
		return fmt.Errorf("at least one instance type must be specified")
	}

	// Basic format check for instance types (e.g., t3.medium, m5.large)
	validInstancePattern := regexp.MustCompile(`^[a-z][a-z0-9]+\.[a-z0-9]+$`)

	for _, instanceType := range instanceTypes {
		if !validInstancePattern.MatchString(instanceType) {
			return fmt.Errorf("invalid instance type format: %s", instanceType)
		}
	}

	return nil
}

// GenerateClusterTags merges default tags with custom tags
func GenerateClusterTags(defaultTags, customTags map[string]string) map[string]string {
	result := make(map[string]string)

	// Add default tags first
	for k, v := range defaultTags {
		result[k] = v
	}

	// Override with custom tags
	for k, v := range customTags {
		result[k] = v
	}

	return result
}

// ValidateNodeGroupSize validates min/max/desired node group configuration
func ValidateNodeGroupSize(min, max, desired int) error {
	if min < 0 {
		return fmt.Errorf("min size cannot be negative, got %d", min)
	}

	if max < min {
		return fmt.Errorf("max size (%d) cannot be less than min size (%d)", max, min)
	}

	if desired < min {
		return fmt.Errorf("desired size (%d) cannot be less than min size (%d)", desired, min)
	}

	if desired > max {
		return fmt.Errorf("desired size (%d) cannot exceed max size (%d)", desired, max)
	}

	return nil
}
