package controller

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	_ "github.com/hellofresh/kangal/pkg/backends/jmeter"
	"github.com/hellofresh/kangal/pkg/core/waitfor"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	clientSetV "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned"
)

var (
	clientSet clientSetV.Clientset
)

func TestMain(m *testing.M) {
	clientSet = kubeTestClient()
	res := m.Run()

	os.Exit(res)
}

func TestIntegrationJMeter(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	t.Log()

	distributedPods := int32(2)
	waitForResourceTimeout := 60 * time.Second
	loadtestType := loadTestV1.LoadTestTypeJMeter
	expectedLoadtestName := "loadtest-jmeter-integration"
	testFile := "testdata/valid/integration_test.jmx"
	envVars := map[string]string{"foo": "bar", "foo2": "bar2"}
	testData := "testdata/valid/testdata.csv"

	client := kubeClient(t)

	err := CreateLoadTest(clientSet, distributedPods, expectedLoadtestName, testFile, testData, envVars, loadtestType)
	require.NoError(t, err)

	err = WaitLoadTest(clientSet, expectedLoadtestName)
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

	var jmeterNamespace *coreV1.Namespace

	t.Run("Checking namespace is created", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			jmeterNamespace, _ = client.CoreV1().Namespaces().Get(context.Background(), expectedLoadtestName, metaV1.GetOptions{})
			if jmeterNamespace != nil {
				break
			}
		}
		// loadtest namespace name is equal to loadtest name
		require.Equal(t, expectedLoadtestName, jmeterNamespace.Name)
	})

	var cm *coreV1.ConfigMapList

	t.Run("Checking JMeter configmap is created", func(t *testing.T) {
		for i := 0; i < 5; i++ {
			cm, _ = client.CoreV1().ConfigMaps(jmeterNamespace.Name).List(context.Background(), metaV1.ListOptions{LabelSelector: "app=hf-jmeter"})
			if len(cm.Items) != 0 {
				break
			}
		}
		assert.NotEmpty(t, cm.Items)
	})

	t.Run("Checking env vars secret is created and not empty", func(t *testing.T) {
		var secretsCount int
		var secretItem coreV1.Secret
		for i := 0; i < 5; i++ {
			secrets, err := GetSecret(client.CoreV1(), jmeterNamespace.Name)
			require.NoError(t, err, "Could not get namespace secrets")

			if len(secrets.Items) == 1 {
				secretsCount = len(secrets.Items)
				secretItem = secrets.Items[0]
				break
			}
		}
		assert.Equal(t, 1, secretsCount)
		assert.NotEmpty(t, secretItem)

	})

	t.Run("Checking all worker pods are created", func(t *testing.T) {
		var podsCount int
		for i := 0; i < 5; i++ {
			pods, _ := GetDistributedPods(client.CoreV1(), jmeterNamespace.Name)

			if len(pods.Items) == int(distributedPods) {
				podsCount = len(pods.Items)
				break
			}
		}
		assert.Equal(t, distributedPods, int32(podsCount))
	})

	t.Run("Checking master pod is created", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		watchObj, _ := client.CoreV1().Pods(expectedLoadtestName).Watch(context.Background(), metaV1.ListOptions{
			LabelSelector: "app=loadtest-master",
		})

		watchEvent, err := waitfor.ResourceWithContext(ctx, watchObj, (waitfor.Condition{}).PodRunning)
		require.NoError(t, err)

		pod := watchEvent.Object.(*coreV1.Pod)
		assert.Equal(t, coreV1.PodRunning, pod.Status.Phase)
	})

	t.Run("Checking Job is created", func(t *testing.T) {
		var job *batchV1.Job
		for i := 0; i < 5; i++ {
			job, err = client.BatchV1().Jobs(jmeterNamespace.Name).Get(context.Background(), "loadtest-master", metaV1.GetOptions{})
			require.NoError(t, err, "Could not get job")

			if job.Name == "loadtest-master" {
				break
			}
		}
		assert.Equal(t, "loadtest-master", job.Name)
	})

	t.Run("Checking loadtest is in Running state", func(t *testing.T) {
		var phase string
		phase, err = GetLoadTestPhase(clientSet, expectedLoadtestName)
		require.NoError(t, err)
		assert.Equal(t, string(loadTestV1.LoadTestRunning), phase)
	})

	t.Run("Checking loadtest is in Finished state", func(t *testing.T) {
		ctx, cancel := context.WithTimeout(context.Background(), waitForResourceTimeout)
		defer cancel()

		watchObj, _ := clientSet.KangalV1().LoadTests().Watch(ctx, metaV1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", expectedLoadtestName),
		})
		watchEvent, err := waitfor.Resource(watchObj, (waitfor.Condition{}).LoadTestFinished, waitForResourceTimeout)
		require.NoError(t, err)
		loadtest := watchEvent.Object.(*loadTestV1.LoadTest)
		require.Equal(t, loadTestV1.LoadTestFinished, loadtest.Status.Phase)
	})
}

func kubeTestClient() clientSetV.Clientset {
	if len(os.Getenv("KUBECONFIG")) == 0 {
		log.Println("Skipping kube config builder, KUBECONFIG is missed")
		return clientSetV.Clientset{}
	}

	config, err := BuildConfig()
	if err != nil {
		log.Fatal(err)
	}

	clientSet, err := clientSetV.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}
	return *clientSet
}

func kubeClient(t *testing.T) *kubernetes.Clientset {
	t.Helper()

	config, err := BuildConfig()
	require.NoError(t, err)

	cSet, err := kubernetes.NewForConfig(config)
	require.NoError(t, err)

	return cSet
}
