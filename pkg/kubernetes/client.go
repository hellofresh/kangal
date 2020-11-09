package kubernetes

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.uber.org/zap"

	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restClient "k8s.io/client-go/rest"

	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned/typed/loadtest/v1"
)

var (
	loadTestMasterLabelSelector = "app=loadtest-master"
	loadTestWorkerLabelSelector = "app=loadtest-worker-pod"
	// GracePeriod is duration in seconds before the object should be deleted.
	// The value zero indicates delete immediately.
	gracePeriod = int64(0)
)

//Client manages calls to Kubernetes API
type Client struct {
	ltClient   loadTestV1.LoadTestInterface
	kubeClient kubernetes.Interface
	logger     *zap.Logger
}

// ListOptions is options to find load tests.
type ListOptions struct {
	// List of tags.
	Tags map[string]string
	// Limit.
	Limit int64
	// Continue.
	Continue string
}

//NewClient creates new Kubernetes client
func NewClient(loadTestClient loadTestV1.LoadTestInterface, kubeClient kubernetes.Interface, logger *zap.Logger) *Client {
	return &Client{
		ltClient:   loadTestClient,
		kubeClient: kubeClient,
		logger:     logger,
	}
}

// GetLoadTestsByLabel lists the load test from given load test labels
func (c *Client) GetLoadTestsByLabel(ctx context.Context, loadTest *apisLoadTestV1.LoadTest) (*apisLoadTestV1.LoadTestList, error) {
	fileHashLabel := loadTest.Labels["test-file-hash"]
	opts := metaV1.ListOptions{
		LabelSelector: "test-file-hash=" + fileHashLabel,
	}
	labeledLoadTests, err := c.ltClient.List(ctx, opts)
	if err != nil {
		c.logger.Error("Error on getting the list of created load tests with the given hash", zap.String("hash-label", loadTest.Labels["testFileHash"]), zap.Error(err))
		return nil, err
	}

	return labeledLoadTests, nil
}

// CreateLoadTest creates new load test from given request data
func (c *Client) CreateLoadTest(ctx context.Context, loadTest *apisLoadTestV1.LoadTest) (string, error) {
	c.logger.Debug("Creating load test CR ...")

	result, err := c.ltClient.Create(ctx, loadTest, metaV1.CreateOptions{})
	if err != nil {
		c.logger.Error("Error on creating new load test", zap.String("loadtest", loadTest.Name), zap.Error(err))
		return "", err
	}
	c.logger.Info("Created load test ", zap.String("loadtest", result.GetObjectMeta().GetName()))

	return result.GetObjectMeta().GetName(), err
}

// DeleteLoadTest deletes load test CR
func (c *Client) DeleteLoadTest(ctx context.Context, loadTest string) error {
	c.logger.Debug("Deleting load test", zap.String("loadtest", loadTest))

	err := c.ltClient.Delete(ctx, loadTest, metaV1.DeleteOptions{GracePeriodSeconds: &gracePeriod})
	if err != nil {
		c.logger.Error("Error on deleting the load test", zap.String("loadtest", loadTest), zap.Error(err))
		return err
	}
	c.logger.Info("Deleted load test", zap.String("loadtest", loadTest))

	return err
}

// GetLoadTest returns load test information
func (c *Client) GetLoadTest(ctx context.Context, loadTest string) (*apisLoadTestV1.LoadTest, error) {
	c.logger.Debug("Retrieving info for loadtest", zap.String("loadtest", loadTest))
	result, err := c.ltClient.Get(ctx, loadTest, metaV1.GetOptions{})
	if err != nil {
		c.logger.Error("Error on retrieving info for loadtest", zap.String("loadtest", loadTest), zap.Error(err))
		return nil, err
	}
	return result, nil
}

// ListLoadTest returns list of load tests.
func (c *Client) ListLoadTest(ctx context.Context, opt ListOptions) (*apisLoadTestV1.LoadTestList, error) {
	k8sOpt := metaV1.ListOptions{
		Limit:    opt.Limit,
		Continue: opt.Continue,
	}

	// Label Selector.
	labelSelectors := make([]string, 0, len(opt.Tags))

	for label, value := range opt.Tags {
		labelSelectors = append(labelSelectors, fmt.Sprintf("test-tag-%s=%s", label, value))
	}

	k8sOpt.LabelSelector = strings.Join(labelSelectors, ",")

	// List load tests.
	c.logger.Debug("List load tests")

	loadTests, err := c.ltClient.List(ctx, k8sOpt)
	if err != nil {
		c.logger.Error("failed to list load tests", zap.Error(err))
		return nil, err
	}

	return loadTests, nil
}

// CountActiveLoadTests returns a number of currently running load tests
func (c *Client) CountActiveLoadTests(ctx context.Context) (int, error) {
	loadTests, err := c.ltClient.List(ctx, metaV1.ListOptions{})
	if err != nil {
		return 0, err
	}
	counter := 0

	// CRD-s currently don't support custom field selectors, so we have to iterate via all load tests and check status phase
	for _, loadTest := range loadTests.Items {
		if loadTest.Status.Phase == apisLoadTestV1.LoadTestRunning || loadTest.Status.Phase == apisLoadTestV1.LoadTestCreating {
			counter++
		}
	}
	return counter, nil
}

// GetMasterPodRequest is making an assumptions that we only care about the logs
// from the most recently created pod. It gets the pods associated with
// the master job and returns the request that is used for getting the logs
func (c *Client) GetMasterPodRequest(ctx context.Context, namespace string) (*restClient.Request, error) {
	pods, err := c.kubeClient.CoreV1().Pods(namespace).List(ctx, metaV1.ListOptions{
		LabelSelector: loadTestMasterLabelSelector,
	})
	if err != nil {
		c.logger.Error(err.Error())
		return nil, err
	}

	podID := getMostRecentPod(pods)

	return c.kubeClient.CoreV1().Pods(namespace).GetLogs(podID, &coreV1.PodLogOptions{}), nil
}

// GetWorkerPodRequest is used for getting the logs from worker pod
func (c *Client) GetWorkerPodRequest(ctx context.Context, namespace, workerPodID string) *restClient.Request {
	return c.kubeClient.CoreV1().Pods(namespace).GetLogs(workerPodID, &coreV1.PodLogOptions{})
}

func getMostRecentPod(pods *coreV1.PodList) string {
	podID := ""
	// duration is initialized to 20 years
	duration := time.Hour * 24 * 30 * 12 * 20

	for _, pod := range pods.Items {
		if pod.Status.StartTime == nil {
			continue
		}

		if d := time.Since(pod.Status.StartTime.Time); d < duration {
			duration = d
			podID = pod.GetName()
		}
	}

	return podID
}
