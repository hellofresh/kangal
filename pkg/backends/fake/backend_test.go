package fake

import (
	"context"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.uber.org/zap/zaptest"
	batchV1 "k8s.io/api/batch/v1"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/client-go/kubernetes/fake"
	k8stesting "k8s.io/client-go/testing"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

type StatusError struct{}

func (e *StatusError) Error() string {
	return ""
}

func (e *StatusError) Status() metaV1.Status {
	return metaV1.Status{Reason: metaV1.StatusReasonNotFound}
}

func TestSync(t *testing.T) {
	lt := loadTestV1.LoadTest{}
	lt.Status.Namespace = "test-namespace"

	t.Run("namespace not found", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		client.Fake.PrependReactor("get", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			actionImpl := action.(k8stesting.GetActionImpl)
			assert.Equal(t, "test-namespace", actionImpl.Name)
			return true, nil, &StatusError{}
		})

		backend := &Backend{
			kubeClient: client,
			logger:     zaptest.NewLogger(t),
		}
		assert.NoError(t, backend.Sync(context.TODO(), lt, ""))
	})

	t.Run("job exists", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		client.Fake.PrependReactor("get", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, &coreV1.Namespace{
				ObjectMeta: metaV1.ObjectMeta{
					Name: "test-namespace",
				},
			}, nil
		})
		client.Fake.PrependReactor("get", "jobs", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			actionImpl := action.(k8stesting.GetActionImpl)
			assert.Equal(t, "loadtest-master", actionImpl.Name)
			return true, &batchV1.Job{}, nil
		})

		backend := &Backend{
			kubeClient: client,
			logger:     zaptest.NewLogger(t),
		}
		assert.NoError(t, backend.Sync(context.TODO(), lt, ""))
	})

	t.Run("job doesn't exist, creating", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		client.Fake.PrependReactor("get", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, &coreV1.Namespace{
				ObjectMeta: metaV1.ObjectMeta{
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

		backend := &Backend{
			kubeClient: client,
			logger:     zaptest.NewLogger(t),
		}
		assert.NoError(t, backend.Sync(context.TODO(), lt, ""))
	})
}

func TestSyncStatus(t *testing.T) {
	lt := loadTestV1.LoadTest{}
	lt.Status.Namespace = "test-namespace"

	t.Run("namespace and job already exists, load test is starting", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		client.Fake.PrependReactor("get", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			actionImpl := action.(k8stesting.GetActionImpl)
			assert.Equal(t, "test-namespace", actionImpl.Name)
			return true, &coreV1.Namespace{
				ObjectMeta: metaV1.ObjectMeta{
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
				TypeMeta:   metaV1.TypeMeta{},
				ObjectMeta: metaV1.ObjectMeta{},
				Spec:       batchV1.JobSpec{},
				Status:     batchV1.JobStatus{},
			}, nil
		})

		backend := &Backend{
			kubeClient: client,
			logger:     zaptest.NewLogger(t),
		}
		assert.NoError(t, backend.SyncStatus(context.TODO(), lt, &lt.Status))
		assert.Equal(t, lt.Status.Phase, loadTestV1.LoadTestStarting)
	})

	t.Run("namespace and job already exists, load test is running", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		client.Fake.PrependReactor("get", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			actionImpl := action.(k8stesting.GetActionImpl)
			assert.Equal(t, "test-namespace", actionImpl.Name)
			return true, &coreV1.Namespace{
				ObjectMeta: metaV1.ObjectMeta{
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
				TypeMeta:   metaV1.TypeMeta{},
				ObjectMeta: metaV1.ObjectMeta{},
				Spec:       batchV1.JobSpec{},
				Status: batchV1.JobStatus{
					Active: 1,
				},
			}, nil
		})

		backend := &Backend{
			kubeClient: client,
			logger:     zaptest.NewLogger(t),
		}
		assert.NoError(t, backend.SyncStatus(context.TODO(), lt, &lt.Status))
		assert.Equal(t, lt.Status.Phase, loadTestV1.LoadTestRunning)
	})

	t.Run("namespace and job already exists, load test is finished", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		client.Fake.PrependReactor("get", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			actionImpl := action.(k8stesting.GetActionImpl)
			assert.Equal(t, "test-namespace", actionImpl.Name)
			return true, &coreV1.Namespace{
				ObjectMeta: metaV1.ObjectMeta{
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
				TypeMeta:   metaV1.TypeMeta{},
				ObjectMeta: metaV1.ObjectMeta{},
				Spec:       batchV1.JobSpec{},
				Status: batchV1.JobStatus{
					Succeeded: 1,
				},
			}, nil
		})

		backend := &Backend{
			kubeClient: client,
			logger:     zaptest.NewLogger(t),
		}
		assert.NoError(t, backend.SyncStatus(context.TODO(), lt, &lt.Status))
		assert.Equal(t, lt.Status.Phase, loadTestV1.LoadTestFinished)
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
				TypeMeta:   metaV1.TypeMeta{},
				ObjectMeta: metaV1.ObjectMeta{},
				Spec:       batchV1.JobSpec{},
				Status:     batchV1.JobStatus{},
			}, nil
		})

		backend := &Backend{
			kubeClient: client,
			logger:     zaptest.NewLogger(t),
		}
		assert.NoError(t, backend.SyncStatus(context.TODO(), lt, &lt.Status))
		assert.Equal(t, lt.Status.Phase, loadTestV1.LoadTestFinished)
	})

	t.Run("loadtest in error state", func(t *testing.T) {
		lt := loadTestV1.LoadTest{}
		lt.Status.Phase = loadTestV1.LoadTestErrored

		client := fake.NewSimpleClientset()
		client.Fake.PrependReactor("get", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, &coreV1.Namespace{
				ObjectMeta: metaV1.ObjectMeta{
					Name: "test-namespace",
				},
			}, nil
		})

		backend := &Backend{
			kubeClient: client,
			logger:     zaptest.NewLogger(t),
		}
		assert.NoError(t, backend.SyncStatus(context.TODO(), lt, &lt.Status))
	})

	t.Run("job doesn't exist", func(t *testing.T) {
		client := fake.NewSimpleClientset()
		client.Fake.PrependReactor("get", "namespaces", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, &coreV1.Namespace{
				ObjectMeta: metaV1.ObjectMeta{
					Name: "test-namespace",
				},
			}, nil
		})

		client.Fake.PrependReactor("get", "jobs", func(action k8stesting.Action) (handled bool, ret runtime.Object, err error) {
			return true, nil, &StatusError{}
		})

		backend := &Backend{
			kubeClient: client,
			logger:     zaptest.NewLogger(t),
		}
		assert.Error(t, backend.SyncStatus(context.TODO(), lt, &lt.Status))
	})
}

func TestTransformLoadTestSpec(t *testing.T) {
	var distributedPods int32 = 1

	spec := loadTestV1.LoadTestSpec{
		Type:            loadTestV1.LoadTestTypeFake,
		Overwrite:       true,
		DistributedPods: &distributedPods,
		Tags:            loadTestV1.LoadTestTags{"team": "kangal"},
	}

	b := Backend{}
	err := b.TransformLoadTestSpec(&spec)
	if nil != err {
		t.Error(err)
		t.FailNow()
	}

	assert.Equal(t, string(spec.MasterConfig), fmt.Sprintf("%s:%s", imageName, imageTag))
}
