// REFERENCE IMPLEMENTATION: Replace with tests for your module.
//
// TEST PATTERN:
// 1. Configure terraform options (module path, vars, env)
// 2. Defer terraform.Destroy for cleanup
// 3. terraform.InitAndApply
// 4. Validate outputs (terraform.Output)
// 5. Validate infrastructure (AWS SDK calls)
// 6. Optional: validate workload (k8s pod deployment)
package test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
)

const (
	testTimeout      = 30 * time.Minute
	nodeReadyTimeout = 10 * time.Minute
	podReadyTimeout  = 5 * time.Minute
)

// TestEksClusterComplete runs a full integration test deploying VPC and EKS
func TestEksClusterComplete(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	t.Parallel()

	// Generate a unique cluster name to avoid conflicts
	uniqueID := strings.ToLower(random.UniqueId())
	clusterName := fmt.Sprintf("terratest-%s", uniqueID)
	awsRegion := getEnvWithDefault("AWS_REGION", "us-west-1")

	// Path to the Terraform example
	examplesDir := filepath.Join("../..", "examples", "complete")

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: examplesDir,
		Vars: map[string]interface{}{
			"cluster_name":        clusterName,
			"aws_region":          awsRegion,
			"environment":         "terratest",
			"node_instance_types": []string{"t3.small"}, // Smaller instance sufficient for tests
			"node_desired_size":   1,                    // Single node speeds up readiness
			"node_min_size":       1,
			"node_max_size":       1,
		},
		NoColor:     true,
		Parallelism: 20, // Higher parallelism for faster VPC resource creation
	})

	// Ensure cleanup happens regardless of test outcome
	defer terraform.Destroy(t, terraformOptions)

	// Deploy infrastructure
	terraform.InitAndApply(t, terraformOptions)

	// Retrieve outputs
	clusterEndpoint := terraform.Output(t, terraformOptions, "cluster_endpoint")
	clusterCAData := terraform.Output(t, terraformOptions, "cluster_certificate_authority_data")
	actualClusterName := terraform.Output(t, terraformOptions, "cluster_name")
	clusterVersion := terraform.Output(t, terraformOptions, "cluster_version")

	// Validate basic outputs
	assert.NotEmpty(t, clusterEndpoint, "Cluster endpoint should not be empty")
	assert.NotEmpty(t, clusterCAData, "Cluster CA data should not be empty")
	assert.Equal(t, clusterName, actualClusterName, "Cluster name should match")
	assert.True(t, strings.HasPrefix(clusterVersion, "1."), "Cluster version should start with 1.")

	// Run sub-tests
	t.Run("ValidateClusterEndpoint", func(t *testing.T) {
		validateClusterEndpoint(t, clusterEndpoint)
	})

	t.Run("ValidateClusterWithAWSSdk", func(t *testing.T) {
		validateClusterStatus(t, awsRegion, actualClusterName, "")
	})

	// Create Kubernetes client once for all k8s-based validations (avoids duplicate IAM token generation)
	clientset := getKubernetesClient(t, awsRegion, actualClusterName, clusterEndpoint, clusterCAData)

	t.Run("ValidateNodesReady", func(t *testing.T) {
		validateNodeReadiness(t, clientset)
	})

	t.Run("ValidateTestWorkload", func(t *testing.T) {
		validateWorkloadDeployment(t, clientset)
	})
}
