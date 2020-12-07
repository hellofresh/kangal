package controller

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	_ "github.com/hellofresh/kangal/pkg/backends/fake"
	"github.com/hellofresh/kangal/pkg/core/waitfor"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

func TestIntegrationKangalController(t *testing.T) {
	// This integration test cover main idea and logic about Kangal controller
	// First of all it creates new LoadTest resource, then it expects that Kangal Controller created resources
	// and changed LoadTest status to "finished".
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	expectedLoadtestName := "loadtest-fake-integration"

	// TODO: those attributes should gone once we do improvements on proxy side and move kube_client to own kube package
	distributedPods := int32(1)
	loadtestType := loadTestV1.LoadTestTypeFake
	testFile := "testdata/valid/loadtest.jmx"
	envVars := "testdata/valid/envvars.csv"
	testData := "testdata/valid/testdata.csv"

	client := kubeClient(t)

	err := CreateLoadTest(clientSet, distributedPods, expectedLoadtestName, testFile, testData, envVars, loadtestType)
	require.NoError(t, err)

	t.Cleanup(func() {
		err := DeleteLoadTest(clientSet, expectedLoadtestName, t.Name())
		assert.NoError(t, err)
	})

	t.Run("Checking the name of created loadtest", func(t *testing.T) {
		createdName, err := GetLoadTest(clientSet, expectedLoadtestName)
		require.NoError(t, err)
		assert.Equal(t, expectedLoadtestName, createdName)
	})

	t.Run("Checking namespace is created", func(t *testing.T) {
		watchObj, _ := client.CoreV1().Namespaces().Watch(context.Background(), metaV1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", expectedLoadtestName),
		})

		watchEvent, err := waitfor.Resource(watchObj, (waitfor.Condition{}).Added)
		require.NoError(t, err)

		namespace := watchEvent.Object.(*coreV1.Namespace)
		require.Equal(t, expectedLoadtestName, namespace.Name)
	})

	t.Run("Checking master pod is created", func(t *testing.T) {
		watchObj, _ := client.CoreV1().Pods(expectedLoadtestName).Watch(context.Background(), metaV1.ListOptions{
			LabelSelector: "app=loadtest-master",
		})

		watchEvent, err := waitfor.Resource(watchObj, (waitfor.Condition{}).PodRunning)
		require.NoError(t, err)

		pod := watchEvent.Object.(*coreV1.Pod)
		assert.Equal(t, coreV1.PodRunning, pod.Status.Phase)
	})

	t.Run("Checking loadtest is in Running state", func(t *testing.T) {
		watchObj, _ := clientSet.KangalV1().LoadTests().Watch(context.Background(), metaV1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", expectedLoadtestName),
		})

		watchEvent, err := waitfor.Resource(watchObj, (waitfor.Condition{}).LoadTestRunning)
		require.NoError(t, err)

		loadtest := watchEvent.Object.(*loadTestV1.LoadTest)
		assert.Equal(t, loadTestV1.LoadTestRunning, loadtest.Status.Phase)
	})

	t.Run("Checking loadtest is in Finished state", func(t *testing.T) {
		// We do run fake provider which has 10 sec sleep before finishing job
		watchObj, _ := clientSet.KangalV1().LoadTests().Watch(context.Background(), metaV1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", expectedLoadtestName),
		})

		watchEvent, err := waitfor.Resource(watchObj, (waitfor.Condition{}).LoadTestFinished)
		require.NoError(t, err)

		loadtest := watchEvent.Object.(*loadTestV1.LoadTest)
		assert.Equal(t, loadTestV1.LoadTestFinished, loadtest.Status.Phase)
	})
}
