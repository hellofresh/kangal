package kubernetes

import (
	"context"
	"os"
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

//NewClient creates new Kubernetes client
func NewClient(loadTestClient loadTestV1.LoadTestInterface, kubeClient kubernetes.Interface, logger *zap.Logger) *Client {
	return &Client{
		ltClient:   loadTestClient,
		kubeClient: kubeClient,
		logger:     logger,
	}
}

// CreateLoadTest creates new load test from given request data
func (c *Client) CreateLoadTest(ctx context.Context, loadTest *apisLoadTestV1.LoadTest) (string, error) {
	c.logger.Debug("Creating load test CR ...")

	fileHashLabel := loadTest.Labels["test-file-hash"]
	opts := metaV1.ListOptions{
		LabelSelector: "test-file-hash=" + fileHashLabel,
	}
	labeledLoadTests, err := c.ltClient.List(ctx, opts)
	if err != nil {
		c.logger.Error("Error on getting the list of created load tests with the given hash", zap.String("hash-label", loadTest.Labels["testFileHash"]), zap.Error(err))
		return "", err
	}

	if len(labeledLoadTests.Items) > 0 {
		c.logger.Error("Load test with given testfile already exists, aborting")
		return "", os.ErrExist
	}

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

// GetMasterPodLogs is making an assumptions that we only care about the logs
// from the most recently created pod. It gets the pods associated with
// the master job and returns the request that is used for getting the logs
func (c *Client) GetMasterPodLogs(ctx context.Context, namespace string) (*restClient.Request, error) {
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
