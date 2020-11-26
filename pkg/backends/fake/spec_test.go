package fake

import (
	"testing"

	"github.com/stretchr/testify/assert"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

func TestBuildFakeLoadTestSpec(t *testing.T) {
	var distributedPods int32 = 1

	type args struct {
		tags      loadTestV1.LoadTestTags
		overwrite bool
	}
	tests := []struct {
		name    string
		args    args
		want    loadTestV1.LoadTestSpec
		wantErr bool
	}{
		{
			name: "Spec is valid",
			args: args{
				tags:      loadTestV1.LoadTestTags{"team": "kangal"},
				overwrite: true,
			},
			want: loadTestV1.LoadTestSpec{
				Type:      "Fake",
				Overwrite: true,
				MasterConfig: loadTestV1.ImageDetails{
					Image: imageName,
					Tag:   imageTag,
				},
				WorkerConfig:    loadTestV1.ImageDetails{},
				DistributedPods: &distributedPods,
				Tags:            loadTestV1.LoadTestTags{"team": "kangal"},
				TestFile:        "",
				TestData:        "",
				EnvVars:         "",
			},
			wantErr: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := BuildLoadTestSpec(tt.args.tags, tt.args.overwrite)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
