package jmeter

import (
	"context"
	"fmt"
	"time"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	clientSetV "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	coreListersV1 "k8s.io/client-go/listers/core/v1"
)

var (
	// MaxWaitTimeForPods is the time we should wait for worker pods to get to a "Running" state
	MaxWaitTimeForPods = time.Minute * 10
	// LoadTestWorkerLabelSelector is the selector used for selecting jmeter worker resources
	LoadTestWorkerLabelSelector = fmt.Sprintf("%s=%s", loadTestWorkerPodLabelKey, loadTestWorkerPodLabelValue)
)

const (
	defaultMasterImageName = "hellofreshtech/kangal-jmeter-master"
	defaultWorkerImageName = "hellofreshtech/kangal-jmeter-worker"
	defaultMasterImageTag  = "latest"
	defaultWorkerImageTag  = "latest"
)

// JMeter enables the controller to run a loadtest using JMeter
type JMeter struct {
	backend   *Backend
	loadTest  *loadTestV1.LoadTest
	reportURL string
}

//New initializes new JMeter provider handler to manage load test resources with Kangal Controller
func New(
	kubeClientSet kubernetes.Interface,
	kangalClientSet clientSetV.Interface,
	lt *loadTestV1.LoadTest,
	logger *zap.Logger,
	namespacesLister coreListersV1.NamespaceLister,
	reportURL string,
	podAnnotations, _ map[string]string,
	config Config,
) *JMeter {
	backend := &Backend{
		kubeClientSet:   kubeClientSet,
		kangalClientSet: kangalClientSet,
		logger:          logger,
		namespaceLister: namespacesLister,
		config:          &config,
		podAnnotations:  podAnnotations,
	}

	backend.SetDefaults()

	return &JMeter{
		backend:   backend,
		loadTest:  lt,
		reportURL: reportURL,
	}
}

// CheckOrCreateResources check if JMeter kubernetes resources have been create,
// if they have not been create them
func (c *JMeter) CheckOrCreateResources(ctx context.Context) error {
	return c.backend.Sync(ctx, *c.loadTest, c.reportURL)
}

// CheckOrUpdateStatus check the JMeter resources and calculate the current
// status of the loadtest from them
func (c *JMeter) CheckOrUpdateStatus(ctx context.Context) error {
	return c.backend.SyncStatus(ctx, *c.loadTest, &c.loadTest.Status)
}
