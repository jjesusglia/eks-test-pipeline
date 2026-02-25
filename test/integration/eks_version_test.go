// REFERENCE IMPLEMENTATION: Only needed for parallel version testing.
// Remove this file if your project doesn't test multiple versions.
package test

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

const (
	versionTestTimeout = 30 * time.Minute
	versionNodeTimeout = 10 * time.Minute
)

// TestEksClusterVersioned tests EKS deployment with a specific version using pre-deployed VPC.
// This test is designed to run in parallel via GitHub Actions matrix strategy.
// It expects the following environment variables:
//   - TF_VAR_vpc_id: VPC ID from shared VPC deployment
//   - TF_VAR_private_subnet_ids: JSON-encoded list of private subnet IDs
//   - TF_VAR_cluster_version: EKS version to test (e.g., "1.31")
//   - TF_VAR_pipeline_tags: Pipeline tags for resource identification (auto-injected by scripts/terraform.sh)
func TestEksClusterVersioned(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Check required environment variables
	// Supports both naming conventions:
	//   TF_VAR_private_subnet_ids  — set explicitly by CI or user
	//   TF_VAR_private_subnets     — auto-loaded from VPC layer outputs
	vpcID := os.Getenv("TF_VAR_vpc_id")
	privateSubnetsJSON := os.Getenv("TF_VAR_private_subnet_ids")
	if privateSubnetsJSON == "" {
		privateSubnetsJSON = os.Getenv("TF_VAR_private_subnets")
	}
	clusterVersion := os.Getenv("TF_VAR_cluster_version")

	if vpcID == "" {
		t.Skip("TF_VAR_vpc_id not set, skipping versioned EKS test")
	}
	if privateSubnetsJSON == "" {
		t.Skip("TF_VAR_private_subnet_ids / TF_VAR_private_subnets not set, skipping versioned EKS test")
	}
	if clusterVersion == "" {
		clusterVersion = "1.31" // Default for local testing
	}

	t.Parallel()

	// Generate unique cluster name with version
	// Keep short — the fixture appends pipeline_run_hash for pipeline isolation
	// IAM role name_prefix limit is 38 chars, EKS module adds "-cluster-" suffix
	versionNormalized := strings.ReplaceAll(clusterVersion, ".", "-")
	clusterName := fmt.Sprintf("test-eks-%s", versionNormalized)
	awsRegion := getEnvWithDefault("AWS_REGION", "us-west-1")

	t.Logf("Testing EKS version %s with cluster name: %s", clusterVersion, clusterName)

	// Path to the EKS-only fixture
	examplesDir := filepath.Join("../..", "examples", "eks")

	// Parse private subnet IDs from JSON
	privateSubnets := parseSubnetIDs(t, privateSubnetsJSON)

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: examplesDir,
		Vars: map[string]interface{}{
			"cluster_name":        clusterName,
			"cluster_version":     clusterVersion,
			"aws_region":          awsRegion,
			"vpc_id":              vpcID,
			"private_subnet_ids":  privateSubnets,
			"environment":         "terratest",
			"node_instance_types": []string{"t3.small"},
			"node_desired_size":   1,
			"node_min_size":       1,
			"node_max_size":       1,
		},
		NoColor:     true,
		Parallelism: 20,
	})

	// Ensure cleanup happens regardless of test outcome
	defer terraform.Destroy(t, terraformOptions)

	// Deploy EKS cluster
	terraform.InitAndApply(t, terraformOptions)

	// Retrieve outputs
	clusterEndpoint := terraform.Output(t, terraformOptions, "cluster_endpoint")
	clusterCAData := terraform.Output(t, terraformOptions, "cluster_certificate_authority_data")
	actualClusterName := terraform.Output(t, terraformOptions, "cluster_name")
	actualVersion := terraform.Output(t, terraformOptions, "cluster_version")

	// Validate basic outputs
	assert.NotEmpty(t, clusterEndpoint, "Cluster endpoint should not be empty")
	assert.NotEmpty(t, clusterCAData, "Cluster CA data should not be empty")
	assert.Equal(t, clusterName, actualClusterName, "Cluster name should match")
	assert.True(t, strings.HasPrefix(actualVersion, clusterVersion),
		"Cluster version should match requested version %s, got %s", clusterVersion, actualVersion)

	// Run sub-tests
	t.Run("ValidateClusterEndpoint", func(t *testing.T) {
		validateClusterEndpoint(t, clusterEndpoint)
	})

	t.Run("ValidateClusterWithAWSSdk", func(t *testing.T) {
		validateClusterStatus(t, awsRegion, actualClusterName, clusterVersion)
	})

	// Create Kubernetes client for node validation
	clientset := getKubernetesClient(t, awsRegion, actualClusterName, clusterEndpoint, clusterCAData)

	t.Run("ValidateNodesReady", func(t *testing.T) {
		validateNodeReadiness(t, clientset)
	})
}

// parseSubnetIDs parses a JSON array of subnet IDs
func parseSubnetIDs(t *testing.T, jsonStr string) []string {
	// Remove JSON array brackets and quotes, split by comma
	cleaned := strings.Trim(jsonStr, "[]")
	cleaned = strings.ReplaceAll(cleaned, "\"", "")
	cleaned = strings.ReplaceAll(cleaned, " ", "")

	if cleaned == "" {
		t.Fatal("No subnet IDs provided")
	}

	return strings.Split(cleaned, ",")
}
