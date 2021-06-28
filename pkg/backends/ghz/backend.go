package ghz

import (
	"context"
	"errors"
	"fmt"

	"github.com/hellofresh/kangal/pkg/backends"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	"go.uber.org/zap"
	coreV1 "k8s.io/api/core/v1"
	k8sAPIErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

var (
	// ErrRequireMinOneDistributedPod Backend spec requires 1 or more DistributedPods
	ErrRequireMinOneDistributedPod = errors.New("LoadTest must specify 1 or more DistributedPods")
	// ErrRequireTestFile the TestFile filed is required to not be an empty string
	ErrRequireTestFile = errors.New("LoadTest TestFile is required")
)

const (
	defaultImageName = "hellofresh/kangal-ghz"
	defaultImageTag  = "latest"
)

func init() {
	backends.Register(&Backend{})
}

// Backend is the ghz implementation of backend interface
type Backend struct {
	logger         *zap.Logger
	kubeClientSet  kubernetes.Interface
	config         *Config
	podAnnotations map[string]string

	// defined on SetDefaults
	image     loadTestV1.ImageDetails
	resources backends.Resources
}

// Type returns backend type name
func (*Backend) Type() loadTestV1.LoadTestType {
	return loadTestV1.LoadTestTypeGhz
}

// GetEnvConfig must return config struct pointer
func (b *Backend) GetEnvConfig() interface{} {
	b.config = &Config{}
	return b.config
}

// SetDefaults must set default values
func (b *Backend) SetDefaults() {
	if b.config.ImageName == "" || b.config.ImageTag == "" {
		b.config.ImageName = defaultImageName
		b.config.ImageTag = defaultImageTag
	}
	b.image = loadTestV1.ImageDetails{
		Image: b.config.ImageName,
		Tag:   b.config.ImageTag,
	}

	b.resources = backends.Resources{
		CPULimits:      b.config.CPULimits,
		CPURequests:    b.config.CPURequests,
		MemoryLimits:   b.config.MemoryLimits,
		MemoryRequests: b.config.MemoryRequests,
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
		spec.MasterConfig.Image = b.image.Image
		spec.MasterConfig.Tag = b.image.Tag
	}

	return nil
}

// Sync checks if ghz kubernetes resources have been created, create them if they haven't
func (b *Backend) Sync(ctx context.Context, loadTest loadTestV1.LoadTest, reportURL string) error {
	jobs, err := b.kubeClientSet.
		BatchV1().
		Jobs(loadTest.Status.Namespace).
		List(ctx, metaV1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", loadTestJobName),
		})
	if err != nil {
		b.logger.Error("Error on listing jobs", zap.Error(err))
		return err
	}

	// Jobs already created, do nothing
	if len(jobs.Items) > 0 {
		return nil
	}

	var (
		tdCfgMap   *coreV1.ConfigMap
		configMaps = make([]*coreV1.ConfigMap, 1)
	)

	// Create testfile ConfigMap
	tfCfgMap, err := NewFileConfigMap(loadTestFileConfigMapName, configFileName, loadTest.Spec.TestFile)
	if err != nil {
		b.logger.Error("Error creating testfile configmap resource", zap.Error(err))
		return err
	}
	configMaps[0] = tfCfgMap

	// Prepare testdata ConfigMap
	if loadTest.Spec.TestData != "" {
		tdCfgMap, err = NewFileConfigMap(loadTestFileConfigMapName, configFileName, loadTest.Spec.TestData)
		if err != nil {
			b.logger.Error("Error creating testdata configmap resource", zap.Error(err))
			return err
		}
		configMaps = append(configMaps, tdCfgMap)
	}

	for _, cfg := range configMaps {
		_, err = b.kubeClientSet.
			CoreV1().
			ConfigMaps(loadTest.Status.Namespace).
			Create(ctx, cfg, metaV1.CreateOptions{})
		if err != nil && !k8sAPIErrors.IsAlreadyExists(err) {
			b.logger.Error("Error creating configmap", zap.String("configmap", cfg.GetName()), zap.Error(err))
			return err
		}
	}

	var (
		volumes = make([]coreV1.Volume, 1)
		mounts  = make([]coreV1.VolumeMount, 1)
	)

	volumes[0], mounts[0] = NewFileVolumeAndMount("testfile", tfCfgMap.Name, configFileName)

	if tdCfgMap != nil {
		v, m := NewFileVolumeAndMount("testdata", tdCfgMap.Name, testdataFileName)
		volumes = append(volumes, v)
		mounts = append(mounts, m)
	}

	// Create Job
	job := b.NewJob(loadTest, volumes, mounts, reportURL)
	_, err = b.kubeClientSet.
		BatchV1().
		Jobs(loadTest.Status.Namespace).
		Create(ctx, job, metaV1.CreateOptions{})
	if err != nil && !k8sAPIErrors.IsAlreadyExists(err) {
		b.logger.Error("Error on creating master job", zap.Error(err))
		return err
	}

	return nil
}

// SyncStatus checks ghz resources and updates the status of the LoadTest resource
func (b *Backend) SyncStatus(ctx context.Context, _ loadTestV1.LoadTest, loadTestStatus *loadTestV1.LoadTestStatus) error {
	if loadTestStatus.Phase == "" {
		loadTestStatus.Phase = loadTestV1.LoadTestCreating
		return nil
	}

	if loadTestStatus.Phase == loadTestV1.LoadTestErrored {
		return nil
	}

	job, err := b.kubeClientSet.
		BatchV1().
		Jobs(loadTestStatus.Namespace).
		Get(ctx, loadTestJobName, metaV1.GetOptions{})
	if err != nil {
		return err
	}

	loadTestStatus.Phase = determineLoadTestStatusFromJobs(job)
	loadTestStatus.JobStatus = job.Status
	return nil
}
