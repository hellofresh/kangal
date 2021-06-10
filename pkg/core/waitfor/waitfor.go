package waitfor

import (
	"context"
	"time"

	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/watch"
	watchtools "k8s.io/client-go/tools/watch"

	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
)

// Condition contains useful functions for watch conditions
type Condition struct {
}

// Added waits until resources exists
func (Condition) Added(event watch.Event) (bool, error) {
	if watch.Added == event.Type {
		return true, nil
	}

	return false, nil
}

// PodRunning waits until Pod are with status phase running
func (Condition) PodRunning(event watch.Event) (bool, error) {
	if coreV1.PodRunning == event.Object.(*coreV1.Pod).Status.Phase {
		return true, nil
	}

	return false, nil
}

// LoadTestRunning waits until Loadtest are with status phase running
func (Condition) LoadTestRunning(event watch.Event) (bool, error) {
	if apisLoadTestV1.LoadTestRunning == event.Object.(*apisLoadTestV1.LoadTest).Status.Phase {
		return true, nil
	}

	return false, nil
}

// LoadTestFinished waits until Loadtest are with status phase finished
func (Condition) LoadTestFinished(event watch.Event) (bool, error) {
	if apisLoadTestV1.LoadTestFinished == event.Object.(*apisLoadTestV1.LoadTest).Status.Phase {
		return true, nil
	}

	return false, nil
}

// Resource waits until a kubernetes resources to match a condition
func Resource(obj watch.Interface, condFunc watchtools.ConditionFunc, timeout time.Duration) (*watch.Event, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	return watchtools.UntilWithoutRetry(ctx, obj, condFunc)
}

// ResourceWithContext is Resource with custom context when the default context is not suitable
func ResourceWithContext(ctx context.Context, obj watch.Interface, condFunc watchtools.ConditionFunc) (*watch.Event, error) {
	return watchtools.UntilWithoutRetry(ctx, obj, condFunc)
}
