package locust

import (
	"context"
	"fmt"
	"net/url"

	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/hellofresh/kangal/pkg/core/helper"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	loadtestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	clientSetV "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned"
	"go.uber.org/zap"
)

var (
	defaultImage    = "locustio/locust"
	defaultImageTag = "latest"
)

// Locust enables the controller to run a loadtest using locust.io
type Locust struct {
	kubeClientSet      kubernetes.Interface
	kangalClientSet    clientSetV.Interface
	loadTest           *loadTestV1.LoadTest
	logger             *zap.Logger
	reportPreSignedURL *url.URL
	masterResources    helper.Resources
	workerResources    helper.Resources
	podAnnotations     map[string]string
}

// SetDefaults mutates the LoadTest object to add default values to empty fields
func (c *Locust) SetDefaults() error {
	if c.loadTest.Status.Phase == "" {
		c.loadTest.Status.Phase = loadTestV1.LoadTestCreating
	}

	if c.loadTest.Spec.MasterConfig.Image == "" {
		c.loadTest.Spec.MasterConfig.Image = defaultImage
	}
	if c.loadTest.Spec.MasterConfig.Tag == "" {
		c.loadTest.Spec.MasterConfig.Tag = defaultImageTag
	}

	if c.loadTest.Spec.WorkerConfig.Image == "" {
		c.loadTest.Spec.WorkerConfig.Image = defaultImage
	}
	if c.loadTest.Spec.WorkerConfig.Tag == "" {
		c.loadTest.Spec.WorkerConfig.Tag = defaultImageTag
	}

	return nil
}

// CheckOrCreateResources check for resources or create the needed resources for the loadtest type
func (c *Locust) CheckOrCreateResources(ctx context.Context) error {
	workerJobs, err := c.kubeClientSet.
		BatchV1().
		Jobs(c.loadTest.Status.Namespace).
		List(ctx, metaV1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", newWorkerJobName(c.loadTest)),
		})
	if err != nil {
		c.logger.Error("Error on listing jobs", zap.Error(err))
		return err
	}

	if len(workerJobs.Items) > 0 {
		return nil
	}

	configMap := newConfigMap(c.loadTest)
	_, err = c.kubeClientSet.
		CoreV1().
		ConfigMaps(c.loadTest.Status.Namespace).
		Create(ctx, configMap, metaV1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		c.logger.Error("Error on creating testfile configmap", zap.Error(err))
		return err
	}

	var secret *coreV1.Secret

	if c.loadTest.Spec.EnvVars != "" {
		envs, err := helper.ReadEnvs(c.loadTest.Spec.EnvVars)
		if err != nil {
			c.logger.Error("Error reading envVars", zap.Error(err))
			return err
		}
		if len(envs) > 0 {
			secret = newSecret(c.loadTest, envs)
			if nil != err {
				c.logger.Error("Error on creating secret", zap.Error(err))
				return err
			}
			_, err = c.kubeClientSet.
				CoreV1().
				Secrets(c.loadTest.Status.Namespace).
				Create(ctx, secret, metaV1.CreateOptions{})
			if err != nil && !errors.IsAlreadyExists(err) {
				c.logger.Error("Error on creating testfile configmap", zap.Error(err))
				return err
			}
		}
	}

	masterJob := newMasterJob(c.loadTest, configMap, secret, c.reportPreSignedURL, c.masterResources, c.podAnnotations)
	_, err = c.kubeClientSet.
		BatchV1().
		Jobs(c.loadTest.Status.Namespace).
		Create(ctx, masterJob, metaV1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		c.logger.Error("Error on creating master job", zap.Error(err))
		return err
	}

	masterService := newMasterService(c.loadTest, masterJob)
	_, err = c.kubeClientSet.CoreV1().Services(c.loadTest.Status.Namespace).Create(ctx, masterService, metaV1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		c.logger.Error("Error on creating master service", zap.Error(err))
		return err
	}

	workerJob := newWorkerJob(c.loadTest, configMap, secret, masterService, c.workerResources, c.podAnnotations)
	_, err = c.kubeClientSet.
		BatchV1().
		Jobs(c.loadTest.Status.Namespace).
		Create(ctx, workerJob, metaV1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		c.logger.Error("Error on creating worker job", zap.Error(err))
		return err
	}

	return nil
}

// CheckOrUpdateStatus check current LoadTest progress
func (c *Locust) CheckOrUpdateStatus(ctx context.Context) error {
	if c.loadTest.Status.Phase == loadTestV1.LoadTestErrored ||
		c.loadTest.Status.Phase == loadTestV1.LoadTestFinished {
		return nil
	}

	_, err := c.kubeClientSet.
		CoreV1().
		ConfigMaps(c.loadTest.Status.Namespace).
		Get(ctx, newConfigMapName(c.loadTest), metaV1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			c.loadTest.Status.Phase = loadTestV1.LoadTestFinished
			return nil
		}
		return err
	}

	workerJob, err := c.kubeClientSet.
		BatchV1().
		Jobs(c.loadTest.Status.Namespace).
		Get(ctx, newWorkerJobName(c.loadTest), metaV1.GetOptions{})
	if err != nil {
		return err
	}

	masterJob, err := c.kubeClientSet.
		BatchV1().
		Jobs(c.loadTest.Status.Namespace).
		Get(ctx, newMasterJobName(c.loadTest), metaV1.GetOptions{})
	if err != nil {
		return err
	}

	setLoadTestStatusFromJobs(c.loadTest, masterJob, workerJob)

	return nil
}

// New creates a instance of Locust backend
func New(
	kubeClientSet kubernetes.Interface,
	kangalClientSet clientSetV.Interface,
	loadTest *loadTestV1.LoadTest,
	logger *zap.Logger,
	reportPreSignedURL *url.URL,
	config Config,
	podAnnotations map[string]string,
) *Locust {
	return &Locust{
		kubeClientSet:      kubeClientSet,
		kangalClientSet:    kangalClientSet,
		loadTest:           loadTest,
		logger:             logger,
		reportPreSignedURL: reportPreSignedURL,
		masterResources: helper.Resources{
			CPULimits:      config.MasterCPULimits,
			CPURequests:    config.MasterCPURequests,
			MemoryLimits:   config.MasterMemoryLimits,
			MemoryRequests: config.MasterMemoryRequests,
		},
		workerResources: helper.Resources{
			CPULimits:      config.WorkerCPULimits,
			CPURequests:    config.WorkerCPURequests,
			MemoryLimits:   config.WorkerMemoryLimits,
			MemoryRequests: config.WorkerMemoryRequests,
		},
		podAnnotations: podAnnotations,
	}
}

func setLoadTestStatusFromJobs(loadTest *loadtestV1.LoadTest, masterJob *batchV1.Job, workerJob *batchV1.Job) {
	if workerJob.Status.Active > int32(0) || masterJob.Status.Active > int32(0) {
		loadTest.Status.Phase = loadTestV1.LoadTestRunning
		return
	}

	if workerJob.Status.Succeeded == 0 && workerJob.Status.Failed == 0 {
		loadTest.Status.Phase = loadTestV1.LoadTestStarting
		return
	}

	if masterJob.Status.Succeeded == 0 && masterJob.Status.Failed == 0 {
		loadTest.Status.Phase = loadTestV1.LoadTestStarting
		return
	}

	if workerJob.Status.Failed > int32(0) || masterJob.Status.Failed > int32(0) {
		loadTest.Status.Phase = loadTestV1.LoadTestErrored
		return
	}

	loadTest.Status.Phase = loadTestV1.LoadTestFinished
}
