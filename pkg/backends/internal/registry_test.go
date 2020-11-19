package internal

import (
	"testing"

	"go.uber.org/zap"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"

	"github.com/golang/mock/gomock"
)

func TestNew(t *testing.T) {
	ctrl := gomock.NewController(t)
	defer ctrl.Finish()

	tests := []struct {
		name           string
		podAnnotations map[string]string
		logger         *zap.Logger
	}{
		{
			name: "no-options",
		},
		{
			name: "pod-annotations",
			podAnnotations: map[string]string{
				"label1": "value1",
			},
		},
		{
			name:   "logger",
			logger: zap.NewNop(),
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

			if nil != tt.logger {
				opts = append(opts, WithLogger(tt.logger))
				b.EXPECT().SetLogger(gomock.Eq(tt.logger))
			}

			New(opts...)
		})
	}
}
