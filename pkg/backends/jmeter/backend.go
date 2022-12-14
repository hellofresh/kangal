package jmeter

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"go.uber.org/zap"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	coreListersV1 "k8s.io/client-go/listers/core/v1"

	"github.com/hellofresh/kangal/pkg/backends"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	clientSetV "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned"
)

var (
	// MaxWaitTimeForPods is the time we should wait for worker pods to get to a "Running" state
	MaxWaitTimeForPods = time.Minute * 10
	// LoadTestWorkerLabelSelector is the selector used for selecting jmeter worker resources
	LoadTestWorkerLabelSelector = fmt.Sprintf("%s=%s", loadTestWorkerPodLabelKey, loadTestWorkerPodLabelValue)
	// ErrRequireMinOneDistributedPod Backend spec requires 1 or more DistributedPods
	ErrRequireMinOneDistributedPod = errors.New("LoadTest must specify 1 or more DistributedPods")
	// ErrRequireTestFile the TestFile filed is required to not be an empty string
	ErrRequireTestFile = errors.New("LoadTest TestFile is required")
)

const (
	defaultMasterImageName = "hellofresh/kangal-jmeter-master"
	defaultWorkerImageName = "hellofresh/kangal-jmeter-worker"
	defaultMasterImageTag  = "latest"
	defaultWorkerImageTag  = "latest"
)

func init() {
	backends.Register(&Backend{})
}

// Backend is the JMeter implementation of backend interface
type Backend struct {
	kubeClientSet   kubernetes.Interface
	kangalClientSet clientSetV.Interface
	logger          *zap.Logger
	namespaceLister coreListersV1.NamespaceLister
	masterResources backends.Resources
	workerResources backends.Resources
	masterConfig    loadTestV1.ImageDetails
	workerConfig    loadTestV1.ImageDetails
	config          *Config
	podAnnotations  map[string]string
	nodeSelector    map[string]string
	tolerations     []coreV1.Toleration
}

// Type returns backend type name
func (*Backend) Type() loadTestV1.LoadTestType {
	return loadTestV1.LoadTestTypeJMeter
}

func (*Backend) UsesCSVTestData() bool {
	return true
}

// GetEnvConfig must return config struct pointer
func (b *Backend) GetEnvConfig() interface{} {
	b.config = &Config{}
	return b.config
}

// SetDefaults must set default values
func (b *Backend) SetDefaults() {
	if b.config.MasterImageName == "" {
		b.config.MasterImageName = defaultMasterImageName
	}
	if b.config.MasterImageTag == "" {
		b.config.MasterImageTag = defaultMasterImageTag
	}

	if b.config.WorkerImageName == "" {
		b.config.WorkerImageName = defaultWorkerImageName
	}
	if b.config.WorkerImageTag == "" {
		b.config.WorkerImageTag = defaultWorkerImageTag
	}

	b.masterConfig = loadTestV1.ImageDetails{
		Image: b.config.MasterImageName,
		Tag:   b.config.MasterImageTag,
	}
	b.workerConfig = loadTestV1.ImageDetails{
		Image: b.config.WorkerImageName,
		Tag:   b.config.WorkerImageTag,
	}

	b.masterResources = backends.Resources{
		CPULimits:      b.config.MasterCPULimits,
		CPURequests:    b.config.MasterCPURequests,
		MemoryLimits:   b.config.MasterMemoryLimits,
		MemoryRequests: b.config.MasterMemoryRequests,
	}
	b.workerResources = backends.Resources{
		CPULimits:      b.config.WorkerCPULimits,
		CPURequests:    b.config.WorkerCPURequests,
		MemoryLimits:   b.config.WorkerMemoryLimits,
		MemoryRequests: b.config.WorkerMemoryRequests,
	}
}

// SetPodAnnotations receives a copy of pod annotations
func (b *Backend) SetPodAnnotations(podAnnotations map[string]string) {
	b.podAnnotations = podAnnotations
}

// SetKubeClientSet receives a copy of kubeClientSet
func (b *Backend) SetKubeClientSet(kubeClientSet kubernetes.Interface) {
	b.kubeClientSet = kubeClientSet
}

// SetKangalClientSet receives a copy of kangalClientSet
func (b *Backend) SetKangalClientSet(kangalClientSet clientSetV.Interface) {
	b.kangalClientSet = kangalClientSet
}

// SetNamespaceLister receives a copy of namespaceLister
func (b *Backend) SetNamespaceLister(namespacesLister coreListersV1.NamespaceLister) {
	b.namespaceLister = namespacesLister
}

// SetLogger receives a copy of logger
func (b *Backend) SetLogger(logger *zap.Logger) {
	b.logger = logger
}

// SetPodNodeSelector receives a copy of pod node selectors
func (b *Backend) SetPodNodeSelector(nodeselector map[string]string) {
	b.nodeSelector = nodeselector
}

// SetPodTolerations receives a copy of pod tolerations
func (b *Backend) SetPodTolerations(tolerations []coreV1.Toleration) {
	b.tolerations = tolerations
}

// TransformLoadTestSpec use given spec to validate and return a new one or error
func (b *Backend) TransformLoadTestSpec(spec *loadTestV1.LoadTestSpec) error {
	if nil == spec.DistributedPods {
		return ErrRequireMinOneDistributedPod
	}

	if *spec.DistributedPods <= int32(0) {
		return ErrRequireMinOneDistributedPod
	}

	if len(spec.TestFile) == 0 {
		return ErrRequireTestFile
	}

	if spec.MasterConfig.Image == "" || spec.MasterConfig.Tag == "" {
		spec.MasterConfig.Image = b.masterConfig.Image
		spec.MasterConfig.Tag = b.masterConfig.Tag
	}

	if spec.WorkerConfig.Image == "" || spec.WorkerConfig.Tag == "" {
		spec.WorkerConfig.Image = b.workerConfig.Image
		spec.WorkerConfig.Tag = b.workerConfig.Tag
	}

	if len(spec.TestData) > 0 {
		testDataBase64, err := generateBase64(string(spec.TestData))
		if err != nil {
			return err
		}

		spec.TestData = []byte(testDataBase64)
	}

	return nil
}

// Sync check if JMeter kubernetes resources have been create, if they have not been create them
func (b *Backend) Sync(ctx context.Context, loadTest loadTestV1.LoadTest, testfileConfigMapName string, testdataConfigMapNames []string, reportURL string) error {
	logger := b.logger.With(
		zap.String("loadtest", loadTest.GetName()),
		zap.String("namespace", loadTest.Status.Namespace),
	)

	JMeterServices, err := b.kubeClientSet.CoreV1().Services(loadTest.Status.Namespace).List(ctx, metaV1.ListOptions{})
	if err != nil {
		return err
	}

	if len(JMeterServices.Items) == 0 {
		secret, err := b.NewSecret(loadTest)
		if err != nil {
			logger.Error("Error on creating environment variables secret", zap.Error(err))
			return err
		}

		_, err = b.kubeClientSet.CoreV1().Secrets(loadTest.Status.Namespace).Create(ctx, secret, metaV1.CreateOptions{})
		if err != nil && !kerrors.IsAlreadyExists(err) {
			return err
		}

		err = b.CreatePodsWithTestdata(ctx, testdataConfigMapNames, &loadTest, loadTest.Status.Namespace)
		if err != nil {
			return err
		}

		_, err = b.kubeClientSet.CoreV1().Services(loadTest.Status.Namespace).Create(ctx, b.NewJMeterService(), metaV1.CreateOptions{})
		if err != nil && !kerrors.IsAlreadyExists(err) {
			logger.Error("Error on creating new JMeter service", zap.Error(err))
			return err
		}

		_, err = b.
			kubeClientSet.
			BatchV1().
			Jobs(loadTest.Status.Namespace).
			Create(
				ctx,
				b.NewJMeterMasterJob(loadTest, testfileConfigMapName, reportURL, b.podAnnotations),
				metaV1.CreateOptions{},
			)
		if err != nil && !kerrors.IsAlreadyExists(err) {
			logger.Error("Error on creating new JMeter master Job", zap.Error(err))
			return err
		}

		logger.Info("Created JMeter resources")
	}
	return nil
}

// SyncStatus check the JMeter resources and calculate the current status of the LoadTest from them
func (b *Backend) SyncStatus(ctx context.Context, loadTest loadTestV1.LoadTest, loadTestStatus *loadTestV1.LoadTestStatus) error {
	// Get the Namespace resource
	namespace, err := b.namespaceLister.Get(loadTestStatus.Namespace)
	if err != nil {
		// The LoadTest resource may no longer exist, in which case we stop
		// processing.
		if kerrors.IsNotFound(err) {
			loadTestStatus.Phase = loadTestV1.LoadTestFinished
			return nil
		}
		return err
	}

	if loadTestStatus.Phase == "" {
		loadTestStatus.Phase = loadTestV1.LoadTestCreating
	}

	if loadTestStatus.Phase == loadTestV1.LoadTestErrored {
		return nil
	}

	pods, err := b.kubeClientSet.CoreV1().Pods(namespace.GetName()).List(ctx, metaV1.ListOptions{
		LabelSelector: LoadTestWorkerLabelSelector,
	})
	if err != nil {
		return err
	}

	if len(pods.Items) != int(*loadTest.Spec.DistributedPods) {
		return nil
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase != coreV1.PodRunning {
			// If the pod is not yet in the running phase check to see if the
			// pod start date is greater than the start time.
			if workerPodHasTimeout(pod.Status.StartTime, *loadTestStatus) {
				loadTestStatus.Phase = loadTestV1.LoadTestFinished
				return nil
			}
			loadTestStatus.Phase = getLoadTestStatusPhaseByPod(pod)
			if loadTestV1.LoadTestErrored == loadTestStatus.Phase {
				b.logger.Info(
					"One of containers is unhealthy, marking LoadTest as errored",
					zap.String("loadTest", loadTest.GetName()),
					zap.String("pod", pod.Name),
				)
			}
			return nil
		}
	}

	job, err := b.kubeClientSet.BatchV1().Jobs(namespace.GetName()).Get(ctx, loadTestJobName, metaV1.GetOptions{})
	if err != nil {
		return err
	}

	// Get jmeter job in namespace and update the LoadTest status with
	// the Job status
	loadTestStatus.Phase = determineLoadTestPhaseFromJob(job.Status)
	loadTestStatus.JobStatus = job.Status

	return nil
}

func generateBase64(testData string) (string, error) {
	var result string

	var by bytes.Buffer
	gz := gzip.NewWriter(&by)
	if _, err := gz.Write([]byte(testData)); err != nil {
		return result, err
	}

	if err := gz.Flush(); err != nil {
		return result, err
	}

	if err := gz.Close(); err != nil {
		return result, err
	}

	result = base64.RawStdEncoding.EncodeToString(by.Bytes())

	return result, nil
}

func getLoadTestStatusPhaseByPod(pod coreV1.Pod) loadTestV1.LoadTestPhase {
	for _, containerStatus := range pod.Status.ContainerStatuses {
		if containerStatus.State.Waiting == nil {
			return loadTestV1.LoadTestErrored
		}
		if containerStatus.State.Waiting.Reason != "Pending" &&
			containerStatus.State.Waiting.Reason != "ContainerCreating" &&
			containerStatus.State.Waiting.Reason != "PodInitializing" {
			return loadTestV1.LoadTestErrored
		}
	}
	return loadTestV1.LoadTestStarting
}

func workerPodHasTimeout(startTime *metaV1.Time, loadtestStatus loadTestV1.LoadTestStatus) bool {
	if startTime == nil {
		return false
	}

	return time.Since(startTime.Time) > MaxWaitTimeForPods &&
		loadtestStatus.Phase == loadTestV1.LoadTestCreating
}

// determineLoadTestPhaseFromJob
func determineLoadTestPhaseFromJob(status batchV1.JobStatus) loadTestV1.LoadTestPhase {
	if status.Active > 0 {
		return loadTestV1.LoadTestRunning
	}

	if status.Succeeded == 0 && status.Failed == 0 {
		return loadTestV1.LoadTestStarting
	}

	return loadTestV1.LoadTestFinished
}
