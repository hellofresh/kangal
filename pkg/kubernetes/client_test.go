package kubernetes

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/hellofresh/kangal/pkg/controller"
	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	fakeClientset "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned/fake"
)

func TestCreateLoadTest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), controller.KubeTimeout)
	defer cancel()

	var logger = zap.NewNop()
	loadtestClientSet := fakeClientset.NewSimpleClientset()
	kubeClientSet := fake.NewSimpleClientset()

	loadTest := &apisLoadTestV1.LoadTest{}
	loadTest.Name = "NameOfMyLoadtest"

	loadtestClientSet.Fake.PrependReactor("create", "loadtests", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, loadTest, nil
	})

	c := NewClient(loadtestClientSet.KangalV1().LoadTests(), kubeClientSet, logger)

	result, err := c.CreateLoadTest(ctx, loadTest)
	assert.NoError(t, err)
	assert.Equal(t, loadTest.Name, result)

}

func TestCreateLoadTestWithError(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), controller.KubeTimeout)
	defer cancel()

	var logger = zap.NewNop()
	loadtestClientset := fakeClientset.NewSimpleClientset()
	kubeClientSet := fake.NewSimpleClientset()
	loadTest := &apisLoadTestV1.LoadTest{}
	loadTest.Name = ""

	loadtestClientset.Fake.PrependReactor("create", "loadtests", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &apisLoadTestV1.LoadTest{}, errors.New("create returns an error")
	})

	c := NewClient(loadtestClientset.KangalV1().LoadTests(), kubeClientSet, logger)
	result, err := c.CreateLoadTest(ctx, loadTest)
	assert.Error(t, err)
	assert.Equal(t, "", result)
}

func TestDeleteLoadTest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), controller.KubeTimeout)
	defer cancel()

	var logger = zap.NewNop()
	loadtestClientset := fakeClientset.NewSimpleClientset()
	kubeClientSet := fake.NewSimpleClientset()

	ltID := "fake-load-test"

	loadtestClientset.Fake.PrependReactor("delete", "loadtests", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &apisLoadTestV1.LoadTest{}, nil
	})

	c := NewClient(loadtestClientset.KangalV1().LoadTests(), kubeClientSet, logger)
	deleteErr := c.DeleteLoadTest(ctx, ltID)
	assert.NoError(t, deleteErr)
}

func TestCreateLoadTestCRNoLoadTest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), controller.KubeTimeout)
	defer cancel()

	var logger = zap.NewNop()
	loadtestClientset := fakeClientset.NewSimpleClientset()
	kubeClientSet := fake.NewSimpleClientset()

	ltID := "fake-load-test"

	loadtestClientset.Fake.PrependReactor("delete", "loadtests", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &apisLoadTestV1.LoadTest{}, errors.New("delete returns an error: no loadtest with given name found")
	})

	c := NewClient(loadtestClientset.KangalV1().LoadTests(), kubeClientSet, logger)
	deleteErr := c.DeleteLoadTest(ctx, ltID)
	assert.Error(t, deleteErr)
}

func TestGetLoadTest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), controller.KubeTimeout)
	defer cancel()

	var logger = zap.NewNop()
	loadtestClientset := fakeClientset.NewSimpleClientset()
	kubeClientSet := fake.NewSimpleClientset()

	ltID := "fake-load-test"

	loadtestClientset.Fake.PrependReactor("get", "loadtests", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &apisLoadTestV1.LoadTest{}, nil
	})

	c := NewClient(loadtestClientset.KangalV1().LoadTests(), kubeClientSet, logger)
	_, err := c.GetLoadTest(ctx, ltID)
	assert.NoError(t, err)
}

func TestCountActiveLoadTests(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), controller.KubeTimeout)
	defer cancel()

	loadtestClientset := fakeClientset.NewSimpleClientset()
	kubeClientSet := fake.NewSimpleClientset()

	loadtestClientset.Fake.PrependReactor("list", "loadtests", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &apisLoadTestV1.LoadTestList{
			Items: []apisLoadTestV1.LoadTest{
				{
					Status: apisLoadTestV1.LoadTestStatus{
						Phase: apisLoadTestV1.LoadTestRunning,
					},
				},
				{
					Status: apisLoadTestV1.LoadTestStatus{
						Phase: apisLoadTestV1.LoadTestCreating,
					},
				},
				{
					Status: apisLoadTestV1.LoadTestStatus{
						Phase: apisLoadTestV1.LoadTestErrored,
					},
				},
			},
		}, nil
	})

	logger := zap.NewNop()
	c := NewClient(loadtestClientset.KangalV1().LoadTests(), kubeClientSet, logger)
	counter, err := c.CountActiveLoadTests(ctx)
	assert.NoError(t, err)
	assert.Equal(t, 2, counter)
}

func TestGetLoadTestNoLoadTest(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), controller.KubeTimeout)
	defer cancel()

	var logger = zap.NewNop()
	loadtestClientset := fakeClientset.NewSimpleClientset()
	kubeClientSet := fake.NewSimpleClientset()

	ltID := "fake-load-test"

	loadtestClientset.Fake.PrependReactor("get", "loadtests", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &apisLoadTestV1.LoadTest{}, errors.New("get returns an error: no loadtest with given name found")
	})

	c := NewClient(loadtestClientset.KangalV1().LoadTests(), kubeClientSet, logger)
	_, getErr := c.GetLoadTest(ctx, ltID)
	assert.Error(t, getErr)
}

func TestGetMasterPodLogs(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), controller.KubeTimeout)
	defer cancel()

	var logger = zap.NewNop()
	loadtestClientset := fakeClientset.NewSimpleClientset()
	client := &fake.Clientset{}
	client.Fake.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &corev1.PodList{}, errors.New("return an error")
	})

	c := NewClient(loadtestClientset.KangalV1().LoadTests(), client, logger)
	_, err := c.GetMasterPodLogs(ctx, "namespace")
	assert.Error(t, err)

	client = &fake.Clientset{}
	client.Fake.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &corev1.PodList{}, nil
	})

	// this is admittedly not a great test, but since we are using a fake clientset the "GetLog"
	// function always returns an empty Request. Unfortunately, there is no way
	// to easily mock this funciton like there is for "ListPods". To do this We would
	// need to wright our own `FakePod` package, and that doesn't seem worth it.
	c = NewClient(loadtestClientset.KangalV1().LoadTests(), client, logger)
	_, err = c.GetMasterPodLogs(ctx, "namespace")
	assert.Nil(t, err)
}

func TestGetMostRecentPod(t *testing.T) {
	time2019 := metav1.NewTime(time.Date(2019, time.January, 14, 14, 14, 14, 14, time.UTC))
	time2018 := metav1.NewTime(time.Date(2018, time.January, 14, 14, 14, 14, 14, time.UTC))

	tests := []struct {
		Pods   *corev1.PodList
		Result string
	}{
		{
			Pods:   &corev1.PodList{},
			Result: "",
		},
		{
			Pods: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pod-1",
						},
						Status: corev1.PodStatus{
							StartTime: &time2018,
						},
					},
				},
			},
			Result: "pod-1",
		},
		{
			Pods: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pod-1",
						},
						Status: corev1.PodStatus{
							StartTime: &time2018,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pod-2",
						},
						Status: corev1.PodStatus{
							StartTime: &time2019,
						},
					},
				},
			},
			Result: "pod-2",
		},
		{
			Pods: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pod-1",
						},
						Status: corev1.PodStatus{
							StartTime: &time2018,
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pod-2",
						},
						Status: corev1.PodStatus{
							StartTime: nil,
						},
					},
				},
			},
			Result: "pod-1",
		},
	}

	for _, test := range tests {
		podID := getMostRecentPod(test.Pods)
		assert.Equal(t, test.Result, podID)
	}
}
