package proxy

import (
	"context"
	"errors"
	"testing"

	"github.com/grpc-ecosystem/go-grpc-middleware/logging/zap/ctxzap"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	k8sAPIErrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8sTesting "k8s.io/client-go/testing"

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
					Tags:            []*grpcProxyV2.Tag{{Key: "team", Value: "kangal"}},
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
			errors.New(`rpc error: code = Internal desc = some test error`),
		},
		{
			"Not found",
			apisLoadTestV1.LoadTest{},
			k8sAPIErrors.NewNotFound(apisLoadTestV1.Resource("loadtest"), "name"),
			nil,
			errors.New(`rpc error: code = NotFound desc = loadtest.kangal.hellofresh.com "name" not found`),
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

			svc := NewLoadTestServiceServer(c)

			out, err := svc.Get(ctx, &grpcProxyV2.GetRequest{Name: "aaa"})
			assert.Equal(t, tt.out, out)

			if tt.outErr != nil {
				require.Error(t, err)
				assert.Equal(t, tt.outErr.Error(), err.Error())
			}
		})
	}
}
