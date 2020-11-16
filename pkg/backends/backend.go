package backends

import (
	"context"
	"fmt"
	"time"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	coreListersV1 "k8s.io/client-go/listers/core/v1"

	"github.com/hellofresh/kangal/pkg/backends/fake"
	"github.com/hellofresh/kangal/pkg/backends/jmeter"
	"github.com/hellofresh/kangal/pkg/backends/locust"
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

// LoadTestType defines the methods that a loadtest type needs to implement
// for the controller to be able to run it
type LoadTestType interface {
	// CheckOrCreateResources check for resources or create the needed resources for the loadtest type
	CheckOrCreateResources(ctx context.Context) error

	// CheckOrUpdateLoadTestStatus check current LoadTest progress
	CheckOrUpdateStatus(ctx context.Context) error
}

// Config contains all configurations related to kangal backends
type Config struct {
	JMeter jmeter.Config
	Locust locust.Config
}

// NewLoadTest returns a new LoadTestType
func NewLoadTest(
	loadTest *loadTestV1.LoadTest,
	kubeClientSet kubernetes.Interface,
	kangalClientSet clientSetV.Interface,
	logger *zap.Logger,
	namespacesLister coreListersV1.NamespaceLister,
	reportURL string,
	podAnnotations, namespaceAnnotations map[string]string,
	backendsConfig Config,
) (LoadTestType, error) {
	switch loadTest.Spec.Type {
	case loadTestV1.LoadTestTypeJMeter:
		return jmeter.New(kubeClientSet, kangalClientSet, loadTest, logger, namespacesLister, reportURL, podAnnotations, namespaceAnnotations, backendsConfig.JMeter), nil
	case loadTestV1.LoadTestTypeFake:
		return fake.New(kubeClientSet, loadTest, logger), nil
	case loadTestV1.LoadTestTypeLocust:
		return locust.New(kubeClientSet, kangalClientSet, loadTest, logger, reportURL, backendsConfig.Locust, podAnnotations), nil
	}
	return nil, fmt.Errorf("load test provider not found: %s", loadTest.Spec.Type)
}

// BuildLoadTestSpecByBackend returns a valid LoadTestSpec based on backend rules
func BuildLoadTestSpecByBackend(
	loadTestType loadTestV1.LoadTestType,
	config Config,
	overwrite bool,
	distributedPods int32,
	tags loadTestV1.LoadTestTags,
	testFileStr, testDataStr, envVarsStr, targetURL string,
	duration time.Duration,
) (loadTestV1.LoadTestSpec, error) {
	switch loadTestType {
	case loadTestV1.LoadTestTypeJMeter:
		return jmeter.BuildLoadTestSpec(config.JMeter, overwrite, distributedPods, tags, testFileStr, testDataStr, envVarsStr)
	case loadTestV1.LoadTestTypeFake:
		return fake.BuildLoadTestSpec(tags, overwrite)
	case loadTestV1.LoadTestTypeLocust:
		return locust.BuildLoadTestSpec(config.Locust, overwrite, distributedPods, tags, testFileStr, envVarsStr, targetURL, duration)
	}
	return loadTestV1.LoadTestSpec{}, fmt.Errorf("load test provider not found to build specs: %s", loadTestType)
}
