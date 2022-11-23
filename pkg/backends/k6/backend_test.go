package k6

import (
	"context"
	"testing"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	batchV1 "k8s.io/api/batch/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestSync(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kubeClient := k8sfake.NewSimpleClientset()
	logger := zaptest.NewLogger(t)

	namespace := "test"
	distributedPods := int32(1)
	reportURL := "http://kangal-proxy.local/load-test/loadtest-name/report"

	loadTest := loadTestV1.LoadTest{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "loadtest-name",
		},
		Spec: loadTestV1.LoadTestSpec{
			DistributedPods: &distributedPods,
			TestFile:        []byte("test"),
		},
		Status: loadTestV1.LoadTestStatus{
			Phase:     "running",
			Namespace: namespace,
			JobStatus: batchV1.JobStatus{},
			Pods:      loadTestV1.LoadTestPodsStatus{},
		},
	}

	b := Backend{
		logger:        logger,
		kubeClientSet: kubeClient,
	}

	err := b.Sync(ctx, loadTest, reportURL)
	require.NoError(t, err, "Error when CheckOrCreateResources")

	jobs, err := kubeClient.BatchV1().Jobs(namespace).List(ctx, metaV1.ListOptions{})
	require.NoError(t, err, "Error when listing jobs")
	assert.NotEmpty(t, jobs.Items, "Expected job to be created but there's none")

	configMaps, err := kubeClient.CoreV1().ConfigMaps(namespace).List(ctx, metaV1.ListOptions{})
	require.NoError(t, err, "Error when listing configmaps")
	assert.NotEmpty(t, configMaps.Items, "Expected configmap to be created but there's none")
}

func TestSyncStatus(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kubeClient := k8sfake.NewSimpleClientset()
	logger := zaptest.NewLogger(t)

	namespace := "test"
	distributedPods := int32(1)
	index := int32(0)
	reportURL := "http://kangal-proxy.local/load-test/loadtest-name/report"

	loadTest := loadTestV1.LoadTest{
		ObjectMeta: metaV1.ObjectMeta{
			Name: jobName(index),
		},
		Spec: loadTestV1.LoadTestSpec{
			DistributedPods: &distributedPods,
			TestFile:        []byte("test"),
		},
		Status: loadTestV1.LoadTestStatus{
			Namespace: namespace,
			JobStatus: batchV1.JobStatus{},
			Pods:      loadTestV1.LoadTestPodsStatus{},
		},
	}

	b := Backend{
		logger:        logger,
		kubeClientSet: kubeClient,
	}

	// First sync should update status to creating
	err := b.Sync(ctx, loadTest, reportURL)
	require.NoError(t, err, "Sync error")

	err = b.SyncStatus(ctx, loadTest, &loadTest.Status)
	require.NoError(t, err, "SyncStatus error")
	assert.Equal(t, loadTestV1.LoadTestCreating, loadTest.Status.Phase)

	// Simulate that job finished successfully
	job, err := kubeClient.BatchV1().Jobs(namespace).Get(ctx, jobName(index), metaV1.GetOptions{})
	require.NoError(t, err, "Error when getting jobs")
	job.Status.Succeeded = 1
	_, err = kubeClient.BatchV1().Jobs(namespace).UpdateStatus(ctx, job, metaV1.UpdateOptions{})
	require.NoError(t, err, "UpdateStatus error")

	// Sync should now update loadtest status to finished
	err = b.SyncStatus(ctx, loadTest, &loadTest.Status)
	require.NoError(t, err, "SyncStatus error")
	assert.Equal(t, loadTestV1.LoadTestFinished, loadTest.Status.Phase)
}
