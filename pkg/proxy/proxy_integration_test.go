package proxy

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"

	testhelper "github.com/hellofresh/kangal/pkg/controller"
	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	clientSetV "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned"
)

var (
	HTTPPort  = 8080
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
	loadtestType := apisLoadTestV1.LoadTestTypeFake
	testdataString := "test data 1\ntest data 2\n"
	envvarsString := "envVar1,value1\nenvVar2,value2\n"

	requestFiles := map[string]string{
		testFile: "testdata/valid/loadtest.jmx",
		envVars:  "testdata/valid/envvars.csv",
		testData: "testdata/valid/testdata.csv",
	}

	var createdLoadTestName string

	t.Run("Creates the loadtest", func(t *testing.T) {
		request, err := createRequestWrapper(requestFiles, distributedPods, string(loadtestType))
		require.NoError(t, err)

		resp, err := http.Post(fmt.Sprintf("http://localhost:%d/load-test", HTTPPort), request.contentType, request.body)
		require.NoError(t, err, "Could not create POST request")
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		createdLoadTestName = parseBody(resp)
	})

	t.Cleanup(func() {
		err := testhelper.DeleteLoadTest(clientSet, createdLoadTestName, t.Name())
		assert.NoError(t, err)
	})

	t.Run("Checking the loadtest is created", func(t *testing.T) {
		err := testhelper.WaitLoadtest(clientSet, createdLoadTestName)
		require.NoError(t, err)
	})

	t.Run("Checking if the loadtest testData is correct", func(t *testing.T) {
		data, err := testhelper.GetLoadtestTestdata(clientSet, createdLoadTestName)
		require.NoError(t, err)
		assert.Equal(t, testdataString, data)
	})

	t.Run("Checking if the loadtest envVars is correct", func(t *testing.T) {
		envVars, err := testhelper.GetLoadtestEnvVars(clientSet, createdLoadTestName)
		require.NoError(t, err)
		assert.Equal(t, envvarsString, envVars)
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
		request, err := createRequestWrapper(requestFiles, distributedPods, string(loadtestType))
		require.NoError(t, err)

		resp, err := http.Post(fmt.Sprintf("http://localhost:%d/load-test", HTTPPort), request.contentType, request.body)
		require.NoError(t, err, "Could not create POST request")
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		createdLoadTestName = parseBody(resp)
	})

	t.Cleanup(func() {
		err := testhelper.DeleteLoadTest(clientSet, createdLoadTestName, t.Name())
		assert.NoError(t, err)
	})

	t.Run("Creates second loadtest, must fail", func(t *testing.T) {
		request, err := createRequestWrapper(requestFiles, distributedPods, string(loadtestType))
		require.NoError(t, err)

		resp, err := http.Post(fmt.Sprintf("http://localhost:%d/load-test", HTTPPort), request.contentType, request.body)
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
		request, err := createRequestWrapper(requestFiles, distributedPods, string(loadtestType))
		require.NoError(t, err)

		resp, err := http.Post(fmt.Sprintf("http://localhost:%d/load-test", HTTPPort), request.contentType, request.body)
		require.NoError(t, err, "Could not create POST request")
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		createdLoadTestName = parseBody(resp)
	})

	t.Cleanup(func() {
		err := testhelper.DeleteLoadTest(clientSet, createdLoadTestName, t.Name())
		assert.NoError(t, err)
	})

	err := testhelper.WaitLoadtest(clientSet, createdLoadTestName)
	require.NoError(t, err)

	t.Run("Creates second loadtest, must fail", func(t *testing.T) {
		request, err := createRequestWrapper(requestFiles, distributedPods, string(loadtestType))
		require.NoError(t, err)

		resp, err := http.Post(fmt.Sprintf("http://localhost:%d/load-test", HTTPPort), request.contentType, request.body)
		require.NoError(t, err, "Could not create POST request")
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
		request, err := createRequestWrapper(requestFiles, distributedPods, string(loadtestType))
		require.NoError(t, err)

		resp, err := http.Post(fmt.Sprintf("http://localhost:%d/load-test", HTTPPort), request.contentType, request.body)
		require.NoError(t, err, "Could not create POST request")
		require.Equal(t, http.StatusCreated, resp.StatusCode)

		createdLoadTestName = parseBody(resp)
	})

	t.Cleanup(func() {
		err := testhelper.DeleteLoadTest(clientSet, createdLoadTestName, t.Name())
		assert.NoError(t, err)
	})

	t.Run("Checking the loadtest is created", func(t *testing.T) {
		err := testhelper.WaitLoadtest(clientSet, createdLoadTestName)
		require.NoError(t, err)
	})

	t.Run("Checking if the loadtest testData is correct", func(t *testing.T) {
		data, err := testhelper.GetLoadtestTestdata(clientSet, createdLoadTestName)
		require.NoError(t, err)
		assert.Equal(t, "", data)
	})

	t.Run("Checking if the loadtest envVars is correct", func(t *testing.T) {
		envVars, err := testhelper.GetLoadtestEnvVars(clientSet, createdLoadTestName)
		require.NoError(t, err)
		assert.Equal(t, "", envVars)
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

	t.Run("Creates the loadtest", func(t *testing.T) {
		request, err := createRequestWrapper(requestFiles, distributedPods, string(loadtestType))
		require.NoError(t, err)

		resp, err := http.Post(fmt.Sprintf("http://localhost:%d/load-test", HTTPPort), request.contentType, request.body)
		require.NoError(t, err, "Could not create POST request")
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body = resp.Body
	})

	defer body.Close()

	t.Run("Check loadtest response", func(t *testing.T) {
		var dat map[string]interface{}

		respbody, err := ioutil.ReadAll(body)
		require.NoError(t, err, "Could not get response body")

		unmarshalErr := json.Unmarshal(respbody, &dat)
		require.NoError(t, unmarshalErr, "Could not unmarshal response body")

		expectedMessage := `error getting "testFile" from request: file is empty`
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
		request, err := createRequestWrapper(requestFiles, distributedPods, string(loadtestType))
		require.NoError(t, err)

		resp, err := http.Post(fmt.Sprintf("http://localhost:%d/load-test", HTTPPort), request.contentType, request.body)
		require.NoError(t, err, "Could not create POST request")
		assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

		body = resp.Body
	})

	defer body.Close()

	t.Run("Check loadtest response", func(t *testing.T) {
		var dat map[string]interface{}

		respbody, err := ioutil.ReadAll(body)
		require.NoError(t, err, "Could not get response body")

		unmarshalErr := json.Unmarshal(respbody, &dat)
		require.NoError(t, unmarshalErr, "Could not unmarshal response body")

		expectedMessage := `error getting "testData" from request: file is empty`
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
		err := testhelper.CreateLoadtest(clientSet, distributedPods, expectedLoadtestName, testFile, "", "", loadtestType)
		require.NoError(t, err)
	})

	t.Cleanup(func() {
		// by default TestDeleteLoadtest will delete a created loadtest so Cleanup has nothing to delete.
		// It means http.StatusNotFound is a good result for Cleanup
		// If Cleanup returns some other error we should assert it
		err := testhelper.DeleteLoadTest(clientSet, expectedLoadtestName, t.Name())
		statusErr, ok := err.(*errors.StatusError)
		if !ok || statusErr.ErrStatus.Code != http.StatusNotFound {
			assert.NoError(t, err)
			return
		}
	})

	t.Run("Deletes the loadtest", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", fmt.Sprintf("http://localhost:%d/load-test/loadtest-for-deletetest", HTTPPort), nil)
		require.NoError(t, err, "Could not create DELETE request")

		res, _ := http.DefaultClient.Do(req)
		assert.Equal(t, http.StatusNoContent, res.StatusCode)

		if _, err := testhelper.GetLoadtest(clientSet, expectedLoadtestName); err != nil {
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
		err := testhelper.CreateLoadtest(clientSet, distributedPods, expectedLoadtestName, testFile, testData, "", loadtestType)
		require.NoError(t, err)
	})

	t.Cleanup(func() {
		err := testhelper.DeleteLoadTest(clientSet, expectedLoadtestName, t.Name())
		assert.NoError(t, err)
	})

	t.Run("Checking the loadtest is created", func(t *testing.T) {
		err := testhelper.WaitLoadtest(clientSet, expectedLoadtestName)
		require.NoError(t, err)
	})

	var respbody []byte

	t.Run("Get loadtest details", func(t *testing.T) {
		req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/load-test/%s", HTTPPort, expectedLoadtestName), nil)
		require.NoError(t, err, "Could not create GET request")

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "Could not send GET request")
		require.Equal(t, http.StatusOK, res.StatusCode)

		defer res.Body.Close()

		respbody, err = ioutil.ReadAll(res.Body)
		require.NoError(t, err, "Could not get response body")
	})

	t.Run("Ensure loadtest GET response is correct", func(t *testing.T) {
		var dat LoadTestStatus

		unmarshalErr := json.Unmarshal(respbody, &dat)
		require.NoError(t, unmarshalErr, "Could not unmarshal response body")
		assert.NotEmpty(t, dat.Namespace, "Could not get namespace from GET request")

		currentNamespace, err := testhelper.GetLoadtestNamespace(clientSet, expectedLoadtestName)
		require.NoError(t, err, "Could not get load test information")

		assert.Equal(t, currentNamespace, dat.Namespace)
		assert.Equal(t, distributedPods, dat.DistributedPods)
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

	testFile := "testdata/valid/loadtest.jmx"

	expectedLoadtestName := "loadtest-for-getlogs-test"

	client := kubeClient(t)

	t.Run("Creates the loadtest", func(t *testing.T) {
		err := testhelper.CreateLoadtest(clientSet, distributedPods, expectedLoadtestName, testFile, "", "", loadtestType)
		require.NoError(t, err)
	})

	t.Cleanup(func() {
		err := testhelper.DeleteLoadTest(clientSet, expectedLoadtestName, t.Name())
		assert.NoError(t, err)
	})

	t.Run("Checking the loadtest is created", func(t *testing.T) {
		err := testhelper.WaitLoadtest(clientSet, expectedLoadtestName)
		require.NoError(t, err)
	})

	t.Run("Checking the loadtest master pod", func(t *testing.T) {
		watchObj, _ := client.CoreV1().Pods(expectedLoadtestName).Watch(context.Background(), metaV1.ListOptions{
			LabelSelector: "app=loadtest-master",
		})

		watchEvent, err := testhelper.WaitResource(watchObj, (testhelper.WaitCondition{}).PodRunning)
		require.NoError(t, err)

		pod := watchEvent.Object.(*coreV1.Pod)
		assert.Equal(t, coreV1.PodRunning, pod.Status.Phase)
	})

	t.Run("Checking the loadtest logs", func(t *testing.T) {
		req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/load-test/%s/logs", HTTPPort, expectedLoadtestName), nil)
		require.NoError(t, err, "Could not create GET request")

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "Could not send GET request")
		require.Equal(t, http.StatusOK, res.StatusCode)

		defer res.Body.Close()

		_, err = ioutil.ReadAll(res.Body)
		require.NoError(t, err, "Could not get response body")
	})
}

func kubeTestClient() clientSetV.Clientset {
	if len(os.Getenv("KUBECONFIG")) == 0 {
		log.Println("Skipping kube config builder, KUBECONFIG is missed")
		return clientSetV.Clientset{}
	}
	config, err := testhelper.BuildConfig()

	clientSet, err := clientSetV.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	return *clientSet
}

func kubeClient(t *testing.T) *kubernetes.Clientset {
	t.Helper()

	config, err := testhelper.BuildConfig()
	require.NoError(t, err)

	cSet, err := kubernetes.NewForConfig(config)
	require.NoError(t, err)

	return cSet
}

func parseBody(res *http.Response) (createdLoadTestName string) {
	var dat LoadTestStatus

	defer res.Body.Close()
	respbody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal("Unable to read the response body:", err)
	}

	unmarshalErr := json.Unmarshal(respbody, &dat)
	if unmarshalErr != nil {
		log.Fatal(fmt.Sprintf("The response body was unable to be unmarshaled: %s", string(respbody)), err)
	}

	return dat.Namespace
}
