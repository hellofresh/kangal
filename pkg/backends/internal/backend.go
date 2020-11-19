package internal

import (
	"context"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	coreListersV1 "k8s.io/client-go/listers/core/v1"

	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	clientSetV "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned"
)

// Backend is basic methods a backend must implement
type Backend interface {
	// Type must return backend unique type string, called by both Proxy and Controller
	Type() loadTestV1.LoadTestType

	// TransformLoadTestSpec should validate and transform LoadTestSpec, called by Proxy
	TransformLoadTestSpec(spec *loadTestV1.LoadTestSpec) error

	// Sync should create resources if not exists, called by Controller
	Sync(ctx context.Context, loadTest loadTestV1.LoadTest, reportURL string) error
	// Sync should update status with current resource state, called by Controller
	SyncStatus(ctx context.Context, loadTest loadTestV1.LoadTest, loadTestStatus *loadTestV1.LoadTestStatus) error
}

// BackendGetEnvConfig backend must implement this if it has Environment Variables
// This method is called by both commands, Proxy and Controller
type BackendGetEnvConfig interface {
	// GetEnvConfig get from backend a envConfig struct pointer
	GetEnvConfig() interface{}
}

// BackendSetDefaults backend must implement this if wants to set default values
// This method is called by both commands, Proxy and Controller
type BackendSetDefaults interface {
	// SetDefaultValues gives a chance to backend initialize itself
	SetDefaults()
}

// BackendSetLogger interface can be implemented by backend to receive an logger
// This method is called by both commands, Proxy and Controller
type BackendSetLogger interface {
	// SetLogger gives backend a logger instance
	SetLogger(*zap.Logger)
}

// BackendSetPodAnnotations interface can be implemented by backend to receive pod annotations
// This method is called only by command Controller
type BackendSetPodAnnotations interface {
	// SetLogger gives backend a logger instance
	SetPodAnnotations(map[string]string)
}

// BackendSetKubeClientSet interface can be implemented by backend to receive an kubeClientSet
// This method is called only by command Controller
type BackendSetKubeClientSet interface {
	// SetKubeClientSet gives backend a kubeClientSet instance
	SetKubeClientSet(kubernetes.Interface)
}

// BackendSetKangalClientSet interface can be implemented by backend to receive an kangalClientSet
// This method is called only by command Controller
type BackendSetKangalClientSet interface {
	// SetKangalClientSet gives backend a kangalClientSet instance
	SetKangalClientSet(clientSetV.Interface)
}

// BackendSetNamespaceLister interface can be implemented by backend to receive an namespaceLister
// This method is called only by command Controller
type BackendSetNamespaceLister interface {
	// SetNamespaceLister gives backend a namespaceLister instance
	SetNamespaceLister(coreListersV1.NamespaceLister)
}
