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
			LoadTestName:    result.Name,
			DistributedPods: *result.Spec.DistributedPods,
			Phase:           s.phaseToGRPC(result.Status.Phase),
			Tags:            s.tagsToGRPC(result.Spec.Tags),
			HasEnvVars:      len(result.Spec.EnvVars) != 0,
			HasTestData:     len(result.Spec.TestData) != 0,
			Type:            s.typeToGRPC(result.Spec.Type),
		},
	}, nil
}

// List searches and returns load tests by given filters
func (s *implLoadTestServiceServer) List(context.Context, *grpcProxyV2.ListRequest) (*grpcProxyV2.ListResponse, error) {
	return new(grpcProxyV2.ListResponse), nil
}

func (s *implLoadTestServiceServer) phaseToGRPC(p apisLoadTestV1.LoadTestPhase) grpcProxyV2.LoadTestPhase {
	switch p {
	case apisLoadTestV1.LoadTestCreating:
		return grpcProxyV2.LoadTestPhase_LOAD_TEST_PHASE_CREATING
	case apisLoadTestV1.LoadTestStarting:
		return grpcProxyV2.LoadTestPhase_LOAD_TEST_PHASE_STARTING
	case apisLoadTestV1.LoadTestRunning:
		return grpcProxyV2.LoadTestPhase_LOAD_TEST_PHASE_RUNNING
	case apisLoadTestV1.LoadTestFinished:
		return grpcProxyV2.LoadTestPhase_LOAD_TEST_PHASE_FINISHED
	case apisLoadTestV1.LoadTestErrored:
		return grpcProxyV2.LoadTestPhase_LOAD_TEST_PHASE_ERRORED
	}

	return grpcProxyV2.LoadTestPhase_LOAD_TEST_PHASE_UNSPECIFIED
}

func (s *implLoadTestServiceServer) typeToGRPC(t apisLoadTestV1.LoadTestType) grpcProxyV2.LoadTestType {
	switch t {
	case apisLoadTestV1.LoadTestTypeJMeter:
		return grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_JMETER
	case apisLoadTestV1.LoadTestTypeFake:
		return grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_FAKE
	case apisLoadTestV1.LoadTestTypeLocust:
		return grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_LOCUST
	}

	return grpcProxyV2.LoadTestType_LOAD_TEST_TYPE_UNSPECIFIED
}

func (s *implLoadTestServiceServer) tagsToGRPC(tt apisLoadTestV1.LoadTestTags) []*grpcProxyV2.Tag {
	tags := make([]*grpcProxyV2.Tag, 0, len(tt))

	for k, v := range tt {
		tags = append(tags, &grpcProxyV2.Tag{
			Key:   k,
			Value: v,
		})
	}

	return tags
}
