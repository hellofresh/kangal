package locust

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	batchV1 "k8s.io/api/batch/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

func TestSync(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Fake clients
	kubeClient := k8sfake.NewSimpleClientset()
	logger := zaptest.NewLogger(t)

	namespace := "test"
	distributedPods := int32(1)
	reportURL := "http://kangal-proxy.local/load-test/loadtest-name/report"

	loadTest := loadTestV1.LoadTest{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "loadtest-name",
		},
		Spec: loadTestV1.LoadTestSpec{
			EnvVars:         "my-secret,my-super-secret\n",
			DistributedPods: &distributedPods,
		},
		Status: loadTestV1.LoadTestStatus{
			Phase:     "running",
			Namespace: namespace,
			JobStatus: batchV1.JobStatus{},
			Pods:      loadTestV1.LoadTestPodsStatus{},
		},
	}

	b := Backend{
		logger:        logger,
		kubeClientSet: kubeClient,
	}

	err := b.Sync(ctx, loadTest, reportURL)
	require.NoError(t, err, "Error when CheckOrCreateResources")

	services, err := kubeClient.CoreV1().Services(namespace).List(ctx, metaV1.ListOptions{})
	require.NoError(t, err, "Error when listing services")
	assert.NotEmpty(t, services.Items, "Expected non-zero services amount after CheckOrCreateResources but found zero")

	configMaps, err := kubeClient.CoreV1().ConfigMaps(namespace).List(ctx, metaV1.ListOptions{})
	require.NoError(t, err, "Error when listing services")
	assert.NotEmpty(t, configMaps.Items, "Expected non-zero configMaps amount after CheckOrCreateResources but found zero")
}

func TestSyncStatus(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Fake clients
	kubeClient := k8sfake.NewSimpleClientset()
	logger := zaptest.NewLogger(t)

	namespace := "test"
	distributedPods := int32(1)

	loadTest := loadTestV1.LoadTest{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "loadtest-name",
		},
		Spec: loadTestV1.LoadTestSpec{
			DistributedPods: &distributedPods,
		},
		Status: loadTestV1.LoadTestStatus{
			Phase:     "running",
			Namespace: namespace,
			JobStatus: batchV1.JobStatus{},
			Pods:      loadTestV1.LoadTestPodsStatus{},
		},
	}

	b := Backend{
		logger:        logger,
		kubeClientSet: kubeClient,
	}

	err := b.SyncStatus(ctx, loadTest, &loadTest.Status)
	require.NoError(t, err, "Error when CheckOrUpdateStatus")
	assert.Equal(t, loadTestV1.LoadTestFinished, loadTest.Status.Phase)
}

func TestTransformLoadTestSpec(t *testing.T) {
	var distributedPods int32 = 3

	type args struct {
		overwrite       bool
		distributedPods int32
		tags            loadTestV1.LoadTestTags
		testFileStr     string
		envVarsStr      string
		targetURL       string
		duration        time.Duration
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
				overwrite:       true,
				distributedPods: 3,
				tags:            loadTestV1.LoadTestTags{"team": "kangal"},
				testFileStr:     "something in the file",
				envVarsStr:      "my-key,my-value",
				targetURL:       "http://my-app.my-domain.com",
			},
			want: loadTestV1.LoadTestSpec{
				Overwrite:       true,
				DistributedPods: &distributedPods,
				Tags:            loadTestV1.LoadTestTags{"team": "kangal"},
				TestFile:        "something in the file",
				EnvVars:         "my-key,my-value",
				TargetURL:       "http://my-app.my-domain.com",
				MasterConfig:    loadTestV1.ImageDetails{Image: defaultImageName, Tag: defaultImageTag},
			},
			wantErr: false,
		},
		{
			name: "Spec invalid - invalid distributed pods",
			args: args{
				distributedPods: 0,
			},
			want:    loadTestV1.LoadTestSpec{},
			wantErr: true,
		},
		{
			name: "Spec invalid - require test file",
			args: args{
				distributedPods: 3,
			},
			want:    loadTestV1.LoadTestSpec{},
			wantErr: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &loadTestV1.LoadTestSpec{
				Overwrite:       tt.args.overwrite,
				DistributedPods: &tt.args.distributedPods,
				Tags:            tt.args.tags,
				TestFile:        tt.args.testFileStr,
				EnvVars:         tt.args.envVarsStr,
				TargetURL:       tt.args.targetURL,
				Duration:        tt.args.duration,
			}

			b := Backend{
				config: &Config{},
			}
			b.SetDefaults()

			err := b.TransformLoadTestSpec(spec)

			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.want.Overwrite, spec.Overwrite)
			assert.Equal(t, tt.want.MasterConfig, spec.MasterConfig)
			assert.Equal(t, tt.want.MasterConfig, spec.WorkerConfig)
			if nil != tt.want.DistributedPods {
				assert.Equal(t, *tt.want.DistributedPods, *spec.DistributedPods)
			}
			assert.Equal(t, tt.want.Tags, spec.Tags)
			assert.Equal(t, tt.want.TestFile, spec.TestFile)
			assert.Equal(t, tt.want.EnvVars, spec.EnvVars)
			assert.Equal(t, tt.want.TargetURL, spec.TargetURL)
		})
	}
}
