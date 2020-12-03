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

	rqJSON, err := protojson.Marshal(&rq)
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

	err = testHelper.WaitLoadtest(clientSet, createdLoadTestName)
	require.NoError(t, err)

	labels, err := testHelper.GetLoadtestLabels(clientSet, createdLoadTestName)
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

	rqJSON, err := protojson.Marshal(&rq)
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

	// Creates second loadtest, must fail
	resp2, err := http.Post(fmt.Sprintf("http://localhost:%d/v2/load-test", restPort), mimeJSON, bytes.NewReader(rqJSON))
	require.NoError(t, err)
	assert.Equal(t, http.StatusConflict, resp2.StatusCode)
}

func TestImplLoadTestServiceServer_Create_ReachMaxLimit(t *testing.T) {
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

	rq2 := grpcProxyV2.CreateRequest{
		DistributedPods: 2,
		Type:            grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_FAKE,
		TargetUrl:       "http://example.com/foo",
		TestFile:        readFileContents(t, "testdata/valid/loadtest2.jmx", true),
		TestData:        readFileContents(t, "testdata/valid/envvars.csv", true),
		EnvVars:         readFileContents(t, "testdata/valid/testdata.csv", true),
	}

	rqJSON, err := protojson.Marshal(&rq)
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

	err = testHelper.WaitLoadtest(clientSet, createdLoadTestName)
	require.NoError(t, err)

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

	rqJSON, err := protojson.Marshal(&rq)
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

	err = testHelper.WaitLoadtest(clientSet, createdLoadTestName)
	require.NoError(t, err)

	data, err := testHelper.GetLoadtestTestdata(clientSet, createdLoadTestName)
	require.NoError(t, err)
	assert.Equal(t, "", data)

	envVars, err := testHelper.GetLoadtestEnvVars(clientSet, createdLoadTestName)
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

	respBytes, err := ioutil.ReadAll(resp.Body)
	require.NoError(t, err)

	var rs status.Status
	err = protojson.Unmarshal(respBytes, &rs)
	require.NoError(t, err)

	expectedMessage := `test_file: must not be empty`
	assert.Equal(t, expectedMessage, rs.Message)
}
