// Self-contained EKS version matrix test. Deploys a shared VPC, discovers
// supported EKS versions from AWS, then runs parallel subtests — one per version.
// All cleanup is handled via defer (VPC destroy runs after all subtests complete).
//
// Remove this file if your project doesn't test multiple EKS versions.
package test

import (
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/gruntwork-io/terratest/modules/terraform"
)

const (
	versionMatrixTimeout = 55 * time.Minute
	versionTestVPCName   = "terratest-vpc"
)

// TestEksClusterVersionMatrix deploys a shared VPC, discovers EKS versions,
// and tests each version in parallel. No external env vars required (AWS creds only).
func TestEksClusterVersionMatrix(t *testing.T) {
	cfg := newTestConfig(t)

	vpcName := fmt.Sprintf("%s-%s", versionTestVPCName, cfg.UniqueID)

	t.Logf("VPC: %s | Region: %s | MinVersion: %s", vpcName, cfg.AWSRegion, cfg.MinVersion)
	t.Logf("Pipeline tags: %v", cfg.PipelineTags)

	// ── Step 1: Deploy shared VPC ──────────────────────────────────────────
	vpcDir := copyFixtureToTemp(t, "examples/vpc")
	vpcOpts := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
		TerraformDir: vpcDir,
		Vars: map[string]interface{}{
			"vpc_name":      vpcName,
			"aws_region":    cfg.AWSRegion,
			"environment":   "terratest",
			"pipeline_tags": cfg.PipelineTags,
		},
		NoColor:     true,
		Parallelism: 20,
	})

	// VPC destroy runs AFTER all parallel subtests complete (Go testing guarantee)
	defer terraform.Destroy(t, vpcOpts)
	terraform.InitAndApply(t, vpcOpts)

	vpcID := terraform.Output(t, vpcOpts, "vpc_id")
	privateSubnets := terraform.OutputList(t, vpcOpts, "private_subnets")

	t.Logf("VPC deployed: %s | Subnets: %v", vpcID, privateSubnets)

	// ── Step 2: Discover EKS versions ──────────────────────────────────────
	versions := discoverEKSVersions(t, cfg.AWSRegion, cfg.MinVersion)
	t.Logf("Discovered EKS versions: %v", versions)

	// ── Step 3: Parallel subtests per version ──────────────────────────────
	for _, v := range versions {
		version := v // capture loop variable
		t.Run("EKS_"+strings.ReplaceAll(version, ".", "_"), func(t *testing.T) {
			t.Parallel()

			versionSlug := strings.ReplaceAll(version, ".", "-")
			clusterName := fmt.Sprintf("test-eks-%s-%s", versionSlug, cfg.UniqueID)

			t.Logf("Testing EKS %s → cluster: %s", version, clusterName)

			// Each version gets its own temp dir (avoids state lock conflicts)
			eksDir := copyFixtureToTemp(t, "examples/eks")
			eksOpts := terraform.WithDefaultRetryableErrors(t, &terraform.Options{
				TerraformDir: eksDir,
				Vars: map[string]interface{}{
					"cluster_name":        clusterName,
					"cluster_version":     version,
					"aws_region":          cfg.AWSRegion,
					"vpc_id":              vpcID,
					"private_subnet_ids":  privateSubnets,
					"environment":         "terratest",
					"node_instance_types": []string{"t3.small"},
					"node_desired_size":   1,
					"node_min_size":       1,
					"node_max_size":       1,
					"pipeline_tags":       cfg.PipelineTags,
					"pipeline_run_hash":   "",
				},
				NoColor:     true,
				Parallelism: 20,
			})

			defer terraform.Destroy(t, eksOpts)
			terraform.InitAndApply(t, eksOpts)

			out := getEKSOutputs(t, eksOpts)
			out.validate(t, clusterName, version)

			validateClusterEndpoint(t, out.ClusterEndpoint)
			validateClusterStatus(t, cfg.AWSRegion, out.ClusterName, version)

			clientset := getKubernetesClient(t, cfg.AWSRegion, out.ClusterName, out.ClusterEndpoint, out.ClusterCAData)
			validateNodeReadiness(t, clientset)
		})
	}
}
