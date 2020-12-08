package proxy

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"google.golang.org/genproto/googleapis/rpc/status"
	"google.golang.org/protobuf/encoding/protojson"

	testHelper "github.com/hellofresh/kangal/pkg/controller"
	grpcProxyV2 "github.com/hellofresh/kangal/pkg/proxy/rpc/pb/grpc/proxy/v2"
)

const (
	restPort = 8090
)

func TestImplLoadTestServiceServer_Create_PostAllFiles(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	rq := grpcProxyV2.CreateRequest{
		DistributedPods: 2,
		Type:            grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_FAKE,
		TargetUrl:       "http://example.com/foo",
		Tags:            map[string]string{"department": "platform", "team": "kangal"},
		TestFile:        readFileContents(t, "testdata/valid/loadtest.jmx", true),
		TestData:        readFileContents(t, "testdata/valid/envvars.csv", true),
		EnvVars:         readFileContents(t, "testdata/valid/testdata.csv", true),
	}

	createdLoadTestName := createLoadtest(t, &rq)

	err := testHelper.WaitLoadTest(clientSet, createdLoadTestName)
	require.NoError(t, err)

	labels, err := testHelper.GetLoadTestLabels(clientSet, createdLoadTestName)
	require.NoError(t, err)

	expected := map[string]string{
		"test-file-hash":      "5a7919885ef46f2e0bd66602944128fde2dce928",
		"test-tag-department": "platform",
		"test-tag-team":       "kangal",
	}
	assert.Equal(t, expected, labels)
}

func TestImplLoadTestServiceServer_Create_Duplicates(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	rq := grpcProxyV2.CreateRequest{
		DistributedPods: 2,
		Type:            grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_FAKE,
		TargetUrl:       "http://example.com/foo",
		TestFile:        readFileContents(t, "testdata/valid/loadtest.jmx", true),
		TestData:        readFileContents(t, "testdata/valid/envvars.csv", true),
		EnvVars:         readFileContents(t, "testdata/valid/testdata.csv", true),
	}

	createLoadtest(t, &rq)

	rqJSON, err := protojson.Marshal(&rq)
	require.NoError(t, err)

	// Creates second loadtest, must fail
	resp, err := http.Post(fmt.Sprintf("http://localhost:%d/v2/load-test", restPort), mimeJSON, bytes.NewReader(rqJSON))
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp.StatusCode)
}

func TestImplLoadTestServiceServer_Create_ReachMaxLimit(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	// this number comes from integration-tests.sh that runs real server with --max-load-tests parameter
	maxLoadTests := 3
	for i := 0; i < maxLoadTests; i++ {
		rq := grpcProxyV2.CreateRequest{
			DistributedPods: 2,
			Type:            grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_FAKE,
			TargetUrl:       "http://example.com/foo",
			TestFile:        encodeContents(t, []byte(fmt.Sprintf("test-%d", i))),
		}

		createdLoadTestName := createLoadtest(t, &rq)

		err := testHelper.WaitLoadTest(clientSet, createdLoadTestName)
		require.NoError(t, err)
	}

	rq2 := grpcProxyV2.CreateRequest{
		DistributedPods: 2,
		Type:            grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_FAKE,
		TargetUrl:       "http://example.com/foo",
		TestFile:        encodeContents(t, []byte(`this should fail`)),
	}

	rqJSON2, err := protojson.Marshal(&rq2)
	require.NoError(t, err)

	resp2, err := http.Post(fmt.Sprintf("http://localhost:%d/v2/load-test", restPort), mimeJSON, bytes.NewReader(rqJSON2))
	require.NoError(t, err)
	require.Equal(t, http.StatusTooManyRequests, resp2.StatusCode)
}

func TestImplLoadTestServiceServer_Create_PostOneFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	rq := grpcProxyV2.CreateRequest{
		DistributedPods: 2,
		Type:            grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_FAKE,
		TargetUrl:       "http://example.com/foo",
		TestFile:        readFileContents(t, "testdata/valid/loadtest2.jmx", true),
	}

	createdLoadTestName := createLoadtest(t, &rq)

	err := testHelper.WaitLoadTest(clientSet, createdLoadTestName)
	require.NoError(t, err)

	data, err := testHelper.GetLoadTestTestdata(clientSet, createdLoadTestName)
	require.NoError(t, err)
	assert.Equal(t, "", data)

	envVars, err := testHelper.GetLoadTestEnvVars(clientSet, createdLoadTestName)
	require.NoError(t, err)
	assert.Equal(t, "", envVars)
}

func TestImplLoadTestServiceServer_Create_EmptyTestFile(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	rq := grpcProxyV2.CreateRequest{
		DistributedPods: 2,
		Type:            grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_FAKE,
		TargetUrl:       "http://example.com/foo",
		TestFile:        readFileContents(t, "testdata/invalid/empty.jmx", true),
	}

	rqJSON, err := protojson.Marshal(&rq)
	require.NoError(t, err)

	resp, err := http.Post(fmt.Sprintf("http://localhost:%d/v2/load-test", restPort), mimeJSON, bytes.NewReader(rqJSON))
	require.NoError(t, err)
	require.Equal(t, http.StatusBadRequest, resp.StatusCode)

	defer func() {
		err := resp.Body.Close()
		assert.NoError(t, err)
	}()

	respBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var rs status.Status
	err = protojson.Unmarshal(respBytes, &rs)
	require.NoError(t, err)

	expectedMessage := `test_file: must not be empty`
	assert.Equal(t, expectedMessage, rs.Message)
}

func TestImplLoadTestServiceServer_Get_Simple(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode")
	}

	rq := grpcProxyV2.CreateRequest{
		DistributedPods: 2,
		Type:            grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_FAKE,
		TargetUrl:       "http://example.com/foo",
		Tags:            map[string]string{"department": "platform", "team": "kangal"},
		TestFile:        readFileContents(t, "testdata/valid/loadtest.jmx", true),
		TestData:        readFileContents(t, "testdata/valid/envvars.csv", true),
	}

	createdLoadTestName := createLoadtest(t, &rq)

	err := testHelper.WaitLoadTest(clientSet, createdLoadTestName)
	require.NoError(t, err)

	req, err := http.NewRequest(http.MethodGet, fmt.Sprintf("http://localhost:%d/v2/load-test/%s", restPort, createdLoadTestName), nil)
	require.NoError(t, err, "Could not create GET request")

	resp, err := http.DefaultClient.Do(req)
	require.NoError(t, err, "Could not send GET request")

	defer func() {
		err := resp.Body.Close()
		assert.NoError(t, err)
	}()

	require.Equal(t, http.StatusOK, resp.StatusCode)

	restBody, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err, "Could not get response body")
	require.NotEmpty(t, restBody)
	t.Logf("gRPC/REST gateway response: %s", restBody)

	dat := new(grpcProxyV2.GetResponse)

	unmarshalErr := protojson.Unmarshal(restBody, dat)
	require.NoError(t, unmarshalErr, "Could not unmarshal response body")
	assert.NotEmpty(t, dat.LoadTestStatus.GetName(), "Could not get namespace from GET request")

	assert.Equal(t, createdLoadTestName, dat.LoadTestStatus.GetName())
	assert.NotEmpty(t, dat.LoadTestStatus.GetPhase())
	assert.NotEqual(t, grpcProxyV2.LoadTestPhase_LOAD_TEST_PHASE_ERRORED.String(), dat.LoadTestStatus.GetPhase())
	assert.Equal(t, true, dat.LoadTestStatus.GetHasTestData())
}

func createLoadtest(t *testing.T, rq *grpcProxyV2.CreateRequest) string {
	t.Helper()

	rqJSON, err := protojson.Marshal(rq)
	require.NoError(t, err)

	resp, err := http.Post(fmt.Sprintf("http://localhost:%d/v2/load-test", restPort), mimeJSON, bytes.NewReader(rqJSON))
	require.NoError(t, err)
	require.Equal(t, http.StatusOK, resp.StatusCode)

	respBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var rs grpcProxyV2.CreateResponse
	err = protojson.Unmarshal(respBytes, &rs)
	require.NoError(t, err)

	createdLoadTestName := rs.GetLoadTestStatus().GetName()

	t.Cleanup(func() {
		err := testHelper.DeleteLoadTest(clientSet, createdLoadTestName, t.Name())
		assert.NoError(t, err)
	})

	return createdLoadTestName
}
