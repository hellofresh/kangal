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

	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	fakeClientset "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned/fake"
)

func TestCreateLoadTest(t *testing.T) {
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

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

func TestGetLoadTestsByLabel(t *testing.T) {
	for _, tt := range []struct {
		name             string
		expectedResponse string
		expectedStatus   int
		testsList        *apisLoadTestV1.LoadTestList
		error            error
	}{
		{
			"No error",
			"",
			1,
			&apisLoadTestV1.LoadTestList{
				Items: []apisLoadTestV1.LoadTest{
					{
						Status: apisLoadTestV1.LoadTestStatus{
							Phase: apisLoadTestV1.LoadTestRunning,
						},
					},
				},
			},
			nil,
		},
		{
			"Error",
			"",
			1,
			&apisLoadTestV1.LoadTestList{},
			errors.New("test error"),
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			ctx := context.Background()

			var logger = zap.NewNop()
			loadtestClientset := fakeClientset.NewSimpleClientset()
			kubeClientSet := fake.NewSimpleClientset()

			loadTest := &apisLoadTestV1.LoadTest{}
			loadTest.Name = "NameOfMyLoadtest"

			loadtestClientset.Fake.PrependReactor("list", "loadtests", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, tt.testsList, tt.error
			})

			c := NewClient(loadtestClientset.KangalV1().LoadTests(), kubeClientSet, logger)
			_, err := c.GetLoadTestsByLabel(ctx, loadTest)
			assert.Equal(t, tt.error, err)

		})
	}
}

func TestGetLoadTest(t *testing.T) {
	ctx := context.Background()

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

func TestClient_ListLoadTest(t *testing.T) {
	distributedPods := int32(2)
	remainCount := int64(42)

	testCases := []struct {
		scenario       string
		opt            ListOptions
		result         *apisLoadTestV1.LoadTestList
		error          error
		expectedResult *apisLoadTestV1.LoadTestList
		expectedError  string
	}{
		{
			scenario:      "error in client",
			result:        &apisLoadTestV1.LoadTestList{},
			error:         errors.New("client error"),
			expectedError: "client error",
		},
		{
			scenario: "success",
			opt: ListOptions{
				Tags: map[string]string{
					"team": "kangal",
				},
				Limit:    10,
				Continue: "continue",
			},
			result: &apisLoadTestV1.LoadTestList{
				ListMeta: metav1.ListMeta{
					Continue:           "continue",
					RemainingItemCount: &remainCount,
				},
				Items: []apisLoadTestV1.LoadTest{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"test-tag-team": "kangal",
							},
						},
						Spec: apisLoadTestV1.LoadTestSpec{
							Type:            apisLoadTestV1.LoadTestTypeJMeter,
							Overwrite:       false,
							MasterConfig:    apisLoadTestV1.ImageDetails{},
							WorkerConfig:    apisLoadTestV1.ImageDetails{},
							DistributedPods: &distributedPods,
							Tags:            apisLoadTestV1.LoadTestTags{"team": "kangal"},
							TestFile:        "file content\n",
							TestData:        "test data\n",
						},
						Status: apisLoadTestV1.LoadTestStatus{
							Phase:     apisLoadTestV1.LoadTestStarting,
							Namespace: "random",
						},
					},
				},
			},
			expectedResult: &apisLoadTestV1.LoadTestList{
				ListMeta: metav1.ListMeta{
					Continue:           "continue",
					RemainingItemCount: &remainCount,
				},
				Items: []apisLoadTestV1.LoadTest{
					{
						ObjectMeta: metav1.ObjectMeta{
							Labels: map[string]string{
								"test-tag-team": "kangal",
							},
						},
						Spec: apisLoadTestV1.LoadTestSpec{
							Type:            apisLoadTestV1.LoadTestTypeJMeter,
							Overwrite:       false,
							MasterConfig:    apisLoadTestV1.ImageDetails{},
							WorkerConfig:    apisLoadTestV1.ImageDetails{},
							DistributedPods: &distributedPods,
							Tags:            apisLoadTestV1.LoadTestTags{"team": "kangal"},
							TestFile:        "file content\n",
							TestData:        "test data\n",
						},
						Status: apisLoadTestV1.LoadTestStatus{
							Phase:     apisLoadTestV1.LoadTestStarting,
							Namespace: "random",
						},
					},
				},
			},
		},
	}

	for _, tc := range testCases {
		tc := tc
		t.Run(tc.scenario, func(t *testing.T) {
			t.Parallel()

			ctx := context.Background()

			loadTestClientSet := fakeClientset.NewSimpleClientset()
			kubeClientSet := fake.NewSimpleClientset()

			loadTest := &apisLoadTestV1.LoadTest{}
			loadTest.Name = "NameOfMyLoadTest"

			loadTestClientSet.Fake.PrependReactor("list", "loadtests", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, tc.result, tc.error
			})

			c := NewClient(loadTestClientSet.KangalV1().LoadTests(), kubeClientSet, zap.NewNop())
			result, err := c.ListLoadTest(ctx, tc.opt)

			assert.Equal(t, tc.expectedResult, result)

			if tc.expectedError == "" {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, tc.expectedError)
			}
		})
	}
}

func TestClient_filterLoadTestsByPhase(t *testing.T) {
	tests := []struct {
		scenario    string
		ltList      *apisLoadTestV1.LoadTestList
		phase       apisLoadTestV1.LoadTestPhase
		resultCount int
	}{
		{
			scenario: "Filter by valid phase",
			ltList: &apisLoadTestV1.LoadTestList{
				TypeMeta: metav1.TypeMeta{},
				ListMeta: metav1.ListMeta{},
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
				},
			},
			phase:       apisLoadTestV1.LoadTestRunning,
			resultCount: 1,
		},
		{
			scenario:    "Empty input list",
			ltList:      &apisLoadTestV1.LoadTestList{},
			phase:       apisLoadTestV1.LoadTestRunning,
			resultCount: 0,
		},
		{
			scenario: "Empty phase should skip filtering and return all",
			ltList: &apisLoadTestV1.LoadTestList{
				TypeMeta: metav1.TypeMeta{},
				ListMeta: metav1.ListMeta{},
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
				},
			},
			phase:       "",
			resultCount: 2,
		},
	}
	for _, test := range tests {
		t.Run(test.scenario, func(t *testing.T) {
			logger := zap.NewNop()
			loadtestClientset := fakeClientset.NewSimpleClientset()
			kubeClientSet := fake.NewSimpleClientset()

			c := NewClient(loadtestClientset.KangalV1().LoadTests(), kubeClientSet, logger)
			filteredList := c.filterLoadTestsByPhase(test.ltList, test.phase)
			assert.Equal(t, test.resultCount, len(filteredList.Items))
		})
	}
}

func TestCountActiveLoadTests(t *testing.T) {
	ctx := context.Background()

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
	ctx := context.Background()

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
	ctx := context.Background()

	var logger = zap.NewNop()
	loadtestClientset := fakeClientset.NewSimpleClientset()
	client := &fake.Clientset{}
	client.Fake.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
		return true, &corev1.PodList{}, errors.New("return an error")
	})

	c := NewClient(loadtestClientset.KangalV1().LoadTests(), client, logger)
	_, err := c.GetMasterPodRequest(ctx, "namespace")
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
	_, err = c.GetMasterPodRequest(ctx, "namespace")
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

func TestGetWorkerPodsRequest(t *testing.T) {
	tests := []struct {
		scenario      string
		pods          corev1.PodList
		expectedError bool
		error         error
		workerID      string
	}{
		{
			scenario:      "Error on listing worker pods",
			expectedError: true,
			error:         errors.New("list worker pods error"),
		},
		{
			scenario: "Out of range error",
			pods: corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "pod-ed-eed12",
							Namespace: "foo",
							Labels: map[string]string{
								"app": "loadtest-worker-pod",
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "pod-ed-rfh34",
							Namespace: "foo",
							Labels: map[string]string{
								"app": "loadtest-worker-pod",
							},
						},
					},
				},
			},
			expectedError: true,
			error:         errors.New("pod index is out of range"),
			workerID:      "4",
		},
		{
			scenario: "Empty request result",
			pods: corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "pod-ed-eed12",
							Namespace: "foo",
							Labels: map[string]string{
								"app": "loadtest-worker-pod",
							},
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name:      "pod-ed-rfh34",
							Namespace: "foo",
							Labels: map[string]string{
								"app": "loadtest-worker-pod",
							},
						},
					},
				},
			},
			expectedError: false,
			error:         nil,
		},
	}
	for _, test := range tests {
		t.Run(test.scenario, func(t *testing.T) {
			ctx := context.TODO()

			var logger = zap.NewNop()
			loadtestClientset := fakeClientset.NewSimpleClientset()
			client := &fake.Clientset{}

			client.Fake.PrependReactor("list", "pods", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
				return true, &test.pods, test.error
			})
			c := NewClient(loadtestClientset.KangalV1().LoadTests(), client, logger)

			_, err := c.GetWorkerPodRequest(ctx, "foo", test.workerID)
			if !test.expectedError {
				assert.NoError(t, err)
			} else {
				assert.EqualError(t, err, test.error.Error())
			}
		})
	}
}

func TestSortWorkerPods(t *testing.T) {
	tests := []struct {
		Pods   *corev1.PodList
		Result []string
	}{
		{
			Pods: &corev1.PodList{
				Items: []corev1.Pod{
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pod-wd-eed12",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pod-ed-rfh34",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pod-rk-rah34",
						},
					},
					{
						ObjectMeta: metav1.ObjectMeta{
							Name: "pod-ad-rfh34",
						},
					},
				},
			},
			Result: []string{
				"pod-ad-rfh34",
				"pod-ed-rfh34",
				"pod-rk-rah34",
				"pod-wd-eed12",
			},
		},
	}
	for _, test := range tests {

		sortWorkerPods(test.Pods)
		for i := range test.Result {
			assert.Equal(t, test.Result[i], test.Pods.Items[i].Name)
		}
	}
}
