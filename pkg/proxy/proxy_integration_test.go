package proxy

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/api/errors"
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

	request, err := createRequestBody(requestFiles, distributedPods, string(loadtestType))

	resp, err := http.Post(fmt.Sprintf("http://localhost:%d/load-test", HTTPPort), request.contentType, request.body)
	require.NoError(t, err, "Could not create POST request")
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	createdLoadTestName := parseBody(resp)

	t.Cleanup(func() {
		err := testhelper.DeleteLoadTest(clientSet, createdLoadTestName, t.Name())
		assert.NoError(t, err)
	})

	err = testhelper.WaitLoadtest(clientSet, createdLoadTestName)
	require.NoError(t, err)

	data, err := testhelper.GetLoadtestTestdata(clientSet, createdLoadTestName)
	require.NoError(t, err)
	assert.Equal(t, testdataString, data)

	envVars, err := testhelper.GetLoadtestEnvVars(clientSet, createdLoadTestName)
	require.NoError(t, err)
	assert.Equal(t, envvarsString, envVars)
}

func TestIntegrationCreateLoadtestDuplicates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	distributedPods := "2"
	loadTestType := apisLoadTestV1.LoadTestTypeFake

	requestFiles := map[string]string{
		testFile: "testdata/valid/loadtest.jmx",
		envVars:  "testdata/valid/envvars.csv",
		testData: "testdata/valid/testdata.csv",
	}

	request, err := createRequestBody(requestFiles, distributedPods, string(loadTestType))
	// we call first time and it is success
	resp, err := http.Post(fmt.Sprintf("http://localhost:%d/load-test", HTTPPort), request.contentType, request.body)
	require.NoError(t, err, "Could not create POST request")
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	createdLoadTestName := parseBody(resp)
	t.Cleanup(func() {
		err := testhelper.DeleteLoadTest(clientSet, createdLoadTestName, t.Name())
		assert.NoError(t, err)
	})

	// we call second time and it fails
	request, err = createRequestBody(requestFiles, distributedPods, string(loadTestType))
	require.NoError(t, err)
	resp, err = http.Post(fmt.Sprintf("http://localhost:%d/load-test", HTTPPort), request.contentType, request.body)
	require.NoError(t, err, "Could not create POST request")

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)
}

func TestIntegrationCreateLoadtestReachMaxLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	distributedPods := "2"
	loadTestType := apisLoadTestV1.LoadTestTypeFake

	requestFiles := map[string]string{
		testFile: "testdata/valid/loadtest.jmx",
		envVars:  "testdata/valid/envvars.csv",
		testData: "testdata/valid/testdata.csv",
	}

	request, err := createRequestBody(requestFiles, distributedPods, string(loadTestType))
	// we call first time and it is success
	resp, err := http.Post(fmt.Sprintf("http://localhost:%d/load-test", HTTPPort), request.contentType, request.body)
	require.NoError(t, err, "Could not create POST request")
	require.Equal(t, http.StatusCreated, resp.StatusCode)

	createdLoadTestName := parseBody(resp)
	t.Cleanup(func() {
		err := testhelper.DeleteLoadTest(clientSet, createdLoadTestName, t.Name())
		assert.NoError(t, err)
	})

	// at the same time we call second test and it fails because we run out of tests
	time.Sleep(5 * time.Second)
	requestFiles = map[string]string{
		testFile: "testdata/valid/loadtest2.jmx",
		envVars:  "testdata/valid/envvars.csv",
		testData: "testdata/valid/testdata.csv",
	}
	request, err = createRequestBody(requestFiles, distributedPods, string(loadTestType))
	require.NoError(t, err)
	resp, err = http.Post(fmt.Sprintf("http://localhost:%d/load-test", HTTPPort), request.contentType, request.body)
	require.NoError(t, err, "Could not create POST request")
	require.Equal(t, http.StatusTooManyRequests, resp.StatusCode)
}

func TestIntegrationCreateLoadtestFormPostOneFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	distributedPods := "2"
	loadTestType := apisLoadTestV1.LoadTestTypeFake

	requestFiles := map[string]string{
		testFile: "testdata/valid/loadtest2.jmx",
	}

	request, err := createRequestBody(requestFiles, distributedPods, string(loadTestType))

	resp, err := http.Post(fmt.Sprintf("http://localhost:%d/load-test", HTTPPort), request.contentType, request.body)
	require.NoError(t, err, "Could not create POST request")

	//added sleep to wait for kangal controller to create a CR from post
	time.Sleep(1 * time.Second)

	require.Equal(t, http.StatusCreated, resp.StatusCode)
	createdLoadTestName := parseBody(resp)
	t.Cleanup(func() {
		err := testhelper.DeleteLoadTest(clientSet, createdLoadTestName, t.Name())
		assert.NoError(t, err)
	})

	data, err := testhelper.GetLoadtestTestdata(clientSet, createdLoadTestName)
	require.NoError(t, err)
	assert.Equal(t, "", data)

	envVars, err := testhelper.GetLoadtestEnvVars(clientSet, createdLoadTestName)
	require.NoError(t, err)
	assert.Equal(t, "", envVars)
}

func TestIntegrationCreateLoadtestEmptyTestFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	distributedPods := "2"
	loadTestType := apisLoadTestV1.LoadTestTypeFake

	var dat map[string]interface{}
	expectedMessage := `error getting "testFile" from request: file is empty`

	requestFiles := map[string]string{
		testFile: "testdata/invalid/empty.jmx",
	}

	request, err := createRequestBody(requestFiles, distributedPods, string(loadTestType))
	require.NoError(t, err, "Could not create request")

	resp, error := http.Post(fmt.Sprintf("http://localhost:%d/load-test", HTTPPort), request.contentType, request.body)
	require.NoError(t, error, "Could not create POST request")

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	defer resp.Body.Close()
	respbody, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err, "Could not get response body")
	unmarshalErr := json.Unmarshal(respbody, &dat)
	require.NoError(t, unmarshalErr, "Could not unmarshal response body")

	message := dat["error"]
	assert.Equal(t, expectedMessage, message)
}

func TestIntegrationCreateLoadtestEmptyTestDataFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	distributedPods := "2"
	loadTestType := apisLoadTestV1.LoadTestTypeFake

	var dat map[string]interface{}
	expectedMessage := `error getting "testData" from request: file is empty`

	requestFiles := map[string]string{
		testFile: "testdata/valid/loadtest2.jmx",
		testData: "testdata/invalid/empty.csv",
	}

	request, err := createRequestBody(requestFiles, distributedPods, string(loadTestType))
	require.NoError(t, err, "Could not create request")

	resp, error := http.Post(fmt.Sprintf("http://localhost:%d/load-test", HTTPPort), request.contentType, request.body)
	require.NoError(t, error, "Could not create POST request")

	assert.Equal(t, http.StatusBadRequest, resp.StatusCode)

	defer resp.Body.Close()
	respbody, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err, "Could not get response body")
	unmarshalErr := json.Unmarshal(respbody, &dat)
	require.NoError(t, unmarshalErr, "Could not unmarshal response body")

	message := dat["error"]
	assert.Equal(t, expectedMessage, message)
}

func TestIntegrationDeleteLoadtest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	distributedPods := int32(2)
	loadTestType := apisLoadTestV1.LoadTestTypeFake

	expectedLoadtestName := "loadtest-for-deletetest"
	notFoundMessage := `loadtests.kangal.hellofresh.com "loadtest-for-deletetest" not found`
	testFile := "testdata/valid/loadtest.jmx"

	err := testhelper.CreateLoadtest(clientSet, distributedPods, expectedLoadtestName, testFile, "", "", loadTestType)
	require.NoError(t, err)

	req, err := http.NewRequest("DELETE", fmt.Sprintf("http://localhost:%d/load-test/loadtest-for-deletetest", HTTPPort), nil)
	require.NoError(t, err, "Could not create DELETE request")

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

	res, _ := http.DefaultClient.Do(req)
	assert.Equal(t, http.StatusNoContent, res.StatusCode)
	if _, err := testhelper.GetLoadtest(clientSet, expectedLoadtestName); err != nil {
		assert.Equal(t, notFoundMessage, err.Error())
	}
}

func TestIntegrationGetLoadtest(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	distributedPods := int32(2)
	loadTestType := apisLoadTestV1.LoadTestTypeFake

	expectedLoadtestName := "loadtest-for-gettest"
	testFile := "testdata/valid/loadtest.jmx"
	testData := "testdata/valid/testdata.csv"
	var dat LoadTestStatus

	err := testhelper.CreateLoadtest(clientSet, distributedPods, expectedLoadtestName, testFile, testData, "", loadTestType)
	require.NoError(t, err)

	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/load-test/loadtest-for-gettest", HTTPPort), nil)
	require.NoError(t, err, "Could not create GET request")

	t.Cleanup(func() {
		err := testhelper.DeleteLoadTest(clientSet, expectedLoadtestName, t.Name())
		assert.NoError(t, err)
	})

	for i := 0; i < 5; i++ {
		//added sleep to wait for kangal controller to create a namespace, related to CR
		time.Sleep(1 * time.Second)

		res, err := http.DefaultClient.Do(req)
		require.NoError(t, err, "Could not send GET request")
		require.Equal(t, http.StatusOK, res.StatusCode)

		defer res.Body.Close()
		respbody, err := ioutil.ReadAll(res.Body)
		require.NoError(t, err, "Could not get response body")

		unmarshalErr := json.Unmarshal(respbody, &dat)
		require.NoError(t, unmarshalErr, "Could not unmarshal response body")
	}

	assert.NotEmpty(t, dat.Namespace, "Could not get namespace from GET request")

	currentNamespace, err := testhelper.GetLoadtestNamespace(clientSet, expectedLoadtestName)
	require.NoError(t, err, "Could not get load test information")

	assert.Equal(t, currentNamespace, dat.Namespace)
	assert.Equal(t, distributedPods, dat.DistributedPods)
	assert.NotEmpty(t, dat.Phase)
	assert.NotEqual(t, apisLoadTestV1.LoadTestErrored, dat.Phase)
	assert.Equal(t, true, dat.HasTestData)
}

func TestIntegrationGetLoadtestLogs(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}
	distributedPods := int32(1)
	loadTestType := apisLoadTestV1.LoadTestTypeFake

	expectedLoadtestName := "loadtest-for-getlogs-test"
	expectedPhase := "running"
	testFile := "testdata/valid/loadtest.jmx"

	client := kubeClient(t)

	err := testhelper.CreateLoadtest(clientSet, distributedPods, expectedLoadtestName, testFile, "", "", loadTestType)
	require.NoError(t, err)

	req, err := http.NewRequest("GET", fmt.Sprintf("http://localhost:%d/load-test/loadtest-for-getlogs-test/logs", HTTPPort), nil)
	require.NoError(t, err, "Could not create GET request")

	t.Cleanup(func() {
		err := testhelper.DeleteLoadTest(clientSet, expectedLoadtestName, t.Name())
		assert.NoError(t, err)
	})

	for i := 0; i < 5; i++ {
		//added sleep to wait for kangal controller to create a namespace and start pods
		time.Sleep(4 * time.Second)

		currentPhase, _ := testhelper.GetLoadtestPhase(clientSet, expectedLoadtestName)
		if currentPhase == expectedPhase {
			// sleep to let JMeter process start and generate logs after load test started
			time.Sleep(1 * time.Second)
			master, _ := testhelper.GetMasterPod(client.CoreV1(), expectedLoadtestName)
			if master.Items[0].Status.Phase == "Running" {
				break
			}
		}
	}
	res, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "Could not send GET request")
	require.Equal(t, http.StatusOK, res.StatusCode)

	defer res.Body.Close()
	_, err = ioutil.ReadAll(res.Body)
	require.NoError(t, err, "Could not get response body")
	//require.NotEmpty(t, respbody, "Response body is empty")
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
