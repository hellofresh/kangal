package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	k8sAPIErrors "k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	_ "github.com/hellofresh/kangal/pkg/backends/fake"
	testHelper "github.com/hellofresh/kangal/pkg/controller"
	"github.com/hellofresh/kangal/pkg/core/waitfor"
	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	clientSetV "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned"
)

const (
	httpPort = 8080
)

var (
	clientSet clientSetV.Clientset
)

func TestMain(m *testing.M) {
	clientSet = kubeTestClient()
	res := m.Run()
	os.Exit(res)
}

func TestIntegrationCreateLoadtestFormPostAllFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	distributedPods := "2"
	tagsString := "department:platform,team:kangal"
	loadtestType := apisLoadTestV1.LoadTestTypeFake

	requestFiles := map[string]string{
		testFile: "testdata/valid/loadtest.jmx",
		envVars:  "testdata/valid/envvars.csv",
		testData: "testdata/valid/testdata.csv",
	}

	var createdLoadTestName string

	t.Run("Creates the loadtest", func(t *testing.T) {
		request := createRequestWrapper(t, requestFiles, distributedPods, string(loadtestType), tagsString, false, "", "")

		resp, err := http.Post(fmt.Sprintf("http://localhost:%d/load-test", httpPort), request.contentType, request.body)
		require.NoError(t, err, "Could not create POST request")
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		createdLoadTestName = parseBody(t, resp)
	})

	t.Cleanup(func() {
		err := testHelper.DeleteLoadTest(clientSet, createdLoadTestName, t.Name())
		assert.NoError(t, err)
	})

	t.Run("Checking the loadtest is created", func(t *testing.T) {
		err := testHelper.WaitLoadTest(clientSet, createdLoadTestName)
		require.NoError(t, err)
	})

	t.Run("Checking if the loadtest labels are correct", func(t *testing.T) {
		labels, err := testHelper.GetLoadTestLabels(clientSet, createdLoadTestName)
		require.NoError(t, err)

		expected := map[string]string{
			"test-file-hash":      "5a7919885ef46f2e0bd66602944128fde2dce928",
			"test-tag-department": "platform",
			"test-tag-team":       "kangal",
		}
		assert.Equal(t, expected, labels)
	})
}

func TestIntegrationCreateLoadtestDuplicates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	distributedPods := "2"
	loadtestType := apisLoadTestV1.LoadTestTypeFake

	requestFiles := map[string]string{
		testFile: "testdata/valid/loadtest.jmx",
		envVars:  "testdata/valid/envvars.csv",
		testData: "testdata/valid/testdata.csv",
	}

	var createdLoadTestName string

	t.Run("Creates first loadtest, must succeed", func(t *testing.T) {
		request := createRequestWrapper(t, requestFiles, distributedPods, string(loadtestType), "", false, "", "")

		resp, err := http.Post(fmt.Sprintf("http://localhost:%d/load-test", httpPort), request.contentType, request.body)
		require.NoError(t, err, "Could not create POST request")
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		createdLoadTestName = parseBody(t, resp)
	})

	t.Cleanup(func() {
		err := testHelper.DeleteLoadTest(clientSet, createdLoadTestName, t.Name())
		assert.NoError(t, err)
	})

	t.Run("Creates second loadtest, must fail", func(t *testing.T) {
		request := createRequestWrapper(t, requestFiles, distributedPods, string(loadtestType), "", false, "", "")

		resp, err := http.Post(fmt.Sprintf("http://localhost:%d/load-test", httpPort), request.contentType, request.body)
		require.NoError(t, err, "Could not create POST request")
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
	})
}

func TestIntegrationCreateLoadtestReachMaxLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	distributedPods := "2"
	loadtestType := apisLoadTestV1.LoadTestTypeFake

	requestFiles := map[string]string{
		testFile: "testdata/valid/loadtest.jmx",
		envVars:  "testdata/valid/envvars.csv",
		testData: "testdata/valid/testdata.csv",
	}

	requestFilesSecond := map[string]string{
		testFile: "testdata/valid/loadtest2.jmx",
		envVars:  "testdata/valid/envvars.csv",
		testData: "testdata/valid/testdata.csv",
	}

	var createdLoadTestName string

	t.Run("Creates first loadtest, must succeed", func(t *testing.T) {
		request := createRequestWrapper(t, requestFiles, distributedPods, string(loadtestType), "", false, "", "")

		resp, err := http.Post(fmt.Sprintf("http://localhost:%d/load-test", httpPort), request.contentType, request.body)
		require.NoError(t, err, "Could not create POST request")
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		createdLoadTestName = parseBody(t, resp)
	})

	t.Cleanup(func() {
		err := testHelper.DeleteLoadTest(clientSet, createdLoadTestName, t.Name())
		assert.NoError(t, err)
	})

	err := testHelper.WaitLoadTest(clientSet, createdLoadTestName)
	require.NoError(t, err)

	t.Run("Creates second loadtest, must fail", func(t *testing.T) {
		request := createRequestWrapper(t, requestFilesSecond, distributedPods, string(loadtestType), "", false, "", "")

		resp, err := http.Post(fmt.Sprintf("http://localhost:%d/load-test", httpPort), request.contentType, request.body)
		require.NoError(t, err, "Could not create POST request")
		body, _ := io.ReadAll(resp.Body)
		t.Logf(string(body))
		require.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
	})
}

func TestIntegrationCreateLoadtestFormPostOneFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	distributedPods := "2"
	loadtestType := apisLoadTestV1.LoadTestTypeFake

	requestFiles := map[string]string{
		testFile: "testdata/valid/loadtest2.jmx",
	}

	var createdLoadTestName string

	t.Run("Creates the loadtest", func(t *testing.T) {
		request := createRequestWrapper(t, requestFiles, distributedPods, string(loadtestType), "", false, "", "")

		resp, err := http.Post(fmt.Sprintf("http://localhost:%d/load-test", httpPort), request.contentType, request.body)
		require.NoError(t, err, "Could not create POST request")
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		createdLoadTestName = parseBody(t, resp)
	})

	t.Cleanup(func() {
		err := testHelper.DeleteLoadTest(clientSet, createdLoadTestName, t.Name())
		assert.NoError(t, err)
	})

	t.Run("Checking the loadtest is created", func(t *testing.T) {
		err := testHelper.WaitLoadTest(clientSet, createdLoadTestName)
		require.NoError(t, err)
	})

	t.Run("Checking if the loadtest testData is correct", func(t *testing.T) {
		data, err := testHelper.GetLoadTestTestdata(clientSet, createdLoadTestName)
		require.NoError(t, err)
		var empty []byte
		assert.Equal(t, empty, data)
	})

	t.Run("Checking if the loadtest envVars is correct", func(t *testing.T) {
		envVars, err := testHelper.GetLoadTestEnvVars(clientSet, createdLoadTestName)
		require.NoError(t, err)
		assert.Equal(t, map[string]string(nil), envVars)
	})
}

func TestIntegrationCreateLoadtestEmptyTestFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	distributedPods := "2"
	loadtestType := apisLoadTestV1.LoadTestTypeFake

	requestFiles := map[string]string{
		testFile: "testdata/invalid/empty.jmx",
	}

	var body io.ReadCloser

	t.Run("Creates the loadtest with empty testFile", func(t *testing.T) {
		request := createRequestWrapper(t, requestFiles, distributedPods, string(loadtestType), "", false, "", "")

		resp, err := http.Post(fmt.Sprintf("http://localhost:%d/load-test", httpPort), request.contentType, request.body)
		require.NoError(t, err, "Could not create POST request")
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body = resp.Body
	})

	defer func() {
		err := body.Close()
		assert.NoError(t, err)
	}()

	t.Run("Expect loadtest bad request response", func(t *testing.T) {
		var dat map[string]interface{}

		respBody, err := io.ReadAll(body)
		require.NoError(t, err, "Could not get response body")

		unmarshalErr := json.Unmarshal(respBody, &dat)
		require.NoError(t, unmarshalErr, "Could not unmarshal response body")

		expectedMessage := `error getting testFile from request: file is empty`
		gotMessage := dat["error"]

		assert.Equal(t, expectedMessage, gotMessage)
	})
}

func TestIntegrationCreateLoadtestEmptyTestDataFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	distributedPods := "2"
	loadtestType := apisLoadTestV1.LoadTestTypeFake

	requestFiles := map[string]string{
		testFile: "testdata/valid/loadtest2.jmx",
		testData: "testdata/invalid/empty.csv",
	}

	var body io.ReadCloser

	t.Run("Creates the loadtest", func(t *testing.T) {
		request := createRequestWrapper(t, requestFiles, distributedPods, string(loadtestType), "", false, "", "")

		resp, err := http.Post(fmt.Sprintf("http://localhost:%d/load-test", httpPort), request.contentType, request.body)
		require.NoError(t, err, "Could not create POST request")
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body = resp.Body
	})

	defer func() {
		err := body.Close()
		assert.NoError(t, err)
	}()

	t.Run("Check loadtest response", func(t *testing.T) {
		var dat map[string]interface{}

		respbody, err := io.ReadAll(body)
		require.NoError(t, err, "Could not get response body")

		unmarshalErr := json.Unmarshal(respbody, &dat)
		require.NoError(t, unmarshalErr, "Could not unmarshal response body")

		expectedMessage := `error getting testData from request: file is empty`
		gotMessage := dat["error"]

		assert.Equal(t, expectedMessage, gotMessage)
	})
}

func TestIntegrationDeleteLoadtest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	distributedPods := int32(2)
	loadtestType := apisLoadTestV1.LoadTestTypeFake

	testFile := "testdata/valid/loadtest.jmx"

	expectedLoadtestName := "loadtest-for-deletetest"

	t.Run("Creates the loadtest", func(t *testing.T) {
		err := testHelper.CreateLoadTest(clientSet, distributedPods, expectedLoadtestName, testFile, "", nil, loadtestType)
		require.NoError(t, err)
	})

	t.Cleanup(func() {
		// by default TestDeleteLoadtest will delete a created loadtest so Cleanup has nothing to delete.
		// It means http.StatusNotFound is a good result for Cleanup
		// If Cleanup returns some other error we should assert it
		err := testHelper.DeleteLoadTest(clientSet, expectedLoadtestName, t.Name())
		statusErr, ok := err.(*k8sAPIErrors.StatusError)
		if !ok || statusErr.ErrStatus.Code != http.StatusNotFound {
			assert.NoError(t, err)
			return
		}
	})

	t.Run("Deletes the loadtest", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodDelete, fmt.Sprintf("http://localhost:%d/load-test/loadtest-for-deletetest", httpPort), nil)
		require.NoError(t, err, "Could not create DELETE request")

		res, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusNoContent, res.StatusCode)

		if _, err := testHelper.GetLoadTest(clientSet, expectedLoadtestName); err != nil {
			notFoundMessage := `loadtests.kangal.hellofresh.com "loadtest-for-deletetest" not found`
			assert.Equal(t, notFoundMessage, err.Error())
		}
	})
}

func TestIntegrationGetLoadtest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	distributedPods := int32(2)
	loadtestType := apisLoadTestV1.LoadTestTypeFake

	testFile := "testdata/valid/loadtest.jmx"
	testData := "testdata/valid/testdata.csv"

	expectedLoadtestName := "loadtest-for-gettest"

	t.Run("Creates the loadtest", func(t *testing.T) {
		err := testHelper.CreateLoadTest(clientSet, distributedPods, expectedLoadtestName, testFile, testData, nil, loadtestType)
		require.NoError(t, err)
	})

	t.Cleanup(func() {
		err := testHelper.DeleteLoadTest(clientSet, expectedLoadtestName, t.Name())
		assert.NoError(t, err)
	})

	t.Run("Checking the loadtest is created", func(t *testing.T) {
		err := testHelper.WaitLoadTest(clientSet, expectedLoadtestName)
		require.NoError(t, err)
	})

	var httpBody []byte

	t.Run("Get loadtest details", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%d/load-test/%s", httpPort, expectedLoadtestName), nil)
		require.NoError(t, err, "Could not create GET request")

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "Could not send GET request")

		defer func() {
			err := res.Body.Close()
			assert.NoError(t, err)
		}()

		require.Equal(t, http.StatusOK, res.StatusCode)

		httpBody, err = io.ReadAll(res.Body)
		require.NoError(t, err, "Could not get response body")
	})

	t.Run("Ensure loadtest GET response is correct", func(t *testing.T) {
		var dat LoadTestStatus

		unmarshalErr := json.Unmarshal(httpBody, &dat)
		require.NoError(t, unmarshalErr, "Could not unmarshal response body")
		assert.NotEmpty(t, dat.Namespace, "Could not get namespace from GET request")

		currentNamespace, err := testHelper.GetLoadTestNamespace(clientSet, expectedLoadtestName)
		require.NoError(t, err, "Could not get load test information")

		assert.Equal(t, currentNamespace, dat.Namespace)
		assert.NotEmpty(t, dat.Phase)
		assert.NotEqual(t, apisLoadTestV1.LoadTestErrored, dat.Phase)
		assert.Equal(t, true, dat.HasTestData)
	})
}

func TestIntegrationGetLoadtestLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	distributedPods := int32(1)
	loadtestType := apisLoadTestV1.LoadTestTypeFake
	waitForResourceTimeout := 10 * time.Second

	testFile := "testdata/valid/loadtest.jmx"

	expectedLoadtestName := "loadtest-for-getlogs-test"

	client := kubeClient(t)

	t.Run("Creates the loadtest", func(t *testing.T) {
		err := testHelper.CreateLoadTest(clientSet, distributedPods, expectedLoadtestName, testFile, "", nil, loadtestType)
		require.NoError(t, err)
	})

	t.Cleanup(func() {
		err := testHelper.DeleteLoadTest(clientSet, expectedLoadtestName, t.Name())
		assert.NoError(t, err)
	})

	t.Run("Checking the loadtest is created", func(t *testing.T) {
		err := testHelper.WaitLoadTest(clientSet, expectedLoadtestName)
		require.NoError(t, err)
	})

	t.Run("Checking the loadtest master pod", func(t *testing.T) {
		watchObj, _ := client.CoreV1().Pods(expectedLoadtestName).Watch(context.Background(), metaV1.ListOptions{
			LabelSelector: "app=loadtest-master",
		})

		watchEvent, err := waitfor.Resource(watchObj, (waitfor.Condition{}).PodRunning, waitForResourceTimeout)
		require.NoError(t, err)

		pod := watchEvent.Object.(*coreV1.Pod)
		assert.Equal(t, coreV1.PodRunning, pod.Status.Phase)
	})

	t.Run("Checking the loadtest logs", func(t *testing.T) {
		req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%d/load-test/%s/logs", httpPort, expectedLoadtestName), nil)
		require.NoError(t, err, "Could not create GET request")

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "Could not send GET request")
		require.Equal(t, http.StatusOK, res.StatusCode)

		defer func() {
			err := res.Body.Close()
			assert.NoError(t, err)
		}()

		_, err = io.ReadAll(res.Body)
		require.NoError(t, err, "Could not get response body")
	})
}
