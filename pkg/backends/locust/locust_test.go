package locust

import (
	"context"
	"testing"

	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8sfake "k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestSetDefaults(t *testing.T) {
	t.Run("With env default", func(t *testing.T) {
		locust := &Locust{
			envConfig: &Config{
				ImageName: "my-image-name",
				ImageTag:  "my-image-tag",
			},
		}
		locust.SetDefaults()

		assert.Equal(t, locust.config.Image, "my-image-name")
		assert.Equal(t, locust.config.Tag, "my-image-tag")
	})

	t.Run("No default", func(t *testing.T) {
		jmeter := &Locust{
			envConfig: &Config{},
		}
		jmeter.SetDefaults()

		assert.Equal(t, jmeter.config.Image, defaultImageName)
		assert.Equal(t, jmeter.config.Tag, defaultImageTag)
	})
}

func TestTransformLoadTestSpec(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	locust := &Locust{
		logger:        logger,
		kubeClientSet: nil, // it's always nil when TransformLoadTestSpec is called
	}

	distributedPods := int32(1)

	loadTestSpec := loadTestV1.LoadTestSpec{
		DistributedPods: &distributedPods,
		TestFile:        "something",
	}

	err := locust.TransformLoadTestSpec(&loadTestSpec)
	require.NoError(t, err, "Error when TransformLoadTestSpec")
}

func TestSync(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Fake clients
	kubeClient := k8sfake.NewSimpleClientset()
	logger, _ := zap.NewDevelopment()

	namespace := "test"
	distributedPods := int32(1)

	loadTest := loadTestV1.LoadTest{
		Spec: loadTestV1.LoadTestSpec{
			DistributedPods: &distributedPods,
			EnvVars:         "key1,val2\n",
		},
		Status: loadTestV1.LoadTestStatus{
			Namespace: namespace,
		},
	}

	locust := &Locust{
		logger:        logger,
		kubeClientSet: kubeClient,
	}

	err := locust.Sync(ctx, loadTest, "")
	require.NoError(t, err, "Error when Sync")

	services, err := kubeClient.CoreV1().Services(namespace).List(ctx, metaV1.ListOptions{})
	require.NoError(t, err, "Error when listing services")
	assert.NotZero(t, len(services.Items), "Expected non-zero services amount after Sync but found zero")

	configMaps, err := kubeClient.CoreV1().ConfigMaps(namespace).List(ctx, metaV1.ListOptions{})
	require.NoError(t, err, "Error when listing services")
	assert.NotZero(t, len(configMaps.Items), "Expected non-zero configMaps amount after Sync but found zero")
}

func TestSyncStatus(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Fake clients
	kubeClient := k8sfake.NewSimpleClientset()
	logger, _ := zap.NewDevelopment()

	namespace := "test"
	distributedPods := int32(1)

	loadTest := loadTestV1.LoadTest{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "mytest",
		},
		Spec: loadTestV1.LoadTestSpec{
			DistributedPods: &distributedPods,
		},
		Status: loadTestV1.LoadTestStatus{
			Namespace: namespace,
		},
	}

	locust := &Locust{
		logger:        logger,
		kubeClientSet: kubeClient,
	}

	t.Run("No configmap or master/worker job", func(t *testing.T) {
		err := locust.SyncStatus(ctx, loadTest, &loadTest.Status)
		require.NoError(t, err, "Error when SyncStatus")
		assert.Equal(t, loadTestV1.LoadTestFinished, loadTest.Status.Phase)
	})

	t.Run("No configmap or master/worker job", func(t *testing.T) {
		kubeClient.PrependReactor("get", "configmaps", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, &coreV1.ConfigMap{
				ObjectMeta: metaV1.ObjectMeta{
					Name: "mytest-testfile",
				},
			}, nil
		})

		kubeClient.PrependReactor("get", "jobs", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, &batchV1.Job{
				ObjectMeta: metaV1.ObjectMeta{
					Name: action.(k8stesting.GetActionImpl).Name,
				},
				Status: batchV1.JobStatus{
					Active:    0,
					Succeeded: 2,
					Failed:    0,
				},
			}, nil
		})

		// reset
		loadTest.Status = loadTestV1.LoadTestStatus{
			Namespace: namespace,
		}

		err := locust.SyncStatus(ctx, loadTest, &loadTest.Status)
		require.NoError(t, err, "Error when SyncStatus")
		assert.Equal(t, loadTestV1.LoadTestFinished, loadTest.Status.Phase)
	})
}
