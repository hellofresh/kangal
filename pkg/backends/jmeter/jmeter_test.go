package jmeter

import (
	"context"
	"testing"

	"github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	batchV1 "k8s.io/api/batch/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	loadtestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

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

func TestJMeter_CheckOrCreateResources(t *testing.T) {
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

	c := New(
		kubeClient,
		client,
		&loadtestV1.LoadTest{
			ObjectMeta: metaV1.ObjectMeta{
				Name: "loadtest-name",
			},
			Spec: loadtestV1.LoadTestSpec{
				DistributedPods: &distributedPodsNum,
				MasterConfig: loadtestV1.ImageDetails{
					Image: defaultMasterImageName,
					Tag:   defaultMasterImageTag,
				},
				WorkerConfig: loadtestV1.ImageDetails{
					Image: defaultWorkerImageName,
					Tag:   defaultWorkerImageTag,
				},
			},
			Status: loadtestV1.LoadTestStatus{
				Phase:     "",
				Namespace: namespace,
				JobStatus: batchV1.JobStatus{},
				Pods:      loadtestV1.LoadTestPodsStatus{},
			},
		},
		logger,
		namespaceLister,
		"http://kangal-proxy.local/load-test/loadtest-name/report",
		map[string]string{"": ""},
		map[string]string{"": ""},
		Config{},
	)

	c.CheckOrCreateResources(ctx)

	services, err := kubeClient.CoreV1().Services(namespace).List(ctx, metaV1.ListOptions{})
	require.NoError(t, err, "Error when listing services")
	assert.NotZero(t, len(services.Items), "Expected non-zero service amount after CheckOrCreateResources but found zero services")
}
