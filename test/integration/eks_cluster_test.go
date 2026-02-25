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
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
)

const (
	testTimeout      = 30 * time.Minute
	nodeReadyTimeout = 10 * time.Minute
	podReadyTimeout  = 5 * time.Minute
)

// TestEksClusterComplete runs a full integration test deploying VPC and EKS
func TestEksClusterComplete(t *testing.T) {
	cfg := newTestConfig(t)
	t.Parallel()

	clusterName := fmt.Sprintf("terratest-%s", cfg.UniqueID)
	examplesDir := filepath.Join("../..", "examples", "complete")

	terraformOptions := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: examplesDir,
		Vars: map[string]interface{}{
			"cluster_name":        clusterName,
			"aws_region":          cfg.AWSRegion,
			"environment":         "terratest",
			"node_instance_types": []string{"t3.small"},
			"node_desired_size":   1,
			"node_min_size":       1,
			"node_max_size":       1,
			"pipeline_tags":       cfg.PipelineTags,
		},
		NoColor:     true,
		Parallelism: 20,
	})

	defer terraform.Destroy(t, terraformOptions)
	terraform.InitAndApply(t, terraformOptions)

	out := getEKSOutputs(t, terraformOptions)
	out.validate(t, clusterName, "1.")

	t.Run("ValidateClusterEndpoint", func(t *testing.T) {
		validateClusterEndpoint(t, out.ClusterEndpoint)
	})

	t.Run("ValidateClusterWithAWSSdk", func(t *testing.T) {
		validateClusterStatus(t, cfg.AWSRegion, out.ClusterName, "")
	})

	clientset := getKubernetesClient(t, cfg.AWSRegion, out.ClusterName, out.ClusterEndpoint, out.ClusterCAData)

	t.Run("ValidateNodesReady", func(t *testing.T) {
		validateNodeReadiness(t, clientset)
	})

	t.Run("ValidateTestWorkload", func(t *testing.T) {
		validateWorkloadDeployment(t, clientset)
	})
}
