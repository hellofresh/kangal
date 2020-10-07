package locust

import (
	"context"
	"net/url"

	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
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
	configMaps, err := c.kubeClientSet.
		CoreV1().
		ConfigMaps(c.loadTest.Status.Namespace).
		List(ctx, metaV1.ListOptions{})
	if err != nil {
		c.logger.Error("Error on listing configmaps", zap.Error(err))
		return err
	}

	if len(configMaps.Items) > 0 {
		return nil
	}

	configMap := newConfigMap(c.loadTest)
	configMap, err = c.kubeClientSet.
		CoreV1().
		ConfigMaps(c.loadTest.Status.Namespace).
		Create(ctx, configMap, metaV1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		c.logger.Error("Error on creating testfile configmap", zap.Error(err))
		return err
	}

	masterJob := newMasterJob(c.loadTest, configMap, c.reportPreSignedURL, c.podAnnotations)
	masterJob, err = c.kubeClientSet.
		BatchV1().
		Jobs(c.loadTest.Status.Namespace).
		Create(ctx, masterJob, metaV1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		c.logger.Error("Error on creating master job", zap.Error(err))
		return err
	}

	masterService := newMasterService(c.loadTest, masterJob)
	masterService, err = c.kubeClientSet.CoreV1().Services(c.loadTest.Status.Namespace).Create(ctx, masterService, metaV1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		c.logger.Error("Error on creating master service", zap.Error(err))
		return err
	}

	workerJob := newWorkerJob(c.loadTest, configMap, masterService, c.podAnnotations)
	workerJob, err = c.kubeClientSet.
		BatchV1().
		Jobs(c.loadTest.Status.Namespace).
		Create(ctx, workerJob, metaV1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		c.logger.Error("Error on creating worker job", zap.Error(err))
		return err
	}

	return nil
}

// CheckOrUpdateLoadTestStatus check current LoadTest progress
func (*Locust) CheckOrUpdateStatus(ctx context.Context) error {
	return nil
}

func New(
	kubeClientSet kubernetes.Interface,
	kangalClientSet clientSetV.Interface,
	loadTest *loadTestV1.LoadTest,
	logger *zap.Logger,
	reportPreSignedURL *url.URL,
	podAnnotations map[string]string,
) *Locust {
	return &Locust{
		kubeClientSet:      kubeClientSet,
		kangalClientSet:    kangalClientSet,
		loadTest:           loadTest,
		logger:             logger,
		reportPreSignedURL: reportPreSignedURL,
		podAnnotations:     podAnnotations,
	}
}
