package backends

import (
	"context"

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

// NewLoadTest returns a new LoadTestType
func NewLoadTest(loadTest *loadTestV1.LoadTest, kubeClientSet kubernetes.Interface, kangalClientSet clientSetV.Interface, logger *zap.Logger, namespacesLister coreListersV1.NamespaceLister, reportConfig report.Config, podAnnotations, namespaceAnnotations map[string]string) LoadTestType {
	switch ltType := loadTest.Spec.Type; ltType {
	case loadTestV1.LoadTestTypeJMeter:
		return jmeter.New(kubeClientSet, kangalClientSet, loadTest, logger, namespacesLister, reportConfig, podAnnotations, namespaceAnnotations)
	default:
		return fake.New(kubeClientSet, kangalClientSet, loadTest, logger, namespacesLister, reportConfig, podAnnotations, namespaceAnnotations)
	}
}
