package jmeter

import (
	"context"
	"strings"
	"testing"

	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/informers"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	loadtestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"

	"github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSetDefaults(t *testing.T) {
	t.Run("With env default", func(t *testing.T) {
		jmeter := &JMeter{
			envConfig: &Config{
				MasterImageName: "my-master-image-name",
				MasterImageTag:  "my-master-image-tag",
				WorkerImageName: "my-worker-image-name",
				WorkerImageTag:  "my-worker-image-tag",
			},
		}
		jmeter.SetDefaults()

		assert.Equal(t, jmeter.workerConfig.Image, "my-worker-image-name")
		assert.Equal(t, jmeter.workerConfig.Tag, "my-worker-image-tag")
		assert.Equal(t, jmeter.masterConfig.Image, "my-master-image-name")
		assert.Equal(t, jmeter.masterConfig.Tag, "my-master-image-tag")
	})

	t.Run("No default", func(t *testing.T) {
		jmeter := &JMeter{
			envConfig: &Config{},
		}
		jmeter.SetDefaults()

		assert.Equal(t, jmeter.masterConfig.Image, defaultMasterImageName)
		assert.Equal(t, jmeter.masterConfig.Tag, defaultMasterImageTag)
		assert.Equal(t, jmeter.workerConfig.Image, defaultWorkerImageName)
		assert.Equal(t, jmeter.workerConfig.Tag, defaultWorkerImageTag)
	})
}

func TestTransformLoadTestSpec(t *testing.T) {
	jmeter := &JMeter{
		masterConfig: loadTestV1.ImageDetails{
			Image: "master-image",
			Tag:   "master-tag",
		},
		workerConfig: loadTestV1.ImageDetails{
			Image: "worker-image",
			Tag:   "worker-tag",
		},
	}

	spec := &loadTestV1.LoadTestSpec{}

	t.Run("Empty spec", func(t *testing.T) {
		err := jmeter.TransformLoadTestSpec(spec)
		assert.EqualError(t, err, ErrRequireMinOneDistributedPod.Error())
	})

	t.Run("Negative distributedPods", func(t *testing.T) {
		distributedPods := int32(-10)
		spec.DistributedPods = &distributedPods
		err := jmeter.TransformLoadTestSpec(spec)
		assert.EqualError(t, err, ErrRequireMinOneDistributedPod.Error())
	})

	t.Run("Empty testFile", func(t *testing.T) {
		distributedPods := int32(2)
		spec.DistributedPods = &distributedPods
		err := jmeter.TransformLoadTestSpec(spec)
		assert.EqualError(t, err, ErrRequireTestFile.Error())
	})

	t.Run("All valid", func(t *testing.T) {
		distributedPods := int32(2)
		spec.DistributedPods = &distributedPods
		spec.TestFile = "my-test"
		err := jmeter.TransformLoadTestSpec(spec)
		assert.NoError(t, err)
		assert.Equal(t, spec.MasterConfig.Image, "master-image")
		assert.Equal(t, spec.MasterConfig.Tag, "master-tag")
		assert.Equal(t, spec.WorkerConfig.Image, "worker-image")
		assert.Equal(t, spec.WorkerConfig.Tag, "worker-tag")
	})
}

func TestCheckForTimeout(t *testing.T) {
	// subtract 10 minutes from the current time
	now := metaV1.Now()
	maxWaitTime := metaV1.Now().Add(MaxWaitTimeForPods * -1)
	past := metaV1.Time{Time: maxWaitTime}

	var tests = []struct {
		Time           *metaV1.Time
		LoadTestStatus loadtestV1.LoadTestStatus
		Expected       bool
	}{
		{
			// Less than MaxWaitTimeForPods
			Time: &now,
			LoadTestStatus: loadtestV1.LoadTestStatus{
				Phase: loadtestV1.LoadTestCreating,
			},
			Expected: false,
		},
		{
			// pod was created MaxWaitTimeForPods and still in creation phase
			Time: &past,
			LoadTestStatus: loadtestV1.LoadTestStatus{
				Phase: loadtestV1.LoadTestCreating,
			},
			Expected: true,
		},
		{
			// Pod has been up for more than MaxWaitTimeForPods, but the test is running
			Time: &past,
			LoadTestStatus: loadtestV1.LoadTestStatus{
				Phase: loadtestV1.LoadTestRunning,
			},
			Expected: false,
		},
		{
			// Pod has been up for more than MaxWaitTimeForPods, but the test is running
			Time: nil,
			LoadTestStatus: loadtestV1.LoadTestStatus{
				Phase: loadtestV1.LoadTestCreating,
			},
			Expected: false,
		},
	}

	for _, test := range tests {
		r := workerPodHasTimeout(test.Time, test.LoadTestStatus)
		assert.Equal(t, test.Expected, r)
	}
}

func TestGetLoadTestPhaseFromJob(t *testing.T) {
	var testPhases = []struct {
		ExpectedPhase loadtestV1.LoadTestPhase
		JobStatus     batchV1.JobStatus
	}{
		{
			loadtestV1.LoadTestStarting,
			batchV1.JobStatus{
				Active: 0,
			},
		},
		{
			loadtestV1.LoadTestRunning,
			batchV1.JobStatus{
				Active: 1,
			},
		},
		{
			loadtestV1.LoadTestRunning,
			batchV1.JobStatus{
				Active: 1,
				Failed: 1,
			},
		},
		{
			loadtestV1.LoadTestFinished,
			batchV1.JobStatus{
				Active: 0,
				Failed: 2,
			},
		},
		{
			loadtestV1.LoadTestFinished,
			batchV1.JobStatus{
				Active:    0,
				Succeeded: 1,
				Failed:    0,
			},
		},
	}

	for _, test := range testPhases {
		phase := getLoadTestPhaseFromJob(test.JobStatus)
		assert.Equal(t, test.ExpectedPhase, phase)
	}
}

func TestSync(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Arbitrary test values
	distributedPodsNum := int32(2)
	namespace := "loadtest-namespace"

	// Fake clients
	kubeClient := k8sfake.NewSimpleClientset()
	client := fake.NewSimpleClientset()
	informers := informers.NewSharedInformerFactory(kubeClient, 0)
	namespaceLister := informers.Core().V1().Namespaces().Lister()
	logger, _ := zap.NewDevelopment()

	c := &JMeter{
		kubeClientSet:   kubeClient,
		kangalClientSet: client,
		logger:          logger,
		namespaceLister: namespaceLister,
	}

	loadTest := loadTestV1.LoadTest{
		Spec: loadtestV1.LoadTestSpec{
			DistributedPods: &distributedPodsNum,
		},
		Status: loadtestV1.LoadTestStatus{
			Namespace: namespace,
		},
	}

	err := c.Sync(ctx, loadTest, "")
	require.NoError(t, err, "Error when Sync")

	services, err := kubeClient.CoreV1().Services(namespace).List(ctx, metaV1.ListOptions{})
	require.NoError(t, err, "Error when listing services")
	assert.NotZero(t, len(services.Items), "Expected non-zero service amount after CheckOrCreateResources but found zero services")
}

func TestSyncStatus(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Arbitrary test values
	distributedPodsNum := int32(2)
	namespace := &coreV1.Namespace{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "loadtest-namespace",
		},
	}

	// Fake clients
	kubeClient := k8sfake.NewSimpleClientset()
	client := fake.NewSimpleClientset()
	informer := informers.NewSharedInformerFactory(kubeClient, 0)
	namespaceLister := informer.Core().V1().Namespaces().Lister()
	logger, _ := zap.NewDevelopment()

	// Fake state
	var podPhase coreV1.PodPhase
	var podContainersReason string

	labelSelect := strings.Split(LoadTestWorkerLabelSelector, "=")

	// Fake responses
	listPodsReactFunc := func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		obj := &coreV1.PodList{}
		for i := int32(0); i < distributedPodsNum; i++ {
			pod := coreV1.Pod{
				ObjectMeta: metaV1.ObjectMeta{
					Labels: map[string]string{
						labelSelect[0]: labelSelect[1],
					},
				},
				Status: coreV1.PodStatus{
					Phase: podPhase,
					ContainerStatuses: []coreV1.ContainerStatus{
						{
							State: coreV1.ContainerState{
								Waiting: &coreV1.ContainerStateWaiting{
									Reason: podContainersReason,
								},
							},
						},
					},
				},
			}
			obj.Items = append(obj.Items, pod)
		}
		return true, obj, nil
	}
	getJobsReactFunc := func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &batchV1.Job{
			ObjectMeta: metaV1.ObjectMeta{
				Name: "loadtest-master",
			},
			Status: batchV1.JobStatus{
				Active:    0,
				Succeeded: 2,
				Failed:    0,
			},
		}, nil
	}

	loadTest := loadTestV1.LoadTest{
		Spec: loadtestV1.LoadTestSpec{
			DistributedPods: &distributedPodsNum,
		},
		Status: loadtestV1.LoadTestStatus{
			Namespace: namespace.GetName(),
			Phase:     "",
		},
	}

	c := &JMeter{
		kubeClientSet:   kubeClient,
		kangalClientSet: client,
		logger:          logger,
		namespaceLister: namespaceLister,
	}

	t.Run("No namespace", func(t *testing.T) {
		// test initial scenario
		err := c.SyncStatus(ctx, loadTest, &loadTest.Status)
		require.NoError(t, err, "Error when SyncStatus")
		assert.Equal(t, loadtestV1.LoadTestFinished, loadTest.Status.Phase)
	})

	t.Run("With namespace", func(t *testing.T) {
		// reset
		loadTest.Status.Phase = ""

		// change scenario
		informer.Core().V1().Namespaces().Informer().GetIndexer().Add(namespace)

		// test scenario
		err := c.SyncStatus(ctx, loadTest, &loadTest.Status)
		require.NoError(t, err, "Error when SyncStatus")
		assert.Equal(t, loadtestV1.LoadTestCreating, loadTest.Status.Phase)
	})

	t.Run("Pods errored", func(t *testing.T) {
		// reset
		loadTest.Status.Phase = loadtestV1.LoadTestRunning

		// change scenario
		podPhase = coreV1.PodFailed
		podContainersReason = "Errored"
		kubeClient.PrependReactor("list", "pods", listPodsReactFunc)
		kubeClient.PrependReactor("get", "jobs", getJobsReactFunc)

		// test scenario
		err := c.SyncStatus(ctx, loadTest, &loadTest.Status)
		require.NoError(t, err, "Error when SyncStatus")
		assert.Equal(t, loadtestV1.LoadTestErrored, loadTest.Status.Phase)
	})

	t.Run("Pods running and job completed", func(t *testing.T) {
		// reset
		loadTest.Status.Phase = loadtestV1.LoadTestRunning

		// change scenario
		podPhase = coreV1.PodRunning

		// test scenario
		err := c.SyncStatus(ctx, loadTest, &loadTest.Status)
		require.NoError(t, err, "Error when SyncStatus")
		assert.Equal(t, loadtestV1.LoadTestFinished, loadTest.Status.Phase)
	})
}
