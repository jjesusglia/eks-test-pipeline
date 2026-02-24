// Shared test helpers for integration tests. Each project writes their own helpers following this pattern.
package test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/eks"
	"github.com/gruntwork-io/terratest/modules/random"
	"github.com/gruntwork-io/terratest/modules/retry"
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
	sharedRetryInterval = 10 * time.Second
	sharedMaxRetries    = 30
)

// getEnvWithDefault returns environment variable value or default.
func getEnvWithDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getKubernetesClient creates a Kubernetes client for the given EKS cluster.
func getKubernetesClient(t *testing.T, region, clusterName, endpoint, caData string) *kubernetes.Clientset {
	t.Helper()

	caBytes, err := base64.StdEncoding.DecodeString(caData)
	require.NoError(t, err, "Failed to decode CA data")

	gen, err := token.NewGenerator(true, false)
	require.NoError(t, err, "Failed to create token generator")

	opts := &token.GetTokenOptions{
		ClusterID: clusterName,
		Region:    region,
	}

	tok, err := gen.GetWithOptions(context.Background(), opts)
	require.NoError(t, err, "Failed to get token")

	config := &rest.Config{
		Host:        endpoint,
		BearerToken: tok.Token,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: caBytes,
		},
	}

	clientset, err := kubernetes.NewForConfig(config)
	require.NoError(t, err, "Failed to create Kubernetes clientset")

	return clientset
}

// validateClusterEndpoint checks that the cluster endpoint uses HTTPS and is an EKS endpoint.
func validateClusterEndpoint(t *testing.T, endpoint string) {
	t.Helper()
	assert.True(t, strings.HasPrefix(endpoint, "https://"), "Endpoint should be HTTPS")
	assert.Contains(t, endpoint, ".eks.amazonaws.com", "Endpoint should be an EKS endpoint")
}

// validateClusterStatus validates the cluster exists via the AWS SDK and is ACTIVE.
// If expectedVersion is non-empty, it also asserts the cluster version starts with that prefix.
func validateClusterStatus(t *testing.T, region, clusterName, expectedVersion string) {
	t.Helper()

	sess, err := session.NewSessionWithOptions(session.Options{
		Config: aws.Config{
			Region: aws.String(region),
		},
		SharedConfigState: session.SharedConfigEnable,
	})
	require.NoError(t, err, "Failed to create AWS session")

	eksSvc := eks.New(sess)

	var actualVersion string
	_, err = retry.DoWithRetryE(t, "Describe EKS cluster", sharedMaxRetries, sharedRetryInterval, func() (string, error) {
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

		actualVersion = aws.StringValue(result.Cluster.Version)
		return status, nil
	})

	require.NoError(t, err, "Cluster should be in ACTIVE state")

	if expectedVersion != "" {
		assert.True(t, strings.HasPrefix(actualVersion, expectedVersion),
			"Cluster version from AWS SDK should match expected %s, got %s", expectedVersion, actualVersion)
	}
}

// validateNodeReadiness checks that at least one worker node is Ready.
func validateNodeReadiness(t *testing.T, clientset *kubernetes.Clientset) {
	t.Helper()

	_, err := retry.DoWithRetryE(t, "Wait for nodes to be ready", sharedMaxRetries, sharedRetryInterval, func() (string, error) {
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

// validateWorkloadDeployment deploys a test nginx pod and waits for it to reach Running state.
func validateWorkloadDeployment(t *testing.T, clientset *kubernetes.Clientset) {
	t.Helper()

	namespace := "default"
	podName := fmt.Sprintf("terratest-pod-%s", strings.ToLower(random.UniqueId()))

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      podName,
			Namespace: namespace,
			Labels: map[string]string{
				"app":  "terratest",
				"test": "true",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:  "nginx",
					Image: "nginx:alpine",
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

	_, err := clientset.CoreV1().Pods(namespace).Create(context.Background(), pod, metav1.CreateOptions{})
	require.NoError(t, err, "Failed to create test pod")

	defer func() {
		_ = clientset.CoreV1().Pods(namespace).Delete(context.Background(), podName, metav1.DeleteOptions{})
	}()

	_, err = retry.DoWithRetryE(t, "Wait for pod to be running", sharedMaxRetries, sharedRetryInterval, func() (string, error) {
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
