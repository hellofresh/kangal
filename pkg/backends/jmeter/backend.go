package jmeter

import (
	"context"

	kerrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	coreListersV1 "k8s.io/client-go/listers/core/v1"

	"github.com/hellofresh/kangal/pkg/backends/internal"
	"github.com/hellofresh/kangal/pkg/core/helper"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	clientSetV "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned"

	"go.uber.org/zap"
)

func init() {
	internal.Register(&Backend{})
}

// Backend is the Fake implementation of backend interface
type Backend struct {
	kubeClientSet   kubernetes.Interface
	kangalClientSet clientSetV.Interface
	logger          *zap.Logger
	namespaceLister coreListersV1.NamespaceLister
	masterResources helper.Resources
	workerResources helper.Resources
	masterConfig    loadTestV1.ImageDetails
	workerConfig    loadTestV1.ImageDetails
	config          *Config
	podAnnotations  map[string]string
}

// Type returns backend type name
func (*Backend) Type() loadTestV1.LoadTestType {
	return loadTestV1.LoadTestTypeJMeter
}

// GetEnvConfig must return config struct pointer
func (b *Backend) GetEnvConfig() interface{} {
	b.config = &Config{}
	return b.config
}

// SetDefaults must set default values
func (b *Backend) SetDefaults() {
	if b.config.MasterImageName == "" || b.config.MasterImageTag == "" {
		b.config.MasterImageName = defaultMasterImageName
		b.config.MasterImageTag = defaultMasterImageTag
	}

	if b.config.WorkerImageName == "" || b.config.WorkerImageTag == "" {
		b.config.WorkerImageName = defaultWorkerImageName
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

	b.masterResources = helper.Resources{
		CPULimits:      b.config.MasterCPULimits,
		CPURequests:    b.config.MasterCPURequests,
		MemoryLimits:   b.config.MasterMemoryLimits,
		MemoryRequests: b.config.MasterMemoryRequests,
	}
	b.workerResources = helper.Resources{
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

// TransformLoadTestSpec use given spec to validate and return a new one or error
func (b *Backend) TransformLoadTestSpec(spec *loadTestV1.LoadTestSpec) error {
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

// Sync check if Backend kubernetes resources have been create, if they have not been create them
func (b *Backend) Sync(ctx context.Context, loadTest loadTestV1.LoadTest, reportURL string) error {
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

		_, err = b.kubeClientSet.CoreV1().ConfigMaps(loadTest.Status.Namespace).Create(ctx, b.NewJMeterSettingsConfigMap(), metaV1.CreateOptions{})
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

// SyncStatus check the Backend resources and calculate the current status of the LoadTest from them
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
