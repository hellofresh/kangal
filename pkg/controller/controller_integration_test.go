package controller

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	_ "github.com/hellofresh/kangal/pkg/backends/fake"
	"github.com/hellofresh/kangal/pkg/core/waitfor"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

func TestIntegrationKangalController(t *testing.T) {
	// This integration test covers main workflow of Kangal controller.
	// First of all test_helper creates a new LoadTest object.
	// After this Kangal controller creates all associated resources
	// and updates LoadTest statuses from "created" to "finished".
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	expectedLoadtestName := "loadtest-fake-integration"
	waitForResourceTimeout := 30 * time.Second

	// TODO: those attributes should gone once we do improvements on proxy side and move kube_client to own kube package
	distributedPods := int32(1)
	loadtestType := loadTestV1.LoadTestTypeFake
	testFile := "testdata/valid/loadtest.jmx"
	envVars := map[string]string{"foo": "bar", "foo2": "bar2"}
	testData := "testdata/valid/testdata.csv"

	client := kubeClient(t)

	err := CreateLoadTest(clientSet, distributedPods, expectedLoadtestName, testFile, testData, envVars, loadtestType)
	require.NoError(t, err)

	err = WaitLoadTest(clientSet, expectedLoadtestName)
	require.NoError(t, err)

	t.Cleanup(func() {
		if t.Failed() {
			err := DeleteLoadTest(clientSet, expectedLoadtestName, t.Name())
			assert.NoError(t, err)
		}
	})

	// Checking the name of created loadtest
	createdName, err := GetLoadTest(clientSet, expectedLoadtestName)
	require.NoError(t, err)
	require.Equal(t, expectedLoadtestName, createdName)

	// Checking namespace is created
	watchObj, _ := client.CoreV1().Namespaces().Watch(context.Background(), metaV1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", expectedLoadtestName),
	})
	watchEvent, err := waitfor.Resource(watchObj, (waitfor.Condition{}).Added, waitForResourceTimeout)
	require.NoError(t, err)
	namespace := watchEvent.Object.(*coreV1.Namespace)
	require.Equal(t, expectedLoadtestName, namespace.Name)

	// Checking master pod is created
	watchObj, _ = client.CoreV1().Pods(expectedLoadtestName).Watch(context.Background(), metaV1.ListOptions{
		LabelSelector: "app=loadtest-master",
	})
	watchEvent, err = waitfor.Resource(watchObj, (waitfor.Condition{}).PodRunning, waitForResourceTimeout)
	require.NoError(t, err)
	pod := watchEvent.Object.(*coreV1.Pod)
	require.Equal(t, coreV1.PodRunning, pod.Status.Phase)

	// Checking loadtest is in Running state
	watchObj, _ = clientSet.KangalV1().LoadTests().Watch(context.Background(), metaV1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", expectedLoadtestName),
	})
	watchEvent, err = waitfor.Resource(watchObj, (waitfor.Condition{}).LoadTestRunning, waitForResourceTimeout)
	require.NoError(t, err)
	loadtest := watchEvent.Object.(*loadTestV1.LoadTest)
	require.Equal(t, loadTestV1.LoadTestRunning, loadtest.Status.Phase)

	// Checking loadtest is in Finished state
	// We use loadtests with fake provider which only runs for 10 sec and then finishes the job
	watchObj, _ = clientSet.KangalV1().LoadTests().Watch(context.Background(), metaV1.ListOptions{
		FieldSelector: fmt.Sprintf("metadata.name=%s", expectedLoadtestName),
	})
	watchEvent, err = waitfor.Resource(watchObj, (waitfor.Condition{}).LoadTestFinished, waitForResourceTimeout)
	require.NoError(t, err)
	loadtest = watchEvent.Object.(*loadTestV1.LoadTest)
	require.Equal(t, loadTestV1.LoadTestFinished, loadtest.Status.Phase)

	// Checking finished loadtest is deleted
	// SyncHandler runs every 10s for integration test.
	// We expect SyncHandler to delete finished loadtest after 10s but wait 40s and check every 5s
	var deleted bool
	for i := 0; i < 8; i++ {
		time.Sleep(5 * time.Second)
		lt, _ := clientSet.KangalV1().LoadTests().Get(context.Background(), expectedLoadtestName, metaV1.GetOptions{})
		// assert that the returned object is empty which means lt "loadtest-fake-integration" was deleted
		if lt.Name == "" && lt.Namespace == "" {
			deleted = true
			break
		}
	}
	assert.True(t, deleted, "Looks like test was not deleted during expected time frame")
}
