package jmeter

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	batchV1 "k8s.io/api/batch/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/informers"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	"github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned/fake"
)

func TestCheckForTimeout(t *testing.T) {
	// subtract 10 minutes from the current time
	now := metaV1.Now()
	maxWaitTime := metaV1.Now().Add(MaxWaitTimeForPods * -1)
	past := metaV1.Time{Time: maxWaitTime}

	var tests = []struct {
		Time           *metaV1.Time
		LoadTestStatus loadTestV1.LoadTestStatus
		Expected       bool
	}{
		{
			// Less than MaxWaitTimeForPods
			Time: &now,
			LoadTestStatus: loadTestV1.LoadTestStatus{
				Phase: loadTestV1.LoadTestCreating,
			},
			Expected: false,
		},
		{
			// pod was created MaxWaitTimeForPods and still in creation phase
			Time: &past,
			LoadTestStatus: loadTestV1.LoadTestStatus{
				Phase: loadTestV1.LoadTestCreating,
			},
			Expected: true,
		},
		{
			// Pod has been up for more than MaxWaitTimeForPods, but the test is running
			Time: &past,
			LoadTestStatus: loadTestV1.LoadTestStatus{
				Phase: loadTestV1.LoadTestRunning,
			},
			Expected: false,
		},
		{
			// Pod has been up for more than MaxWaitTimeForPods, but the test is running
			Time: nil,
			LoadTestStatus: loadTestV1.LoadTestStatus{
				Phase: loadTestV1.LoadTestCreating,
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
		ExpectedPhase loadTestV1.LoadTestPhase
		JobStatus     batchV1.JobStatus
	}{
		{
			loadTestV1.LoadTestStarting,
			batchV1.JobStatus{
				Active: 0,
			},
		},
		{
			loadTestV1.LoadTestRunning,
			batchV1.JobStatus{
				Active: 1,
			},
		},
		{
			loadTestV1.LoadTestRunning,
			batchV1.JobStatus{
				Active: 1,
				Failed: 1,
			},
		},
		{
			loadTestV1.LoadTestFinished,
			batchV1.JobStatus{
				Active: 0,
				Failed: 2,
			},
		},
		{
			loadTestV1.LoadTestFinished,
			batchV1.JobStatus{
				Active:    0,
				Succeeded: 1,
				Failed:    0,
			},
		},
	}

	for _, test := range testPhases {
		phase := determineLoadTestPhaseFromJob(test.JobStatus)
		assert.Equal(t, test.ExpectedPhase, phase)
	}
}

func TestSync(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Arbitrary test values
	distributedPodsNum := int32(2)
	waitForResourceTimeout := 3 * time.Second
	namespace := "loadtest-namespace"
	reportURL := "http://kangal-proxy.local/load-test/loadtest-name/report"

	// Fake clients
	logger := zaptest.NewLogger(t)
	kubeClient := k8sfake.NewSimpleClientset()
	client := fake.NewSimpleClientset()
	namespaceLister := informers.NewSharedInformerFactory(kubeClient, 0).Core().V1().Namespaces().Lister()

	loadTest := loadTestV1.LoadTest{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "loadtest-name",
		},
		Spec: loadTestV1.LoadTestSpec{
			DistributedPods: &distributedPodsNum,
			MasterConfig: loadTestV1.ImageDetails{
				Image: defaultMasterImageName,
				Tag:   defaultMasterImageTag,
			},
			WorkerConfig: loadTestV1.ImageDetails{
				Image: defaultWorkerImageName,
				Tag:   defaultWorkerImageTag,
			},
		},
		Status: loadTestV1.LoadTestStatus{
			Phase:     "",
			Namespace: namespace,
			JobStatus: batchV1.JobStatus{},
			Pods:      loadTestV1.LoadTestPodsStatus{},
		},
	}

	b := Backend{
		kubeClientSet:   kubeClient,
		kangalClientSet: client,
		logger:          logger,
		namespaceLister: namespaceLister,
		config: &Config{
			WaitForResourceTimeout: waitForResourceTimeout,
		},
	}

	err := b.Sync(ctx, loadTest, reportURL, []string{}, "")
	require.NoError(t, err, "Error when syncing")

	services, err := kubeClient.CoreV1().Services(namespace).List(ctx, metaV1.ListOptions{})
	require.NoError(t, err, "Error when listing services")
	assert.NotEmpty(t, services.Items, "Expected non-zero service amount after CheckOrCreateResources but found zero services")
}

func TestTransformLoadTestSpec(t *testing.T) {
	jmeter := &Backend{
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
		spec.TestFile = []byte("my-test")
		err := jmeter.TransformLoadTestSpec(spec)
		assert.NoError(t, err)
		assert.Equal(t, spec.MasterConfig.Image, "master-image")
		assert.Equal(t, spec.MasterConfig.Tag, "master-tag")
		assert.Equal(t, spec.WorkerConfig.Image, "worker-image")
		assert.Equal(t, spec.WorkerConfig.Tag, "worker-tag")
	})
}

func TestSetDefaults(t *testing.T) {
	t.Run("With env default", func(t *testing.T) {
		jmeter := &Backend{
			config: &Config{
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
		jmeter := &Backend{
			config: &Config{},
		}
		jmeter.SetDefaults()

		assert.Equal(t, jmeter.masterConfig.Image, defaultMasterImageName)
		assert.Equal(t, jmeter.masterConfig.Tag, defaultMasterImageTag)
		assert.Equal(t, jmeter.workerConfig.Image, defaultWorkerImageName)
		assert.Equal(t, jmeter.workerConfig.Tag, defaultWorkerImageTag)
	})
}
