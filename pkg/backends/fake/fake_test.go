package fake

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap"
	batchV1 "k8s.io/api/batch/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	loadtestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

type StatusError struct{}

func (e *StatusError) Error() string {
	return ""
}

func (e *StatusError) Status() metav1.Status {
	return metav1.Status{Reason: metav1.StatusReasonNotFound}
}

func TestCheckOrCreateResources(t *testing.T) {
	lt := &loadtestV1.LoadTest{}
	lt.Status.Namespace = "test-namespace"

	t.Run("namespace not found", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		client.Fake.PrependReactor("get", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			actionImpl := action.(k8stesting.GetActionImpl)
			assert.Equal(t, "test-namespace", actionImpl.Name)
			return true, nil, &StatusError{}
		})

		backend := New(client, lt, zap.NewNop())
		assert.NoError(t, backend.CheckOrCreateResources(context.TODO()))
	})

	t.Run("job exists", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		client.Fake.PrependReactor("get", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespace",
				},
			}, nil
		})
		client.Fake.PrependReactor("get", "jobs", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			actionImpl := action.(k8stesting.GetActionImpl)
			assert.Equal(t, "loadtest-master", actionImpl.Name)
			return true, &batchV1.Job{}, nil
		})

		backend := New(client, lt, zap.NewNop())
		assert.NoError(t, backend.CheckOrCreateResources(context.TODO()))
	})

	t.Run("job doesn't exist, creating", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		client.Fake.PrependReactor("get", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespace",
				},
			}, nil
		})
		client.Fake.PrependReactor("get", "jobs", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, &StatusError{}
		})
		client.Fake.PrependReactor("create", "jobs", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, &batchV1.Job{}, nil
		})

		backend := New(client, lt, zap.NewNop())
		assert.NoError(t, backend.CheckOrCreateResources(context.TODO()))
	})
}

func TestCheckOrUpdateStatus(t *testing.T) {
	lt := &loadtestV1.LoadTest{}
	lt.Status.Namespace = "test-namespace"

	t.Run("namespace and job already exists, load test is starting", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		client.Fake.PrependReactor("get", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			actionImpl := action.(k8stesting.GetActionImpl)
			assert.Equal(t, "test-namespace", actionImpl.Name)
			return true, &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:         "test-namespace",
					GenerateName: "test-namespace",
					Namespace:    "test-namespace",
				},
			}, nil
		})

		client.Fake.PrependReactor("get", "jobs", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			actionImpl := action.(k8stesting.GetActionImpl)
			assert.Equal(t, "loadtest-master", actionImpl.Name)
			return true, &batchV1.Job{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       batchV1.JobSpec{},
				Status:     batchV1.JobStatus{},
			}, nil
		})

		backend := New(client, lt, zap.NewNop())
		assert.NoError(t, backend.CheckOrUpdateStatus(context.TODO()))
		assert.Equal(t, backend.loadTest.Status.Phase, loadtestV1.LoadTestStarting)
	})

	t.Run("namespace and job already exists, load test is running", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		client.Fake.PrependReactor("get", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			actionImpl := action.(k8stesting.GetActionImpl)
			assert.Equal(t, "test-namespace", actionImpl.Name)
			return true, &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:         "test-namespace",
					GenerateName: "test-namespace",
					Namespace:    "test-namespace",
				},
			}, nil
		})

		client.Fake.PrependReactor("get", "jobs", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			actionImpl := action.(k8stesting.GetActionImpl)
			assert.Equal(t, "loadtest-master", actionImpl.Name)
			return true, &batchV1.Job{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       batchV1.JobSpec{},
				Status: batchV1.JobStatus{
					Active: 1,
				},
			}, nil
		})

		backend := New(client, lt, zap.NewNop())
		assert.NoError(t, backend.CheckOrUpdateStatus(context.TODO()))
		assert.Equal(t, backend.loadTest.Status.Phase, loadtestV1.LoadTestRunning)

	})

	t.Run("namespace and job already exists, load test is finished", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		client.Fake.PrependReactor("get", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			actionImpl := action.(k8stesting.GetActionImpl)
			assert.Equal(t, "test-namespace", actionImpl.Name)
			return true, &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name:         "test-namespace",
					GenerateName: "test-namespace",
					Namespace:    "test-namespace",
				},
			}, nil
		})

		client.Fake.PrependReactor("get", "jobs", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			actionImpl := action.(k8stesting.GetActionImpl)
			assert.Equal(t, "loadtest-master", actionImpl.Name)
			return true, &batchV1.Job{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       batchV1.JobSpec{},
				Status: batchV1.JobStatus{
					Succeeded: 1,
				},
			}, nil
		})

		backend := New(client, lt, zap.NewNop())
		assert.NoError(t, backend.CheckOrUpdateStatus(context.TODO()))
		assert.Equal(t, backend.loadTest.Status.Phase, loadtestV1.LoadTestFinished)

	})

	t.Run("namespace doesn't exist - finished status", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		client.Fake.PrependReactor("get", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			actionImpl := action.(k8stesting.GetActionImpl)
			assert.Equal(t, "test-namespace", actionImpl.Name)

			return true, nil, &StatusError{}
		})

		client.Fake.PrependReactor("get", "jobs", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			actionImpl := action.(k8stesting.GetActionImpl)
			assert.Equal(t, "loadtest-master", actionImpl.Name)
			return true, &batchV1.Job{
				TypeMeta:   metav1.TypeMeta{},
				ObjectMeta: metav1.ObjectMeta{},
				Spec:       batchV1.JobSpec{},
				Status:     batchV1.JobStatus{},
			}, nil
		})

		backend := New(client, lt, zap.NewNop())
		assert.NoError(t, backend.CheckOrUpdateStatus(context.TODO()))
		assert.Equal(t, backend.loadTest.Status.Phase, loadtestV1.LoadTestFinished)
	})

	t.Run("loadtest in error state", func(t *testing.T) {
		lt := &loadtestV1.LoadTest{}
		lt.Status.Phase = loadtestV1.LoadTestErrored

		client := fake.NewSimpleClientset()
		client.Fake.PrependReactor("get", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespace",
				},
			}, nil
		})

		backend := New(client, lt, zap.NewNop())
		assert.NoError(t, backend.CheckOrUpdateStatus(context.TODO()))
	})

	t.Run("job doesn't exist", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		client.Fake.PrependReactor("get", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, &v1.Namespace{
				ObjectMeta: metav1.ObjectMeta{
					Name: "test-namespace",
				},
			}, nil
		})

		client.Fake.PrependReactor("get", "jobs", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, &StatusError{}
		})

		backend := New(client, lt, zap.NewNop())
		assert.Error(t, backend.CheckOrUpdateStatus(context.TODO()))
	})
}
