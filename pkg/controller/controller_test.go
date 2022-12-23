package controller

import (
	"context"
	"testing"

	"github.com/hellofresh/kangal/pkg/backends"
	"github.com/hellofresh/kangal/pkg/backends/fake"
	"github.com/hellofresh/kangal/pkg/backends/jmeter"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap/zaptest"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"
)

func TestShouldCreateConfigMaps(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kubeClient := k8sfake.NewSimpleClientset()
	logger := zaptest.NewLogger(t)
	c := &Controller{
		kubeClientSet: kubeClient,
		logger:        logger,
	}

	namespace := "test"
	distributedPods := int32(4)

	loadTest := &loadTestV1.LoadTest{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "loadtest-name",
		},
		Spec: loadTestV1.LoadTestSpec{
			DistributedPods: &distributedPods,
			TestFile:        []byte("test"),
			TestData:        []byte("data"),
		},
		Status: loadTestV1.LoadTestStatus{
			Namespace: namespace,
		},
	}

	tfName, tdNames, _ := c.checkOrCreateConfigMaps(ctx, &fake.Backend{}, loadTest)

	assert.Equal(t, 1, len(tdNames))

	cms, err := kubeClient.CoreV1().ConfigMaps(namespace).List(ctx, metaV1.ListOptions{})
	require.NoError(t, err, "Error when listing config maps")
	assert.Equal(t, 2, len(cms.Items))
	assert.Equal(t, tdNames[0], cms.Items[0].Name)
	assert.Equal(t, []byte("data"), cms.Items[0].BinaryData[backends.LoadTestData])
	assert.Equal(t, tfName, cms.Items[1].Name)
	assert.Equal(t, []byte("test"), cms.Items[1].BinaryData[backends.LoadTestScript])
}

func TestShouldSplitCSVTestData(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	kubeClient := k8sfake.NewSimpleClientset()
	logger := zaptest.NewLogger(t)
	c := &Controller{
		kubeClientSet: kubeClient,
		logger:        logger,
	}

	namespace := "test"
	distributedPods := int32(4)

	loadTest := &loadTestV1.LoadTest{
		ObjectMeta: metaV1.ObjectMeta{
			Name: "loadtest-name",
		},
		Spec: loadTestV1.LoadTestSpec{
			DistributedPods: &distributedPods,
			TestFile:        []byte("test"),
			TestData:        []byte("first line\nsecond line\nthird line\nfourth line"),
		},
		Status: loadTestV1.LoadTestStatus{
			Namespace: namespace,
		},
	}

	tfName, tdNames, _ := c.checkOrCreateConfigMaps(ctx, &jmeter.Backend{}, loadTest)

	assert.Equal(t, 4, len(tdNames))

	cms, err := kubeClient.CoreV1().ConfigMaps(namespace).List(ctx, metaV1.ListOptions{})
	require.NoError(t, err, "Error when listing config maps")
	assert.Equal(t, 5, len(cms.Items))

	assert.Equal(t, tdNames[0], cms.Items[0].Name)
	assert.Equal(t, []byte("first line\n"), cms.Items[0].BinaryData[backends.LoadTestData])

	assert.Equal(t, tdNames[2], cms.Items[2].Name)
	assert.Equal(t, []byte("third line\n"), cms.Items[2].BinaryData[backends.LoadTestData])

	assert.Equal(t, tfName, cms.Items[4].Name)
}
