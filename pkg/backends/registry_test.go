package backends

import (
	"testing"

	"github.com/golang/mock/gomock"
	"go.uber.org/zap"
	kubeFake "k8s.io/client-go/kubernetes/fake"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	kangalFake "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned/fake"
)

func TestNew(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name            string
		podAnnotations  map[string]string
		nodeSelector    map[string]string
		logger          *zap.Logger
		kangalClientSet *kangalFake.Clientset
		kubeClientSet   *kubeFake.Clientset
	}{
		{
			name: "no-options",
		},
		{
			name: "pod-annotations",
			podAnnotations: map[string]string{
				"annotation1": "value1",
			},
		},
		{
			name: "pod-labels",
			nodeSelector: map[string]string{
				"label1": "value1",
			},
		},
		{
			name:   "logger",
			logger: zap.NewNop(),
		},
		{
			name:            "kangal-client-set",
			kangalClientSet: kangalFake.NewSimpleClientset(),
		},
		{
			name:          "kube-client-set",
			kubeClientSet: kubeFake.NewSimpleClientset(),
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			defaultRegistry = map[loadTestV1.LoadTestType]Backend{}

			b := NewMockBackend(ctrl)

			b.EXPECT().Type().AnyTimes()
			b.EXPECT().GetEnvConfig().Return(&struct{}{})
			b.EXPECT().SetDefaults()

			Register(b)

			opts := make([]Option, 0)

			if len(tt.podAnnotations) > 0 {
				opts = append(opts, WithPodAnnotations(tt.podAnnotations))
				b.EXPECT().SetPodAnnotations(gomock.Eq(tt.podAnnotations))
			}

			if len(tt.nodeSelector) > 0 {
				opts = append(opts, WithNodeSelector(tt.nodeSelector))
				b.EXPECT().SetPodNodeSelector(gomock.Eq(tt.nodeSelector))
			}

			if nil != tt.logger {
				opts = append(opts, WithLogger(tt.logger))
				b.EXPECT().SetLogger(gomock.Eq(tt.logger))
			}

			if nil != tt.kangalClientSet {
				opts = append(opts, WithKangalClientSet(tt.kangalClientSet))
				b.EXPECT().SetKangalClientSet(gomock.Eq(tt.kangalClientSet))
			}

			if nil != tt.kubeClientSet {
				opts = append(opts, WithKubeClientSet(tt.kubeClientSet))
				b.EXPECT().SetKubeClientSet(gomock.Eq(tt.kubeClientSet))
			}

			New(opts...)
		})
	}
}
