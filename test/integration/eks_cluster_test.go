package test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/retry"
	"github.com/gruntwork-io/terratest/modules/terraform"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/aws-iam-authenticator/pkg/token"
)

const (
	testTimeout        = 30 * time.Minute
	nodeReadyTimeout   = 10 * time.Minute
	podReadyTimeout    = 5 * time.Minute
	retryInterval      = 30 * time.Second
	maxRetries         = 20
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
			"cluster_name":       clusterName,
			"aws_region":         awsRegion,
			"environment":        "terratest",
			"node_instance_types": []string{"t3.medium"},
			"node_desired_size":  2,
			"node_min_size":      1,
			"node_max_size":      3,
		},
		NoColor: true,
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
		validateClusterWithAWSSdk(t, awsRegion, actualClusterName)
	})

	t.Run("ValidateNodesReady", func(t *testing.T) {
		validateNodesReady(t, awsRegion, actualClusterName, clusterEndpoint, clusterCAData)
	})

	t.Run("ValidateTestWorkload", func(t *testing.T) {
		validateTestWorkload(t, awsRegion, actualClusterName, clusterEndpoint, clusterCAData)
	})
}

// validateClusterEndpoint checks that the cluster endpoint is accessible
func validateClusterEndpoint(t *testing.T, endpoint string) {
	assert.True(t, strings.HasPrefix(endpoint, "https://"), "Endpoint should be HTTPS")
	assert.Contains(t, endpoint, ".eks.amazonaws.com", "Endpoint should be an EKS endpoint")
}

// validateClusterWithAWSSdk validates the cluster exists via AWS SDK
func validateClusterWithAWSSdk(t *testing.T, region, clusterName string) {
	sess, err := session.NewSession(&aws.Config{
		Region: aws.String(region),
	})
	require.NoError(t, err, "Failed to create AWS session")

	eksSvc := eks.New(sess)

	_, err = retry.DoWithRetryE(t, "Describe EKS cluster", maxRetries, retryInterval, func() (string, error) {
		result, err := eksSvc.DescribeCluster(&eks.DescribeClusterInput{
			Name: aws.String(clusterName),
		})
		if err != nil {
			return "", err
		}

		status := aws.StringValue(result.Cluster.Status)
		if status != "ACTIVE" {
			return "", fmt.Errorf("cluster status is %s, waiting for ACTIVE", status)
		}

		return status, nil
	})

	require.NoError(t, err, "Cluster should be in ACTIVE state")
}

// validateNodesReady checks that worker nodes are ready
func validateNodesReady(t *testing.T, region, clusterName, endpoint, caData string) {
	clientset := getKubernetesClient(t, region, clusterName, endpoint, caData)

	_, err := retry.DoWithRetryE(t, "Wait for nodes to be ready", maxRetries, retryInterval, func() (string, error) {
		nodes, err := clientset.CoreV1().Nodes().List(context.Background(), metav1.ListOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to list nodes: %w", err)
		}

		if len(nodes.Items) == 0 {
			return "", fmt.Errorf("no nodes found")
		}

		readyCount := 0
		for _, node := range nodes.Items {
			for _, condition := range node.Status.Conditions {
				if condition.Type == corev1.NodeReady && condition.Status == corev1.ConditionTrue {
					readyCount++
					break
				}
			}
		}

		if readyCount == 0 {
			return "", fmt.Errorf("no nodes are ready yet")
		}

		return fmt.Sprintf("%d nodes ready", readyCount), nil
	})

	require.NoError(t, err, "At least one node should be ready")
}

// validateTestWorkload deploys a test pod and validates it runs successfully
func validateTestWorkload(t *testing.T, region, clusterName, endpoint, caData string) {
	clientset := getKubernetesClient(t, region, clusterName, endpoint, caData)

	namespace := "default"
	podName := fmt.Sprintf("terratest-pod-%s", strings.ToLower(random.UniqueId()))

	// Create test pod
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":     "terratest",
				"test":    "true",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    "nginx",
					Image:   "nginx:alpine",
					Ports: []corev1.ContainerPort{
						{
							ContainerPort: 80,
						},
					},
					Resources: corev1.ResourceRequirements{
						Requests: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("100m"),
							corev1.ResourceMemory: resource.MustParse("64Mi"),
						},
						Limits: corev1.ResourceList{
							corev1.ResourceCPU:    resource.MustParse("200m"),
							corev1.ResourceMemory: resource.MustParse("128Mi"),
						},
					},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}

	// Create the pod
	_, err := clientset.CoreV1().Pods(namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	require.NoError(t, err, "Failed to create test pod")

	// Cleanup pod after test
	defer func() {
		_ = clientset.CoreV1().Pods(namespace).Delete(context.Background(), podName, metav1.DeleteOptions{})
	}()

	// Wait for pod to be running
	_, err = retry.DoWithRetryE(t, "Wait for pod to be running", maxRetries, retryInterval, func() (string, error) {
		p, err := clientset.CoreV1().Pods(namespace).Get(context.Background(), podName, metav1.GetOptions{})
		if err != nil {
			return "", fmt.Errorf("failed to get pod: %w", err)
		}

		if p.Status.Phase != corev1.PodRunning {
			return "", fmt.Errorf("pod is in %s state, waiting for Running", p.Status.Phase)
		}

		return "pod running", nil
	})

	require.NoError(t, err, "Test pod should be running")
}

// getKubernetesClient creates a Kubernetes client for the EKS cluster
func getKubernetesClient(t *testing.T, region, clusterName, endpoint, caData string) *kubernetes.Clientset {
	// Decode CA data
	caBytes, err := base64.StdEncoding.DecodeString(caData)
	require.NoError(t, err, "Failed to decode CA data")

	// Get auth token using AWS IAM authenticator
	gen, err := token.NewGenerator(true, false)
	require.NoError(t, err, "Failed to create token generator")

	opts := &token.GetTokenOptions{
		ClusterID: clusterName,
		Region:    region,
	}

	tok, err := gen.GetWithOptions(context.Background(), opts)
	require.NoError(t, err, "Failed to get token")

	// Create Kubernetes client config
	config := &rest.Config{
		Host:        endpoint,
		BearerToken: tok.Token,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: caBytes,
		},
	}

	// Create clientset
	clientset, err := kubernetes.NewForConfig(config)
	require.NoError(t, err, "Failed to create Kubernetes clientset")

	return clientset
}

// getEnvWithDefault returns environment variable value or default
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
