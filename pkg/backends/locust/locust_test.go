package locust

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	batchV1 "k8s.io/api/batch/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	loadtestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	"github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned/fake"
)

func TestLocustCheckOrCreateResources(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Fake clients
	kubeClient := k8sfake.NewSimpleClientset()
	kangalClient := fake.NewSimpleClientset()
	logger, _ := zap.NewDevelopment()

	namespace := "test"
	distributedPods := int32(1)

	c := New(
		kubeClient,
		kangalClient,
		&loadtestV1.LoadTest{
			ObjectMeta: metaV1.ObjectMeta{
				Name: "loadtest-name",
			},
			Spec: loadtestV1.LoadTestSpec{
				EnvVars:         "my-secret,my-super-secret\n",
				DistributedPods: &distributedPods,
			},
			Status: loadtestV1.LoadTestStatus{
				Phase:     "running",
				Namespace: namespace,
				JobStatus: batchV1.JobStatus{},
				Pods:      loadtestV1.LoadTestPodsStatus{},
			},
		},
		logger,
		"http://kangal-proxy.local/load-test/loadtest-name/report",
		Config{},
		map[string]string{},
	)

	err := c.CheckOrCreateResources(ctx)
	require.NoError(t, err, "Error when CheckOrCreateResources")

	services, err := kubeClient.CoreV1().Services(namespace).List(ctx, metaV1.ListOptions{})
	require.NoError(t, err, "Error when listing services")
	assert.NotZero(t, len(services.Items), "Expected non-zero services amount after CheckOrCreateResources but found zero")

	configMaps, err := kubeClient.CoreV1().ConfigMaps(namespace).List(ctx, metaV1.ListOptions{})
	require.NoError(t, err, "Error when listing services")
	assert.NotZero(t, len(configMaps.Items), "Expected non-zero configMaps amount after CheckOrCreateResources but found zero")
}

func TestLocustCheckOrUpdateStatus(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Fake clients
	kubeClient := k8sfake.NewSimpleClientset()
	kangalClient := fake.NewSimpleClientset()
	logger, _ := zap.NewDevelopment()

	namespace := "test"
	distributedPods := int32(1)

	c := New(
		kubeClient,
		kangalClient,
		&loadtestV1.LoadTest{
			ObjectMeta: metaV1.ObjectMeta{
				Name: "loadtest-name",
			},
			Spec: loadtestV1.LoadTestSpec{
				DistributedPods: &distributedPods,
			},
			Status: loadtestV1.LoadTestStatus{
				Phase:     "running",
				Namespace: namespace,
				JobStatus: batchV1.JobStatus{},
				Pods:      loadtestV1.LoadTestPodsStatus{},
			},
		},
		logger,
		"http://kangal-proxy.local/load-test/loadtest-name/report",
		Config{},
		map[string]string{"": ""},
	)

	err := c.CheckOrUpdateStatus(ctx)
	require.NoError(t, err, "Error when CheckOrUpdateStatus")

	assert.Equal(t, loadtestV1.LoadTestFinished, c.loadTest.Status.Phase)
}
