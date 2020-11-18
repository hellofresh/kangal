package jmeter

import (
	"context"
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

	"github.com/hellofresh/kangal/pkg/core/helper"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	clientSetV "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned"
)

var (
	// MaxWaitTimeForPods is the time we should wait for worker pods to get to a "Running" state
	MaxWaitTimeForPods = time.Minute * 10
	// LoadTestWorkerLabelSelector is the selector used for selecting jmeter worker resources
	LoadTestWorkerLabelSelector = fmt.Sprintf("%s=%s", loadTestWorkerPodLabelKey, loadTestWorkerPodLabelValue)
)

var (
	// ErrRequireMinOneDistributedPod spec requires 1 or more DistributedPods
	ErrRequireMinOneDistributedPod = errors.New("LoadTest must specify 1 or more DistributedPods")
	// ErrRequireTestFile the TestFile filed is required to not be an empty string
	ErrRequireTestFile = errors.New("LoadTest TestFile is required")
)

var (
	defaultMasterImageName = "hellofreshtech/kangal-jmeter-master"
	defaultMasterImageTag  = "latest"
	defaultWorkerImageName = "hellofreshtech/kangal-jmeter-worker"
	defaultWorkerImageTag  = "latest"
)

// JMeter enables the controller to run a loadtest using JMeter
type JMeter struct {
	kubeClientSet   kubernetes.Interface
	kangalClientSet clientSetV.Interface
	logger          *zap.Logger
	namespaceLister coreListersV1.NamespaceLister
	masterResources helper.Resources
	workerResources helper.Resources
	masterConfig    loadTestV1.ImageDetails
	workerConfig    loadTestV1.ImageDetails
	envConfig       *Config
	podAnnotations  map[string]string
}

// Type returns backend type name
func (*JMeter) Type() loadTestV1.LoadTestType {
	return loadTestV1.LoadTestTypeJMeter
}

// GetEnvConfig must return envConfig struct pointer
func (b *JMeter) GetEnvConfig() interface{} {
	b.envConfig = &Config{}
	return b.envConfig
}

// SetDefaults must set default values
func (b *JMeter) SetDefaults() {
	if b.envConfig.MasterImageName == "" || b.envConfig.MasterImageTag == "" {
		b.envConfig.MasterImageName = defaultMasterImageName
		b.envConfig.MasterImageTag = defaultMasterImageTag
	}

	if b.envConfig.WorkerImageName == "" || b.envConfig.WorkerImageTag == "" {
		b.envConfig.WorkerImageName = defaultWorkerImageName
		b.envConfig.WorkerImageTag = defaultWorkerImageTag
	}

	b.masterConfig = loadTestV1.ImageDetails{
		Image: b.envConfig.MasterImageName,
		Tag:   b.envConfig.MasterImageTag,
	}
	b.workerConfig = loadTestV1.ImageDetails{
		Image: b.envConfig.WorkerImageName,
		Tag:   b.envConfig.WorkerImageTag,
	}

	b.masterResources = helper.Resources{
		CPULimits:      b.envConfig.MasterCPULimits,
		CPURequests:    b.envConfig.MasterCPURequests,
		MemoryLimits:   b.envConfig.MasterMemoryLimits,
		MemoryRequests: b.envConfig.MasterMemoryRequests,
	}
	b.workerResources = helper.Resources{
		CPULimits:      b.envConfig.WorkerCPULimits,
		CPURequests:    b.envConfig.WorkerCPURequests,
		MemoryLimits:   b.envConfig.WorkerMemoryLimits,
		MemoryRequests: b.envConfig.WorkerMemoryRequests,
	}
}

// SetLogger recieves a copy of logger
func (b *JMeter) SetLogger(logger *zap.Logger) {
	b.logger = logger
}

// TransformLoadTestSpec use given spec to validate and return a new one or error
func (b *JMeter) TransformLoadTestSpec(spec *loadTestV1.LoadTestSpec) error {
	if nil == spec.DistributedPods {
		return ErrRequireMinOneDistributedPod
	}

	if *spec.DistributedPods <= int32(0) {
		return ErrRequireMinOneDistributedPod
	}

	if spec.TestFile == "" {
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

	return nil
}

// SetPodAnnotations recieves a copy of pod annotations
func (b *JMeter) SetPodAnnotations(podAnnotations map[string]string) {
	b.podAnnotations = podAnnotations
}

// SetKubeClientSet recieves a copy of kubeClientSet
func (b *JMeter) SetKubeClientSet(kubeClientSet kubernetes.Interface) {
	b.kubeClientSet = kubeClientSet
}

// SetKangalClientSet recieves a copy of kangalClientSet
func (b *JMeter) SetKangalClientSet(kangalClientSet clientSetV.Interface) {
	b.kangalClientSet = kangalClientSet
}

// SetNamespaceLister recieves a copy of namespaceLister
func (b *JMeter) SetNamespaceLister(namespacesLister coreListersV1.NamespaceLister) {
	b.namespaceLister = namespacesLister
}

// Sync check if JMeter kubernetes resources have been create,
// if they have not been create them
func (b *JMeter) Sync(ctx context.Context, loadTest loadTestV1.LoadTest, reportURL string) error {
	JMeterServices, err := b.kubeClientSet.CoreV1().Services(loadTest.Status.Namespace).List(ctx, metaV1.ListOptions{})
	if err != nil {
		return err
	}

	if len(JMeterServices.Items) == 0 {
		_, err = b.kubeClientSet.CoreV1().ConfigMaps(loadTest.Status.Namespace).Create(ctx, b.NewConfigMap(loadTest), metaV1.CreateOptions{})
		if err != nil && !kerrors.IsAlreadyExists(err) {
			b.logger.Error("Error on creating testfile configmap", zap.Error(err))
			return err
		}

		_, err = b.kubeClientSet.CoreV1().ConfigMaps(loadTest.Status.Namespace).Create(ctx, b.NewJMeterSettingsConfigMap(loadTest), metaV1.CreateOptions{})
		if err != nil && !kerrors.IsAlreadyExists(err) {
			return err
		}

		secret, err := b.NewSecret(loadTest)
		if err != nil {
			b.logger.Error("Error on creating environment variables secret", zap.Error(err))
			return err
		}

		_, err = b.kubeClientSet.CoreV1().Secrets(loadTest.Status.Namespace).Create(ctx, secret, metaV1.CreateOptions{})
		if err != nil && !kerrors.IsAlreadyExists(err) {
			return err
		}

		configMaps, err := b.NewTestdataConfigMap(loadTest)
		if err != nil {
			b.logger.Error("Error on creating testdata configMaps", zap.Error(err))
			return err
		}

		err = b.createPodsWithTestdata(ctx, configMaps, &loadTest, loadTest.Status.Namespace)
		if err != nil {
			return err
		}

		_, err = b.kubeClientSet.CoreV1().Services(loadTest.Status.Namespace).Create(ctx, b.NewJMeterService(), metaV1.CreateOptions{})
		if err != nil && !kerrors.IsAlreadyExists(err) {
			b.logger.Error("Error on creating new JMeter service", zap.Error(err))
			return err
		}

		_, err = b.
			kubeClientSet.
			BatchV1().
			Jobs(loadTest.Status.Namespace).
			Create(
				ctx,
				b.NewJMeterMasterJob(loadTest, reportURL, b.podAnnotations),
				metaV1.CreateOptions{},
			)
		if err != nil && !kerrors.IsAlreadyExists(err) {
			b.logger.Error("Error on creating new JMeter master Job", zap.Error(err))
			return err
		}

		b.logger.Info(
			"Created JMeter resources",
			zap.String("LoadTest", loadTest.GetName()),
			zap.String("namespace", loadTest.Status.Namespace),
		)
	}
	return nil
}

// SyncStatus check the JMeter resources and calculate the current
// status of the loadtest from them
func (b *JMeter) SyncStatus(ctx context.Context, loadTest loadTestV1.LoadTest, loadTestStatus *loadTestV1.LoadTestStatus) error {
	// Get the Namespace resource
	namespace, err := b.namespaceLister.Get(loadTestStatus.Namespace)
	if err != nil {
		// The LoadTest resource may no longer exist, in which case we stop
		// processing.
		if kerrors.IsNotFound(err) {
			loadTestStatus.Phase = loadTestV1.LoadTestFinished
			return nil
		}
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
		if pod.Status.Phase != "Running" {
			// If the pod is not yet in the running phase check to see if the
			// pod start date is greater than the start time.
			if workerPodHasTimeout(pod.Status.StartTime, *loadTestStatus) {
				loadTestStatus.Phase = loadTestV1.LoadTestFinished
				return nil
			}

			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.State.Waiting == nil {
					loadTestStatus.Phase = loadTestV1.LoadTestErrored
					return nil
				}
				if containerStatus.State.Waiting.Reason != "Pending" &&
					containerStatus.State.Waiting.Reason != "ContainerCreating" &&
					containerStatus.State.Waiting.Reason != "PodInitializing" {
					b.logger.Info(
						"One of containers is unhealthy, marking LoadTest as errored",
						zap.String("LoadTest", loadTest.GetName()),
						zap.String("pod", pod.Name),
						zap.String("namespace", namespace.GetName()),
					)
					loadTestStatus.Phase = loadTestV1.LoadTestErrored
					return nil
				}
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
	loadTestStatus.Phase = getLoadTestPhaseFromJob(job.Status)
	loadTestStatus.JobStatus = job.Status

	return nil
}

func (b *JMeter) createPodsWithTestdata(ctx context.Context, configMaps []*coreV1.ConfigMap, loadTest *loadTestV1.LoadTest, namespace string) error {
	for i, cm := range configMaps {
		configMap, err := b.kubeClientSet.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metaV1.CreateOptions{})
		if err != nil {
			b.logger.Error("Error on creating testdata configMaps", zap.Error(err))
			return err
		}

		_, err = b.kubeClientSet.CoreV1().Pods(namespace).Create(ctx, b.NewPod(*loadTest, i, configMap, b.podAnnotations), metaV1.CreateOptions{})
		if err != nil {
			b.logger.Error("Error on creating distributed pods", zap.Error(err))
			return err
		}
	}
	b.logger.Info("Created pods with test data", zap.String("LoadTest", loadTest.GetName()), zap.String("namespace", namespace))
	return nil
}

func workerPodHasTimeout(startTime *metaV1.Time, loadtestStatus loadTestV1.LoadTestStatus) bool {
	if startTime == nil {
		return false
	}

	return time.Since(startTime.Time) > MaxWaitTimeForPods &&
		loadtestStatus.Phase == loadTestV1.LoadTestCreating
}

func getLoadTestPhaseFromJob(status batchV1.JobStatus) loadTestV1.LoadTestPhase {
	if status.Active > 0 {
		return loadTestV1.LoadTestRunning
	}

	if status.Succeeded == 0 && status.Failed == 0 {
		return loadTestV1.LoadTestStarting
	}

	return loadTestV1.LoadTestFinished
}
