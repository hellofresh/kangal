package controller

import (
	"context"
	"testing"

	loadtestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIntegrationKangalController(t *testing.T) {
	// This integration test cover main idea and logic about Kangal controller
	// First of all it creates new LoadTest resource, then it expects that Kangal Controller created resources
	// and changed LoadTest status to "finished".
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	t.Log()

	// TODO: those attributes should gone once we do improvements on proxy side and move kube_client to own kube package
	distributedPods := int32(1)
	loadtestType := loadtestV1.LoadTestTypeFake
	expectedLoadtestName := "loadtest-fake-integration"
	testFile := "testdata/valid/loadtest.jmx"
	envVars := "testdata/valid/envvars.csv"
	testData := "testdata/valid/testdata.csv"

	client := kubeClient(t)

	err := CreateLoadtest(clientSet, distributedPods, expectedLoadtestName, testFile, testData, envVars, loadtestType)
	require.NoError(t, err)
	t.Cleanup(func() {
		err := DeleteLoadTest(clientSet, expectedLoadtestName, t.Name())
		assert.NoError(t, err)
	})
	var ltNamespace *coreV1.Namespace

	t.Run("Checking the name of created loadtest", func(t *testing.T) {
		createdName, err := GetLoadtest(clientSet, expectedLoadtestName)
		require.NoError(t, err)
		assert.Equal(t, expectedLoadtestName, createdName)
	})

	t.Run("Checking namespace is created", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			WaitForResource(LongWaitSec)
			ltNamespace, _ = client.CoreV1().Namespaces().Get(context.Background(), expectedLoadtestName, metaV1.GetOptions{})
			if ltNamespace != nil {
				break
			}
		}
		// loadtest namespace name is equal to loadtest name
		require.Equal(t, expectedLoadtestName, ltNamespace.Name)
	})

	t.Run("Checking master pod is created", func(t *testing.T) {
		var master coreV1.PodList
		for i := 0; i < 5; i++ {
			WaitForResource(ShortWaitSec)
			master, _ = GetMasterPod(client.CoreV1(), expectedLoadtestName)
			if master.Items[0].Status.Phase == "Running" {
				break
			}
		}
		assert.Equal(t, "Running", string(master.Items[0].Status.Phase))
	})

	t.Run("Checking loadtest is in Running state", func(t *testing.T) {
		var phase string
		phase, err = GetLoadtestPhase(clientSet, expectedLoadtestName)
		require.NoError(t, err)
		assert.Equal(t, string(loadtestV1.LoadTestRunning), phase)
	})

	t.Run("Checking loadtest is in Finished state", func(t *testing.T) {
		// We do run fake provider which has 10 sec sleep before finishing job
		var phase string
		for i := 0; i < 5; i++ {
			WaitForResource(LongWaitSec)
			phase, err = GetLoadtestPhase(clientSet, expectedLoadtestName)
			require.NoError(t, err)
			if phase == string(loadtestV1.LoadTestFinished) {
				break
			}
		}
		assert.Equal(t, string(loadtestV1.LoadTestFinished), phase)
	})
}
