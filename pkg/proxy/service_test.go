package proxy

import (
	"context"
	"encoding/base64"
	"errors"
	"io/ioutil"
	"testing"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	k8sAPIErrors "k8s.io/apimachinery/pkg/api/errors"
	k8sAPIsMetaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8sTesting "k8s.io/client-go/testing"

	"github.com/hellofresh/kangal/pkg/backends"
	kube "github.com/hellofresh/kangal/pkg/kubernetes"
	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	fakeClientset "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned/fake"
	grpcProxyV2 "github.com/hellofresh/kangal/pkg/proxy/rpc/pb/grpc/proxy/v2"
)

func TestImplLoadTestServiceServer_Get(t *testing.T) {
	var pods = int32(1)
	for _, tt := range []struct {
		name     string
		loadTest apisLoadTestV1.LoadTest
		ltErr    error
		out      *grpcProxyV2.GetResponse
		outErr   error
	}{
		{
			"Valid request",
			apisLoadTestV1.LoadTest{
				Spec: apisLoadTestV1.LoadTestSpec{
					Type:            "JMeter",
					DistributedPods: &pods,
					Tags: apisLoadTestV1.LoadTestTags{
						"team": "kangal",
					},
				},
				Status: apisLoadTestV1.LoadTestStatus{
					Phase:     apisLoadTestV1.LoadTestRunning,
					Namespace: "aaa",
				},
			},
			nil,
			&grpcProxyV2.GetResponse{
				LoadTestStatus: &grpcProxyV2.LoadTestStatus{
					Name:            "aaa",
					DistributedPods: 1,
					Phase:           grpcProxyV2.LoadTestPhase_LOAD_TEST_PHASE_RUNNING,
					Tags:            map[string]string{"team": "kangal"},
					HasEnvVars:      false,
					HasTestData:     false,
					Type:            grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_JMETER,
				},
			},
			nil,
		},
		{
			"Error",
			apisLoadTestV1.LoadTest{},
			errors.New("some test error"),
			nil,
			status.Error(codes.Internal, "some test error"),
		},
		{
			"Not found",
			apisLoadTestV1.LoadTest{},
			k8sAPIErrors.NewNotFound(apisLoadTestV1.Resource("loadtest"), "name"),
			nil,
			status.Error(codes.NotFound, `loadtest.kangal.hellofresh.com "name" not found`),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var (
				kubeClientSet     = fake.NewSimpleClientset()
				loadtestClientSet = fakeClientset.NewSimpleClientset()
				logger            = zaptest.NewLogger(t)
			)
			ctx := ctxzap.ToContext(context.Background(), logger)
			loadtestClientSet.Fake.PrependReactor("get", "loadtests", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, &tt.loadTest, tt.ltErr
			})
			c := kube.NewClient(loadtestClientSet.KangalV1().LoadTests(), kubeClientSet, logger)

			b := backends.New(
				backends.WithLogger(logger),
				backends.WithKubeClientSet(kubeClientSet),
				backends.WithKangalClientSet(loadtestClientSet),
			)

			svc := NewLoadTestServiceServer(c, b, 1, 50)

			out, err := svc.Get(ctx, &grpcProxyV2.GetRequest{Name: "aaa"})
			assert.Equal(t, tt.out, out)
			assert.Equal(t, tt.outErr, err)
		})
	}
}

func TestImplLoadTestServiceServer_Create(t *testing.T) {
	for _, tt := range []struct {
		name            string
		distributedPods int32
		tags            map[string]string
		loadTestType    grpcProxyV2.LoadTestType
		testFilePath    string
		testDataPath    string
		envVarsPath     string
		base64Encoded   bool
		createError     error
		out             *grpcProxyV2.CreateResponse
		outErr          error
	}{
		{
			"Valid request, only test file",
			10,
			map[string]string{"team": "kangal"},
			grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_JMETER,
			"testdata/valid/loadtest.jmx",
			"",
			"",
			false,
			nil,
			&grpcProxyV2.CreateResponse{
				LoadTestStatus: &grpcProxyV2.LoadTestStatus{
					Name:            "",
					DistributedPods: 10,
					Phase:           grpcProxyV2.LoadTestPhase_LOAD_TEST_PHASE_CREATING,
					Tags:            map[string]string{"team": "kangal"},
					HasEnvVars:      false,
					HasTestData:     false,
					Type:            grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_JMETER,
				},
			},
			nil,
		},
		{
			"Valid request, all files",
			10,
			map[string]string{},
			grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_FAKE,
			"testdata/valid/loadtest.jmx",
			"testdata/valid/testdata.csv",
			"testdata/valid/envvars.csv",
			false,
			nil,
			&grpcProxyV2.CreateResponse{
				LoadTestStatus: &grpcProxyV2.LoadTestStatus{
					Name:            "",
					DistributedPods: 10,
					Phase:           grpcProxyV2.LoadTestPhase_LOAD_TEST_PHASE_CREATING,
					Tags:            map[string]string{},
					HasEnvVars:      true,
					HasTestData:     true,
					Type:            grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_FAKE,
				},
			},
			nil,
		},
		{
			"Valid request, all files, base64 encoded",
			10,
			map[string]string{},
			grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_FAKE,
			"testdata/valid/loadtest.jmx",
			"testdata/valid/testdata.csv",
			"testdata/valid/envvars.csv",
			true,
			nil,
			&grpcProxyV2.CreateResponse{
				LoadTestStatus: &grpcProxyV2.LoadTestStatus{
					Name:            "",
					DistributedPods: 10,
					Phase:           grpcProxyV2.LoadTestPhase_LOAD_TEST_PHASE_CREATING,
					Tags:            map[string]string{},
					HasEnvVars:      true,
					HasTestData:     true,
					Type:            grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_FAKE,
				},
			},
			nil,
		},
		{
			"Error on creation",
			10,
			map[string]string{},
			grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_FAKE,
			"testdata/valid/loadtest.jmx",
			"",
			"",
			false,
			errors.New("test creation error"),
			nil,
			status.Error(codes.Internal, `could not create load test: test creation error`),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var (
				loadTest          = &apisLoadTestV1.LoadTest{}
				kubeClientSet     = fake.NewSimpleClientset()
				loadtestClientSet = fakeClientset.NewSimpleClientset()
				logger            = zaptest.NewLogger(t)
			)
			ctx := ctxzap.ToContext(context.Background(), logger)
			loadtestClientSet.Fake.PrependReactor("create", "loadtests", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, loadTest, tt.createError
			})
			c := kube.NewClient(loadtestClientSet.KangalV1().LoadTests(), kubeClientSet, logger)

			b := backends.New(
				backends.WithLogger(logger),
				backends.WithKubeClientSet(kubeClientSet),
				backends.WithKangalClientSet(loadtestClientSet),
			)

			rq := grpcProxyV2.CreateRequest{
				DistributedPods: tt.distributedPods,
				Type:            tt.loadTestType,
				TargetUrl:       "http://example.com/foo",
				Tags:            tt.tags,
			}

			if tt.testFilePath != "" {
				rq.TestFile = readFileContents(t, tt.testFilePath, tt.base64Encoded)
			}
			if tt.testDataPath != "" {
				rq.TestData = readFileContents(t, tt.testDataPath, tt.base64Encoded)
			}
			if tt.envVarsPath != "" {
				rq.EnvVars = readFileContents(t, tt.envVarsPath, tt.base64Encoded)
			}

			if tt.base64Encoded {
				ctx = metadata.NewIncomingContext(ctx, metadata.New(map[string]string{mdFromRESTGw: "true"}))
			}

			svc := NewLoadTestServiceServer(c, b, 1, 50)

			out, err := svc.Create(ctx, &rq)
			assert.Equal(t, tt.out, out)
			assert.Equal(t, tt.outErr, err)
		})
	}
}

func TestImplLoadTestServiceServer_List(t *testing.T) {
	distributedPods := int32(2)
	remainCount := int64(42)

	for _, tt := range []struct {
		name   string
		in     *grpcProxyV2.ListRequest
		result *apisLoadTestV1.LoadTestList
		err    error
		out    *grpcProxyV2.ListResponse
		outErr error
	}{
		{
			name:   "error in client",
			in:     &grpcProxyV2.ListRequest{},
			result: &apisLoadTestV1.LoadTestList{},
			err:    errors.New("client error"),
			out:    nil,
			outErr: status.Error(codes.Internal, `could not list load tests: client error`),
		},
		{
			name: "valid phase",
			in: &grpcProxyV2.ListRequest{
				Phase: grpcProxyV2.LoadTestPhase_LOAD_TEST_PHASE_RUNNING,
			},
			result: &apisLoadTestV1.LoadTestList{
				ListMeta: k8sAPIsMetaV1.ListMeta{
					Continue:           "continue",
					RemainingItemCount: &remainCount,
				},
				Items: []apisLoadTestV1.LoadTest{
					{
						Spec: apisLoadTestV1.LoadTestSpec{
							Type:            apisLoadTestV1.LoadTestTypeJMeter,
							DistributedPods: &distributedPods,
							Tags:            apisLoadTestV1.LoadTestTags{},
							TestFile:        "file content\n",
							TestData:        "test data\n",
						},
						Status: apisLoadTestV1.LoadTestStatus{
							Phase:     apisLoadTestV1.LoadTestRunning,
							Namespace: "random",
						},
					},
				},
			},
			err: nil,
			out: &grpcProxyV2.ListResponse{
				PageSize:      0,
				NextPageToken: "continue",
				Remain:        remainCount,
				LoadTestStatuses: []*grpcProxyV2.LoadTestStatus{
					{
						Name:            "random",
						DistributedPods: distributedPods,
						Phase:           grpcProxyV2.LoadTestPhase_LOAD_TEST_PHASE_RUNNING,
						Tags:            map[string]string{},
						HasEnvVars:      false,
						HasTestData:     true,
						Type:            grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_JMETER,
					},
				},
			},
			outErr: nil,
		},
		{
			name: "success",
			in: &grpcProxyV2.ListRequest{
				PageSize: 10,
				Tags:     map[string]string{"department": "platform", "team": "kangal"},
			},
			result: &apisLoadTestV1.LoadTestList{
				ListMeta: k8sAPIsMetaV1.ListMeta{
					Continue:           "continue",
					RemainingItemCount: &remainCount,
				},
				Items: []apisLoadTestV1.LoadTest{
					{
						ObjectMeta: k8sAPIsMetaV1.ObjectMeta{
							Labels: map[string]string{
								"test-tag-department": "platform",
								"test-tag-team":       "kangal",
							},
						},
						Spec: apisLoadTestV1.LoadTestSpec{
							Type:            apisLoadTestV1.LoadTestTypeJMeter,
							Overwrite:       false,
							MasterConfig:    apisLoadTestV1.ImageDetails{},
							WorkerConfig:    apisLoadTestV1.ImageDetails{},
							DistributedPods: &distributedPods,
							Tags:            apisLoadTestV1.LoadTestTags{"department": "platform", "team": "kangal"},
							TestFile:        "file content\n",
							TestData:        "test data\n",
						},
						Status: apisLoadTestV1.LoadTestStatus{
							Phase:     apisLoadTestV1.LoadTestRunning,
							Namespace: "random",
						},
					},
				},
			},
			err: nil,
			out: &grpcProxyV2.ListResponse{
				PageSize:      10,
				NextPageToken: "continue",
				Remain:        remainCount,
				LoadTestStatuses: []*grpcProxyV2.LoadTestStatus{
					{
						Name:            "random",
						DistributedPods: distributedPods,
						Phase:           grpcProxyV2.LoadTestPhase_LOAD_TEST_PHASE_RUNNING,
						Tags:            map[string]string{"department": "platform", "team": "kangal"},
						HasEnvVars:      false,
						HasTestData:     true,
						Type:            grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_JMETER,
					},
				},
			},
			outErr: nil,
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			var (
				kubeClientSet     = fake.NewSimpleClientset()
				loadTestClientSet = fakeClientset.NewSimpleClientset()
				logger            = zaptest.NewLogger(t)
			)
			ctx := ctxzap.ToContext(context.Background(), logger)
			loadTestClientSet.Fake.PrependReactor("list", "loadtests", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, tt.result, tt.err
			})
			c := kube.NewClient(loadTestClientSet.KangalV1().LoadTests(), kubeClientSet, logger)

			b := backends.New(
				backends.WithLogger(logger),
				backends.WithKubeClientSet(kubeClientSet),
				backends.WithKangalClientSet(loadTestClientSet),
			)

			svc := NewLoadTestServiceServer(c, b, 1, 50)

			out, err := svc.List(ctx, tt.in)
			assert.Equal(t, tt.out, out)
			assert.Equal(t, tt.outErr, err)
		})
	}
}

func TestImplLoadTestServiceServer_Delete(t *testing.T) {
	var (
		kubeClientSet     = fake.NewSimpleClientset()
		loadtestClientSet = fakeClientset.NewSimpleClientset()
		logger            = zaptest.NewLogger(t)
	)
	ctx := ctxzap.ToContext(context.Background(), logger)

	loadtestClientSet.Fake.PrependReactor("delete", "loadtests", func(action k8sTesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &apisLoadTestV1.LoadTest{}, nil
	})
	c := kube.NewClient(loadtestClientSet.KangalV1().LoadTests(), kubeClientSet, logger)
	b := backends.New(
		backends.WithLogger(logger),
		backends.WithKubeClientSet(kubeClientSet),
		backends.WithKangalClientSet(loadtestClientSet),
	)

	svc := NewLoadTestServiceServer(c, b, 1, 50)

	deleteResponse, err := svc.Delete(ctx, &grpcProxyV2.DeleteRequest{Name: "loadtest-fake"})

	assert.NoError(t, err)
	assert.Empty(t, deleteResponse)
}

func readFileContents(t *testing.T, path string, base64Encoded bool) []byte {
	t.Helper()

	contents, err := ioutil.ReadFile(path)
	require.NoError(t, err)
	if !base64Encoded {
		return contents
	}

	return encodeContents(t, contents)
}

func encodeContents(t *testing.T, contents []byte) []byte {
	t.Helper()

	encoded := make([]byte, base64.StdEncoding.EncodedLen(len(contents)))
	base64.StdEncoding.Encode(encoded, contents)
	return encoded
}
