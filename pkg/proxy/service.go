package proxy

import (
	"context"
	"encoding/base64"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/thedevsaddam/govalidator"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	k8sAPIErrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/hellofresh/kangal/pkg/backends"
	kube "github.com/hellofresh/kangal/pkg/kubernetes"
	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	grpcProxyV2 "github.com/hellofresh/kangal/pkg/proxy/rpc/pb/grpc/proxy/v2"
)

const (
	mdFromRESTGw = "x-from-rest-gw"
)

var (
	// not sure this is the best way to validate if the string is URL, but we already use this validator in the project
	regexURL = regexp.MustCompile(govalidator.URL)
)

type implLoadTestServiceServer struct {
	kubeClient      *kube.Client
	registry        backends.Registry
	maxLoadTestsRun int
	maxListLimit    int64
}

// NewLoadTestServiceServer instantiates new LoadTestServiceServer implementation
func NewLoadTestServiceServer(kubeClient *kube.Client, registry backends.Registry, maxLoadTestsRun int, maxListLimit int64) grpcProxyV2.LoadTestServiceServer {
	return &implLoadTestServiceServer{
		kubeClient:      kubeClient,
		registry:        registry,
		maxLoadTestsRun: maxLoadTestsRun,
		maxListLimit:    maxListLimit,
	}
}

// Get returns load test by given name
func (s *implLoadTestServiceServer) Get(ctx context.Context, in *grpcProxyV2.GetRequest) (*grpcProxyV2.GetResponse, error) {
	logger := ctxzap.Extract(ctx)
	logger.Debug("Retrieving info for loadtest", zap.String("name", in.GetName()))

	result, err := s.kubeClient.GetLoadTest(ctx, in.GetName())
	if err != nil {
		if k8sAPIErrors.IsNotFound(err) {
			logger.Warn("Could not find load test info", zap.Error(err), zap.String("name", in.GetName()))
			return nil, status.Error(codes.NotFound, err.Error())
		}

		logger.Error("Could not get load test info with error", zap.Error(err), zap.String("name", in.GetName()))
		return nil, status.Error(codes.Internal, err.Error())
	}

	return &grpcProxyV2.GetResponse{
		LoadTestStatus: &grpcProxyV2.LoadTestStatus{
			Name:            result.Status.Namespace,
			DistributedPods: *result.Spec.DistributedPods,
			Phase:           phaseToGRPC(result.Status.Phase),
			Tags:            result.Spec.Tags,
			HasEnvVars:      len(result.Spec.EnvVars) != 0,
			HasTestData:     len(result.Spec.TestData) != 0,
			Type:            typeToGRPC(result.Spec.Type),
		},
	}, nil
}

// Create creates new load test
func (s *implLoadTestServiceServer) Create(ctx context.Context, in *grpcProxyV2.CreateRequest) (*grpcProxyV2.CreateResponse, error) {
	logger := ctxzap.Extract(ctx)

	if err := validateCreateRequest(in); err != nil {
		return nil, status.Error(codes.InvalidArgument, err.Error())
	}

	envVars := in.GetEnvVars()
	testData := in.GetTestData()
	testFile := in.GetTestFile()

	// there are two ways of getting here - either by direct gRPC call or via gRPC REST gateway,
	// when the REST call must encode byte array representing files content with base64
	if md, ok := metadata.FromIncomingContext(ctx); ok && len(md.Get(mdFromRESTGw)) > 0 {
		var err error
		envVars, testData, testFile, err = decodeFileContents(in.GetEnvVars(), in.GetTestData(), in.GetTestFile())
		if err != nil {
			return nil, status.Errorf(codes.InvalidArgument, "could not base64-decode files contents: %s", err.Error())
		}
	}

	ev, err := ReadEnvs(string(envVars))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "%s", err.Error())
	}

	ltSpec := apisLoadTestV1.LoadTestSpec{
		Type:            grpcToTypeMap[in.GetType()],
		Overwrite:       in.GetOverwrite(),
		DistributedPods: &in.DistributedPods,
		Tags:            in.GetTags(),
		TestFile:        string(testFile),
		TestData:        string(testData),
		EnvVars:         ev,
		TargetURL:       in.GetTargetUrl(),
		Duration:        in.GetDuration().AsDuration(),
	}

	// Building LoadTest based on specs
	loadTest, err := apisLoadTestV1.BuildLoadTestObject(ltSpec)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "could not build Load Test object from spec: %s", err.Error())
	}

	// Find the old load test with the same data
	labeledLoadTests, err := s.kubeClient.GetLoadTestsByLabel(ctx, loadTest)
	if err != nil {
		logger.Error("Could not count active load tests with given hash", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not count active load tests with given hash: %s", err.Error())
	}

	if len(labeledLoadTests.Items) > 0 {
		if !loadTest.Spec.Overwrite {
			return nil, status.Error(codes.AlreadyExists, "Load test with given testfile already exists, aborting. Please delete existing load test and try again.")
		}

		// If users wants to overwrite
		for _, item := range labeledLoadTests.Items {

			// Remove the old tests
			err := s.kubeClient.DeleteLoadTest(ctx, item.Name)
			if err != nil {
				logger.Error("Could not delete load test with error", zap.Error(err))
				return nil, status.Errorf(codes.Internal, "could not delete existing load test %q: %s", item.Name, err.Error())
			}
		}
	}

	// check the number of active loadtests currently running on the cluster
	activeLoadTests, err := s.kubeClient.CountActiveLoadTests(ctx)
	if err != nil {
		logger.Error("Could not count active load tests", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not count active load tests: %s", err.Error())
	}

	if activeLoadTests >= s.maxLoadTestsRun {
		logger.Warn("Number of active load tests reached limit", zap.Int("current", activeLoadTests), zap.Int("limit", s.maxLoadTestsRun))
		return nil, status.Error(codes.ResourceExhausted, "number of active load tests reached limit")
	}

	backend, err := s.registry.GetBackend(ltSpec.Type)
	if err != nil {
		logger.Error("Could not get backend", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not get backend: %s", err.Error())
	}

	err = backend.TransformLoadTestSpec(&ltSpec)
	if err != nil {
		logger.Error("Could not transform Load Test spec", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not transform Load Test spec: %s", err.Error())
	}

	// Pushing LoadTest to Kubernetes
	loadTestName, err := s.kubeClient.CreateLoadTest(ctx, loadTest)
	if err != nil {
		logger.Error("Could not create load test", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not create load test: %s", err.Error())
	}

	return &grpcProxyV2.CreateResponse{
		LoadTestStatus: &grpcProxyV2.LoadTestStatus{
			Name:            loadTestName,
			DistributedPods: in.GetDistributedPods(),
			Phase:           grpcProxyV2.LoadTestPhase_LOAD_TEST_PHASE_CREATING,
			Tags:            in.GetTags(),
			HasEnvVars:      len(loadTest.Spec.EnvVars) != 0,
			HasTestData:     loadTest.Spec.TestData != "",
			Type:            in.GetType(),
		},
	}, nil
}

// List searches and returns load tests by given filters
func (s *implLoadTestServiceServer) List(ctx context.Context, in *grpcProxyV2.ListRequest) (*grpcProxyV2.ListResponse, error) {
	logger := ctxzap.Extract(ctx)
	logger.Debug("Retrieving list of load tests", zap.Any("in", in))

	opt := kube.ListOptions{
		Limit:    in.GetPageSize(),
		Continue: in.GetPageToken(),
		Tags:     in.GetTags(),
		Phase:    grpcToPhaseMap[in.GetPhase()],
	}
	if opt.Limit > s.maxListLimit {
		return nil, status.Errorf(codes.InvalidArgument, "limit value is too big, max possible value is %d", s.maxListLimit)
	}
	if opt.Limit == 0 {
		opt.Limit = s.maxListLimit
	}

	loadTests, err := s.kubeClient.ListLoadTest(ctx, opt)
	if err != nil {
		logger.Error("Could not list load tests", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not list load tests: %s", err.Error())
	}

	var remain int64
	if loadTests.GetRemainingItemCount() != nil {
		remain = *loadTests.GetRemainingItemCount()
	}

	out := &grpcProxyV2.ListResponse{
		PageSize:         in.GetPageSize(),
		NextPageToken:    loadTests.GetContinue(),
		Remain:           remain,
		LoadTestStatuses: make([]*grpcProxyV2.LoadTestStatus, len(loadTests.Items)),
	}
	for i, lt := range loadTests.Items {
		out.LoadTestStatuses[i] = &grpcProxyV2.LoadTestStatus{
			Name:            lt.Status.Namespace,
			DistributedPods: *lt.Spec.DistributedPods,
			Phase:           phaseToGRPC(lt.Status.Phase),
			Tags:            lt.Spec.Tags,
			HasEnvVars:      len(lt.Spec.EnvVars) != 0,
			HasTestData:     len(lt.Spec.TestData) != 0,
			Type:            typeToGRPC(lt.Spec.Type),
		}
	}

	return out, nil
}

// Delete deletes a load test
func (s *implLoadTestServiceServer) Delete(ctx context.Context, in *grpcProxyV2.DeleteRequest) (*grpcProxyV2.DeleteResponse, error) {
	logger := ctxzap.Extract(ctx)
	logger.Debug("Deleting loadtest", zap.String("name", in.GetName()))

	err := s.kubeClient.DeleteLoadTest(ctx, in.GetName())

	return &grpcProxyV2.DeleteResponse{}, err
}

func (s *implLoadTestServiceServer) GetLogs(ctx context.Context, in *grpcProxyV2.GetLogsRequest) (*grpcProxyV2.GetLogsResponse, error) {
	loadtestName := in.GetName()

	logger := ctxzap.Extract(ctx)
	logger = logger.With(zap.String("loadtest", loadtestName))

	logger.Info("Retrieving logs for loadtest")

	loadTest, err := s.kubeClient.GetLoadTest(ctx, loadtestName)
	if err != nil {
		logger.Error("could not get loadtest object", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not get loadtest object %q: %s", loadtestName, err.Error())
	}

	namespace := loadTest.Status.Namespace
	if namespace == "" {
		logger.Error("loadtest has no namespace", zap.Error(err))
		return nil, status.Errorf(codes.Unavailable, "loadtest has no namespace %q: %s", loadtestName, err.Error())
	}

	logsRequest, err := s.kubeClient.GetMasterPodRequest(ctx, namespace)
	if err != nil {
		logger.Error("could not create log fetching request", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not create log fetching request for loadtest %q: %s", loadtestName, err.Error())
	}

	logs, err := doRequest(logsRequest)
	if err != nil {
		logger.Error("could not get master pod logs", zap.Error(err))
		return nil, status.Errorf(codes.Internal, "could not get master pod logs from loadtest %q: %s", loadtestName, err.Error())
	}

	return &grpcProxyV2.GetLogsResponse{Logs: string(logs)}, nil
}

func decodeFileContents(envVars, testData, testFile []byte) (envVarsDecoded []byte, testDataDecoded []byte, testFileDecoded []byte, err error) {
	envVarsDecoded = make([]byte, base64.StdEncoding.DecodedLen(len(envVars)))
	if _, err = base64.StdEncoding.Decode(envVarsDecoded, envVars); err != nil {
		return
	}

	testDataDecoded = make([]byte, base64.StdEncoding.DecodedLen(len(testData)))
	if _, err = base64.StdEncoding.Decode(testDataDecoded, testData); err != nil {
		return
	}

	testFileDecoded = make([]byte, base64.StdEncoding.DecodedLen(len(testFile)))
	if _, err = base64.StdEncoding.Decode(testFileDecoded, testFile); err != nil {
		return
	}

	return
}

func validateCreateRequest(in *grpcProxyV2.CreateRequest) error {
	var buf []string

	if len(in.GetTestFile()) == 0 {
		buf = append(buf, fmt.Sprintf("test_file: must not be empty"))
	}

	if in.Type == grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_UNSPECIFIED {
		buf = append(buf, fmt.Sprintf("type: one of the available must be specified"))
	}

	if in.GetDistributedPods() < 1 {
		buf = append(buf, fmt.Sprintf("distributed_pods: must be at least 1"))
	}

	targetURL := in.GetTargetUrl()
	if targetURL == "" || !regexURL.MatchString(targetURL) {
		buf = append(buf, fmt.Sprintf("target_url: must be valid URL"))
	}

	if len(buf) > 0 {
		return errors.New(strings.Join(buf, "; "))
	}

	return nil
}

func phaseToGRPC(p apisLoadTestV1.LoadTestPhase) grpcProxyV2.LoadTestPhase {
	if grpcVal, found := phaseToGRPCMap[p]; found {
		return grpcVal
	}

	return grpcProxyV2.LoadTestPhase_LOAD_TEST_PHASE_UNSPECIFIED
}

func typeToGRPC(t apisLoadTestV1.LoadTestType) grpcProxyV2.LoadTestType {
	if grpcVal, found := typeToGRPCMap[t]; found {
		return grpcVal
	}

	return grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_UNSPECIFIED
}

var (
	phaseToGRPCMap = map[apisLoadTestV1.LoadTestPhase]grpcProxyV2.LoadTestPhase{
		"":                              grpcProxyV2.LoadTestPhase_LOAD_TEST_PHASE_UNSPECIFIED,
		apisLoadTestV1.LoadTestCreating: grpcProxyV2.LoadTestPhase_LOAD_TEST_PHASE_CREATING,
		apisLoadTestV1.LoadTestStarting: grpcProxyV2.LoadTestPhase_LOAD_TEST_PHASE_STARTING,
		apisLoadTestV1.LoadTestRunning:  grpcProxyV2.LoadTestPhase_LOAD_TEST_PHASE_RUNNING,
		apisLoadTestV1.LoadTestFinished: grpcProxyV2.LoadTestPhase_LOAD_TEST_PHASE_FINISHED,
		apisLoadTestV1.LoadTestErrored:  grpcProxyV2.LoadTestPhase_LOAD_TEST_PHASE_ERRORED,
	}
	// get populated in init() by inverting map above
	grpcToPhaseMap = map[grpcProxyV2.LoadTestPhase]apisLoadTestV1.LoadTestPhase{}

	typeToGRPCMap = map[apisLoadTestV1.LoadTestType]grpcProxyV2.LoadTestType{
		"":                                grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_UNSPECIFIED,
		apisLoadTestV1.LoadTestTypeJMeter: grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_JMETER,
		apisLoadTestV1.LoadTestTypeFake:   grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_FAKE,
		apisLoadTestV1.LoadTestTypeLocust: grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_LOCUST,
	}
	// get populated in init() by inverting map above
	grpcToTypeMap = map[grpcProxyV2.LoadTestType]apisLoadTestV1.LoadTestType{}
)

func init() {
	for k, v := range phaseToGRPCMap {
		grpcToPhaseMap[v] = k
	}

	for k, v := range typeToGRPCMap {
		grpcToTypeMap[v] = k
	}
}
