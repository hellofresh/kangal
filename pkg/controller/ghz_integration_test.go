package controller

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	_ "github.com/hellofresh/kangal/pkg/backends/ghz"
	"github.com/hellofresh/kangal/pkg/core/waitfor"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

const (
	loadtestName = "loadtest-ghz-integration"
	waitTimeout  = 30 * time.Second
)

func TestIntegrationGhz(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	if os.Getenv("SKIP_GHZ_INTEGRATION_TEST") != "" {
		t.Skip("Skipping ghz integration test!")
	}

	t.Log()

	t.Cleanup(func() {
		err := DeleteLoadTest(clientSet, loadtestName, t.Name())
		assert.NoError(t, err)
	})

	// Prepare loadtest spec
	distributedPods := int32(1)
	loadtestType := loadTestV1.LoadTestTypeGhz
	testFile := "testdata/ghz/config.json"
	envVars := map[string]string{"foo": "bar", "foo2": "bar2"}
	testData := ""

	client := kubeClient(t)

	t.Run("Create loadtest", func(t *testing.T) {
		err := CreateLoadTest(clientSet, distributedPods, loadtestName, testFile, testData, envVars, loadtestType)
		require.NoError(t, err)

		err = WaitLoadTest(clientSet, loadtestName)
		require.NoError(t, err)
	})

	t.Run("Check if namespace is created", func(t *testing.T) {
		ns, _ := client.CoreV1().Namespaces().Get(context.Background(), loadtestName, metaV1.GetOptions{})

		// loadtest namespace name is equal to loadtest name
		require.Equal(t, loadtestName, ns.Name)
	})

	t.Run("Check if configmap is created", func(t *testing.T) {
		loadTestFileConfigMapName := "loadtest-testfile"
		listOptions := metaV1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", loadTestFileConfigMapName),
		}
		cm, _ := client.CoreV1().ConfigMaps(loadtestName).List(context.Background(), listOptions)
		assert.NotEmpty(t, cm.Items)
	})

	t.Run("Wait and check if loadtest is finished", func(t *testing.T) {
		watchObj, err := clientSet.KangalV1().LoadTests().Watch(context.Background(), metaV1.ListOptions{
			FieldSelector: fmt.Sprintf("metadata.name=%s", loadtestName),
		})
		require.NoError(t, err)

		_, err = waitfor.Resource(watchObj, (waitfor.Condition{}).LoadTestFinished, waitTimeout)
		require.NoError(t, err)
	})
}
