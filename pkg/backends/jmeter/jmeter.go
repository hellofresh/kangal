package jmeter

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
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
	//loadTestWorkerLabelSelector is the selector used for selecting jmeter worker resources
	loadTestWorkerLabelSelector = fmt.Sprintf("%s=%s", loadTestWorkerPodLabelKey, loadTestWorkerPodLabelValue)
	masterImage                 = "hellofreshtech/kangal-jmeter-master"
	workerImage                 = "hellofreshtech/kangal-jmeter-worker"
	imageTag                    = "latest"
)

// JMeter enables the controller to run a loadtest using JMeter
type JMeter struct {
	kubeClientSet    kubernetes.Interface
	kangalClientSet  clientSetV.Interface
	loadTest         *loadTestV1.LoadTest
	logger           *zap.Logger
	namespacesLister coreListersV1.NamespaceLister
	reportURL        string
	masterResources  helper.Resources
	workerResources  helper.Resources

	podAnnotations, namespaceAnnotations map[string]string
}

//New initializes new JMeter provider handler to manage load test resources with Kangal Controller
func New(
	kubeClientSet kubernetes.Interface,
	kangalClientSet clientSetV.Interface,
	lt *loadTestV1.LoadTest,
	logger *zap.Logger,
	namespacesLister coreListersV1.NamespaceLister,
	reportURL string,
	podAnnotations, namespaceAnnotations map[string]string,
	config Config,
) *JMeter {
	return &JMeter{
		kubeClientSet:        kubeClientSet,
		kangalClientSet:      kangalClientSet,
		loadTest:             lt,
		logger:               logger,
		namespacesLister:     namespacesLister,
		reportURL:            reportURL,
		podAnnotations:       podAnnotations,
		namespaceAnnotations: namespaceAnnotations,
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
	}
}

// CheckOrCreateResources check if JMeter kubernetes resources have been create,
// if they have not been create them
func (c *JMeter) CheckOrCreateResources(ctx context.Context) error {
	JMeterServices, err := c.kubeClientSet.CoreV1().Services(c.loadTest.Status.Namespace).List(ctx, metaV1.ListOptions{})
	if err != nil {
		return err
	}

	if len(JMeterServices.Items) == 0 {
		_, err = c.kubeClientSet.CoreV1().ConfigMaps(c.loadTest.Status.Namespace).Create(ctx, c.NewConfigMap(), metaV1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			c.logger.Error("Error on creating testfile configmap", zap.Error(err))
			return err
		}

		_, err = c.kubeClientSet.CoreV1().ConfigMaps(c.loadTest.Status.Namespace).Create(ctx, c.NewJMeterSettingsConfigMap(), metaV1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			return err
		}

		secret, err := c.NewSecret()
		if err != nil {
			c.logger.Error("Error on creating environment variables secret", zap.Error(err))
			c.loadTest.Status.Phase = loadTestV1.LoadTestErrored
			return nil
		}

		_, err = c.kubeClientSet.CoreV1().Secrets(c.loadTest.Status.Namespace).Create(ctx, secret, metaV1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			return err
		}

		configMaps, err := c.NewTestdataConfigMap()
		if err != nil {
			c.logger.Error("Error on creating testdata configMaps", zap.Error(err))
			c.loadTest.Status.Phase = loadTestV1.LoadTestErrored
			return nil
		}

		err = c.createPodsWithTestdata(ctx, configMaps, c.loadTest, c.loadTest.Status.Namespace)
		if err != nil {
			return err
		}

		_, err = c.kubeClientSet.CoreV1().Services(c.loadTest.Status.Namespace).Create(ctx, c.NewJMeterService(), metaV1.CreateOptions{})
		if err != nil && !errors.IsAlreadyExists(err) {
			c.logger.Error("Error on creating new JMeter service", zap.Error(err))
			return err
		}
		c.logger.Info(
			"Created JMeter resources",
			zap.String("LoadTest", c.loadTest.GetName()),
			zap.String("namespace", c.loadTest.Status.Namespace),
		)

	}
	return nil
}

// CheckOrUpdateStatus check the JMeter resources and calculate the current
// status of the loadtest from them
func (c *JMeter) CheckOrUpdateStatus(ctx context.Context) error {
	// Get the Namespace resource
	namespace, err := c.namespacesLister.Get(c.loadTest.Status.Namespace)
	if err != nil {
		// The LoadTest resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			c.loadTest.Status.Phase = loadTestV1.LoadTestFinished
			return nil
		}
	}

	if c.loadTest.Status.Phase == "" {
		c.loadTest.Status.Phase = loadTestV1.LoadTestCreating
	}

	if c.loadTest.Status.Phase == loadTestV1.LoadTestErrored {
		return nil
	}

	pods, err := c.kubeClientSet.CoreV1().Pods(namespace.GetName()).List(ctx, metaV1.ListOptions{
		LabelSelector: loadTestWorkerLabelSelector,
	})
	if err != nil {
		return err
	}

	if len(pods.Items) != int(*c.loadTest.Spec.DistributedPods) {
		return nil
	}

	for _, pod := range pods.Items {
		if pod.Status.Phase != "Running" {
			// If the pod is not yet in the running phase check to see if the
			// pod start date is greater than the start time.
			if workerPodHasTimeout(pod.Status.StartTime, c.loadTest.Status) {
				c.loadTest.Status.Phase = loadTestV1.LoadTestFinished
				return nil
			}

			for _, containerStatus := range pod.Status.ContainerStatuses {
				if containerStatus.State.Waiting == nil {
					c.loadTest.Status.Phase = loadTestV1.LoadTestErrored
					return nil
				}
				if containerStatus.State.Waiting.Reason != "Pending" &&
					containerStatus.State.Waiting.Reason != "ContainerCreating" &&
					containerStatus.State.Waiting.Reason != "PodInitializing" {
					c.logger.Info(
						"One of containers is unhealthy, marking LoadTest as errored",
						zap.String("LoadTest", c.loadTest.GetName()),
						zap.String("pod", pod.Name),
						zap.String("namespace", namespace.GetName()),
					)
					c.loadTest.Status.Phase = loadTestV1.LoadTestErrored
					return nil
				}
			}

			return nil
		}
	}

	job, err := c.kubeClientSet.BatchV1().Jobs(namespace.GetName()).Get(ctx, loadTestJobName, metaV1.GetOptions{})
	if err != nil {
		// The LoadTest resource may no longer exist, in which case we stop
		// processing.
		if errors.IsNotFound(err) {
			_, err = c.kubeClientSet.BatchV1().Jobs(namespace.GetName()).Create(
				ctx,
				c.NewJMeterMasterJob(c.reportURL, c.podAnnotations),
				metaV1.CreateOptions{},
			)
			return err
		}
		return err
	}

	// Get jmeter job in namespace and update the LoadTest status with
	// the Job status
	c.loadTest.Status.Phase = getLoadTestPhaseFromJob(job.Status)
	c.loadTest.Status.JobStatus = job.Status

	return nil
}

func (c *JMeter) createPodsWithTestdata(ctx context.Context, configMaps []*coreV1.ConfigMap, loadtest *loadTestV1.LoadTest, namespace string) error {
	for i, cm := range configMaps {
		configMap, err := c.kubeClientSet.CoreV1().ConfigMaps(namespace).Create(ctx, cm, metaV1.CreateOptions{})
		if err != nil {
			c.logger.Error("Error on creating testdata configMaps", zap.Error(err))
			return err
		}

		_, err = c.kubeClientSet.CoreV1().Pods(namespace).Create(ctx, c.NewPod(i, configMap, c.podAnnotations), metaV1.CreateOptions{})
		if err != nil {
			c.logger.Error("Error on creating distributed pods", zap.Error(err))
			return err
		}
	}
	c.logger.Info("Created pods with test data", zap.String("LoadTest", loadtest.GetName()), zap.String("namespace", namespace))
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
