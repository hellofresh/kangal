package backends

import (
	"context"
	"fmt"

	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"
	coreListersV1 "k8s.io/client-go/listers/core/v1"

	"github.com/hellofresh/kangal/pkg/backends/fake"
	"github.com/hellofresh/kangal/pkg/backends/jmeter"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	clientSetV "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned"
	"github.com/hellofresh/kangal/pkg/report"
)

// LoadTestType defines the methods that a loadtest type needs to implement
// for the controller to be able to run it
type LoadTestType interface {
	// SetLoadTestDefaults mutates the LoadTest object to add default values to empty fields
	SetDefaults() error

	// CheckOrCreateResources check for resources or create the needed resources for the loadtest type
	CheckOrCreateResources(ctx context.Context) error

	// CheckOrUpdateLoadTestStatus check current LoadTest progress
	CheckOrUpdateStatus(ctx context.Context) error
}

// Config contains all configurations related to kangal backends
type Config struct {
	JMeter jmeter.Config
}

// NewLoadTest returns a new LoadTestType
func NewLoadTest(loadTest *loadTestV1.LoadTest, kubeClientSet kubernetes.Interface, kangalClientSet clientSetV.Interface, logger *zap.Logger, namespacesLister coreListersV1.NamespaceLister, reportConfig report.Config, podAnnotations, namespaceAnnotations map[string]string, backendsConfig Config) (LoadTestType, error) {
	switch loadTest.Spec.Type {
	case loadTestV1.LoadTestTypeJMeter:
		presignedURL := report.NewPreSignedPutURL(loadTest.GetName())
		return jmeter.New(kubeClientSet, kangalClientSet, loadTest, logger, namespacesLister, presignedURL, podAnnotations, namespaceAnnotations, backendsConfig.JMeter), nil
	case loadTestV1.LoadTestTypeFake:
		return fake.New(kubeClientSet, loadTest, logger), nil
	}
	return nil, fmt.Errorf("load test provider not found: %s", loadTest.Spec.Type)
}

// BuildLoadTestSpecByBackend returns a valid LoadTestSpec based on backend rules
func BuildLoadTestSpecByBackend(loadTestType loadTestV1.LoadTestType, overwrite bool, distributedPods int32, testFileStr, testDataStr, envVarsStr string) (loadTestV1.LoadTestSpec, error) {
	switch loadTestType {
	case loadTestV1.LoadTestTypeJMeter:
		return jmeter.BuildLoadTestSpec(overwrite, distributedPods, testFileStr, testDataStr, envVarsStr)
	case loadTestV1.LoadTestTypeFake:
		return fake.BuildLoadTestSpec(overwrite)
	}
	return loadTestV1.LoadTestSpec{}, fmt.Errorf("load test provider not found to build specs: %s", loadTestType)
}
