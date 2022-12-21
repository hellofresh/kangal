package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.uber.org/zap"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	restClient "k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned/typed/loadtest/v1"
)

var (
	loadTestMasterLabelSelector = "app=loadtest-master"
	loadTestWorkerLabelSelector = "app=loadtest-worker-pod"
	// gracePeriod is duration in seconds before the object should be deleted.
	// The value zero indicates delete immediately.
	gracePeriod = int64(0)
)

// Client manages calls to Kubernetes API
type Client struct {
	ltClient   loadTestV1.LoadTestInterface
	kubeClient kubernetes.Interface
	logger     *zap.Logger
}

// ListOptions is options to find load tests.
type ListOptions struct {
	// List of tags.
	Tags map[string]string
	// Phase of loadTest
	Phase apisLoadTestV1.LoadTestPhase
	// Limit.
	Limit int64
	// Continue.
	Continue string
}

// NewClient creates new Kubernetes client
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

	return c.filterLoadTestsByPhase(loadTests, opt.Phase), nil
}

// filterLoadTestsByPhase returns a list of loadtests filtered by phase
func (c *Client) filterLoadTestsByPhase(list *apisLoadTestV1.LoadTestList, phase apisLoadTestV1.LoadTestPhase) *apisLoadTestV1.LoadTestList {
	if phase == "" {
		return list
	}

	filteredList := apisLoadTestV1.LoadTestList{
		TypeMeta: list.TypeMeta,
		ListMeta: list.ListMeta,
	}
	// CRD-s currently don't support custom field selectors, so we have to iterate via all load tests and check status phase
	for _, loadTest := range list.Items {
		if loadTest.Status.Phase == phase {
			filteredList.Items = append(filteredList.Items, loadTest)
		}
	}
	return &filteredList
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
func (c *Client) GetWorkerPodRequest(ctx context.Context, namespace, workerPodNr string) (*restClient.Request, error) {
	pods, err := c.kubeClient.CoreV1().Pods(namespace).List(ctx, metaV1.ListOptions{
		LabelSelector: loadTestWorkerLabelSelector,
	})
	if err != nil {
		c.logger.Error(err.Error())
		return nil, err
	}

	nr, _ := strconv.Atoi(workerPodNr)
	if nr >= len(pods.Items) {
		c.logger.Error("pod index out of range", zap.String("index", workerPodNr))
		return nil, errors.New("pod index is out of range")
	}

	sortWorkerPods(pods)

	return c.kubeClient.CoreV1().Pods(namespace).GetLogs(pods.Items[nr].Name, &coreV1.PodLogOptions{}), nil
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

func sortWorkerPods(pods *coreV1.PodList) {
	sort.Slice(pods.Items, func(i, j int) bool {
		return strings.Compare(pods.Items[i].ObjectMeta.Name, pods.Items[j].ObjectMeta.Name) < 0
	})
}

// BuildClientConfig is used in cmd package
func BuildClientConfig(masterURL string, kubeConfigPath string, timeout time.Duration) (*restClient.Config, error) {
	kubeCfg, err := clientcmd.BuildConfigFromFlags(masterURL, kubeConfigPath)
	if err != nil {
		return nil, err
	}

	kubeCfg.Timeout = timeout
	return kubeCfg, nil
}

// CountExistingLoadtests used in metrics to report running loadtests
func (c *Client) CountExistingLoadtests() (map[apisLoadTestV1.LoadTestPhase]int64, map[apisLoadTestV1.LoadTestType]int64, error) {
	tt, err := c.ltClient.List(context.Background(), metaV1.ListOptions{})
	if err != nil {
		c.logger.Error("Couldn't list existing loadtests", zap.Error(err))
		return nil, nil, err
	}

	var phaseCount = map[apisLoadTestV1.LoadTestPhase]int64{
		apisLoadTestV1.LoadTestRunning:  0,
		apisLoadTestV1.LoadTestFinished: 0,
		apisLoadTestV1.LoadTestCreating: 0,
		apisLoadTestV1.LoadTestErrored:  0,
		apisLoadTestV1.LoadTestStarting: 0,
	}

	var typeCount = map[apisLoadTestV1.LoadTestType]int64{
		apisLoadTestV1.LoadTestTypeK6:     0,
		apisLoadTestV1.LoadTestTypeJMeter: 0,
		apisLoadTestV1.LoadTestTypeLocust: 0,
		apisLoadTestV1.LoadTestTypeGhz:    0,
	}

	for _, loadTest := range tt.Items {
		phaseString := loadTest.Status.Phase
		phaseCount[phaseString]++

		typeString := loadTest.Spec.Type
		typeCount[typeString]++
	}

	return phaseCount, typeCount, nil
}
