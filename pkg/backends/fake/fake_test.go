package fake

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	loadtestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

func createFake() *Fake {
	return &Fake{
		loadTest: &loadtestV1.LoadTest{},
	}
}

func TestSetLoadTestDefaults(t *testing.T) {
	lt := createFake()

	err := lt.SetDefaults()
	require.NoError(t, err)
	assert.Equal(t, loadtestV1.LoadTestCreating, lt.loadTest.Status.Phase)
	assert.Equal(t, sleepImage, lt.loadTest.Spec.MasterConfig.Image)
	assert.Equal(t, imageTag, lt.loadTest.Spec.MasterConfig.Tag)
	assert.Equal(t, imageTag, lt.loadTest.Spec.MasterConfig.Tag)
}

func TestCheckOrCreateResources(t *testing.T) {
	lt := createFake()
	assert.NoError(t, lt.CheckOrCreateResources(context.TODO()))
}
