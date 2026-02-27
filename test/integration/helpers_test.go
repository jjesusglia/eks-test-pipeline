// Shared test helpers for integration tests. Each project writes their own helpers following this pattern.
package test

import (
	"context"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
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

// discoverEKSVersions queries AWS for supported EKS versions >= minVersion.
// Uses the vpc-cni addon compatibility list as the source of truth.
func discoverEKSVersions(t *testing.T, region, minVersion string) []string {
	t.Helper()

	sess, err := session.NewSessionWithOptions(session.Options{
		Config:            aws.Config{Region: aws.String(region)},
		SharedConfigState: session.SharedConfigEnable,
	})
	require.NoError(t, err, "Failed to create AWS session")

	eksSvc := eks.New(sess)

	input := &eks.DescribeAddonVersionsInput{
		AddonName: aws.String("vpc-cni"),
	}

	result, err := eksSvc.DescribeAddonVersions(input)
	require.NoError(t, err, "Failed to describe addon versions")

	// Collect unique cluster versions
	versionSet := make(map[string]bool)
	for _, addon := range result.Addons {
		for _, addonVersion := range addon.AddonVersions {
			for _, compat := range addonVersion.Compatibilities {
				if compat.ClusterVersion != nil {
					versionSet[*compat.ClusterVersion] = true
				}
			}
		}
	}

	// Filter >= minVersion and sort
	var versions []string
	for v := range versionSet {
		if compareVersions(v, minVersion) >= 0 {
			versions = append(versions, v)
		}
	}

	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i], versions[j]) < 0
	})

	require.NotEmpty(t, versions, "No EKS versions found >= %s", minVersion)

	return versions
}

// compareVersions compares two dotted version strings (e.g. "1.31" vs "1.32").
// Returns -1, 0, or 1.
func compareVersions(a, b string) int {
	aParts := strings.Split(a, ".")
	bParts := strings.Split(b, ".")

	maxLen := len(aParts)
	if len(bParts) > maxLen {
		maxLen = len(bParts)
	}

	for i := 0; i < maxLen; i++ {
		var aNum, bNum int
		if i < len(aParts) {
			aNum, _ = strconv.Atoi(aParts[i])
		}
		if i < len(bParts) {
			bNum, _ = strconv.Atoi(bParts[i])
		}
		if aNum < bNum {
			return -1
		}
		if aNum > bNum {
			return 1
		}
	}

	return 0
}

// copyFixtureToTemp copies a Terraform fixture directory to a temp dir,
// rewriting relative module source paths to absolute. This allows parallel
// Terraform runs without state lock conflicts.
func copyFixtureToTemp(t *testing.T, fixtureRelPath string) string {
	t.Helper()

	// Resolve fixture path relative to repo root (tests run from test/integration/)
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	require.NoError(t, err, "Failed to resolve repo root")

	fixtureSrc := filepath.Join(repoRoot, fixtureRelPath)

	// Create temp directory (auto-cleaned by t.TempDir())
	tmpDir := t.TempDir()

	entries, err := os.ReadDir(fixtureSrc)
	require.NoError(t, err, "Failed to read fixture directory: %s", fixtureSrc)

	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".tf") {
			continue
		}

		srcPath := filepath.Join(fixtureSrc, entry.Name())
		content, err := os.ReadFile(srcPath)
		require.NoError(t, err, "Failed to read %s", srcPath)

		// Rewrite relative module source paths to absolute
		contentStr := rewriteModuleSources(string(content), fixtureSrc)

		dstPath := filepath.Join(tmpDir, entry.Name())
		err = os.WriteFile(dstPath, []byte(contentStr), 0644)
		require.NoError(t, err, "Failed to write %s", dstPath)
	}

	return tmpDir
}

// rewriteModuleSources replaces relative source paths (source = "../..") with
// absolute paths resolved from the original fixture directory.
func rewriteModuleSources(content, fixtureDir string) string {
	re := regexp.MustCompile(`(source\s*=\s*")(\.\.[^"]*)(")`)
	return re.ReplaceAllStringFunc(content, func(match string) string {
		parts := re.FindStringSubmatch(match)
		relPath := parts[2]
		absPath, _ := filepath.Abs(filepath.Join(fixtureDir, relPath))
		return parts[1] + absPath + parts[3]
	})
}

// getPipelineTags generates pipeline tags for resource identification and cleanup.
func getPipelineTags(projectName string) map[string]string {
	runID := os.Getenv("PIPELINE_RUN_ID")

	environment := "local"
	if ghRunID := os.Getenv("GITHUB_RUN_ID"); ghRunID != "" {
		environment = "ci"
		if runID == "" {
			runID = ghRunID
		}
	}

	if runID == "" {
		runID = fmt.Sprintf("local-%s", time.Now().Format("20060102-150405"))
	}

	return map[string]string{
		"Pipeline":    projectName,
		"RunID":       runID,
		"Environment": environment,
	}
}

// testConfig centralizes environment variable lookups and shared test setup.
type testConfig struct {
	AWSRegion    string
	AWSProfile   string
	ProjectName  string
	MinVersion   string
	PipelineTags map[string]string
	UniqueID     string
}

// newTestConfig creates a testConfig, skipping in short mode.
// Defaults AWS_PROFILE to "sandbox" for local development.
// CI uses OIDC credentials so no profile is needed there.
func newTestConfig(t *testing.T) *testConfig {
	t.Helper()
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Default to "sandbox" profile for local runs.
	// In CI (GITHUB_RUN_ID set), OIDC handles auth — no profile needed.
	awsProfile := os.Getenv("AWS_PROFILE")
	if awsProfile == "" && os.Getenv("GITHUB_RUN_ID") == "" {
		awsProfile = "sandbox"
	}
	if awsProfile != "" {
		t.Setenv("AWS_PROFILE", awsProfile)
	}

	projectName := getEnvWithDefault("PROJECT_NAME", "eks-cluster")
	return &testConfig{
		AWSRegion:    getEnvWithDefault("AWS_REGION", "us-west-1"),
		AWSProfile:   awsProfile,
		ProjectName:  projectName,
		MinVersion:   getEnvWithDefault("MIN_EKS_VERSION", "1.31"),
		PipelineTags: getPipelineTags(projectName),
		UniqueID:     strings.ToLower(random.UniqueId()),
	}
}

// eksOutputs holds the standard Terraform outputs from an EKS deployment.
type eksOutputs struct {
	ClusterEndpoint string
	ClusterCAData   string
	ClusterName     string
	ClusterVersion  string
}

// getEKSOutputs retrieves the four standard EKS outputs from Terraform.
func getEKSOutputs(t *testing.T, opts *terraform.Options) *eksOutputs {
	t.Helper()
	return &eksOutputs{
		ClusterEndpoint: terraform.Output(t, opts, "cluster_endpoint"),
		ClusterCAData:   terraform.Output(t, opts, "cluster_certificate_authority_data"),
		ClusterName:     terraform.Output(t, opts, "cluster_name"),
		ClusterVersion:  terraform.Output(t, opts, "cluster_version"),
	}
}

// validate asserts that the EKS outputs are non-empty and match expected values.
func (o *eksOutputs) validate(t *testing.T, expectedName, expectedVersionPrefix string) {
	t.Helper()
	assert.NotEmpty(t, o.ClusterEndpoint, "Cluster endpoint should not be empty")
	assert.NotEmpty(t, o.ClusterCAData, "Cluster CA data should not be empty")
	assert.Equal(t, expectedName, o.ClusterName, "Cluster name should match")
	assert.True(t, strings.HasPrefix(o.ClusterVersion, expectedVersionPrefix),
		"Cluster version should match %s, got %s", expectedVersionPrefix, o.ClusterVersion)
}
