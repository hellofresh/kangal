package proxy

import (
	"context"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"go.uber.org/zap"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	k8sAPIErrors "k8s.io/apimachinery/pkg/api/errors"

	loadtest "github.com/hellofresh/kangal/pkg/controller"
	kube "github.com/hellofresh/kangal/pkg/kubernetes"
	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	grpcProxyV2 "github.com/hellofresh/kangal/pkg/proxy/rpc/pb/grpc/proxy/v2"
)

type implLoadTestServiceServer struct {
	kubeClient *kube.Client
}

// NewLoadTestServiceServer instantiates new LoadTestServiceServer implementation
func NewLoadTestServiceServer(kubeClient *kube.Client) grpcProxyV2.LoadTestServiceServer {
	return &implLoadTestServiceServer{
		kubeClient: kubeClient,
	}
}

// Get returns load test by given name
func (s *implLoadTestServiceServer) Get(ctx context.Context, in *grpcProxyV2.GetRequest) (*grpcProxyV2.GetResponse, error) {
	logger := ctxzap.Extract(ctx)

	ctx, cancel := context.WithTimeout(ctx, loadtest.KubeTimeout)
	defer cancel()

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
func (s *implLoadTestServiceServer) Create(context.Context, *grpcProxyV2.CreateRequest) (*grpcProxyV2.CreateResponse, error) {
	return nil, status.Error(codes.Unimplemented, "Not implemented yet")
}

// List searches and returns load tests by given filters
func (s *implLoadTestServiceServer) List(context.Context, *grpcProxyV2.ListRequest) (*grpcProxyV2.ListResponse, error) {
	return nil, status.Error(codes.Unimplemented, "")
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
