package backends

import (
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const (
	// LoadTestLabel label used for test resources
	LoadTestLabel = "loadtest"
	// LoadTestData is the prefix for the names of the testdata files inside the configmap/filesystem
	LoadTestData = LoadTestLabel + "-testdata"
	// LoadTestScript is the name of the testfile script inside the configmap/filesystem
	LoadTestScript = LoadTestLabel + "-script"
)

// Resources contains resources limits/requests
type Resources struct {
	CPULimits      string
	CPURequests    string
	MemoryLimits   string
	MemoryRequests string
}

// BuildResourceRequirements creates ResourceRequirements that allows not all values to be specified
// This is necessary because setting a resource requirement with value 0 produces a different behavior
func BuildResourceRequirements(resources Resources) coreV1.ResourceRequirements {
	limits := make(map[coreV1.ResourceName]resource.Quantity)
	requests := make(map[coreV1.ResourceName]resource.Quantity)

	if quantity, err := resource.ParseQuantity(resources.CPULimits); err == nil {
		limits[coreV1.ResourceCPU] = quantity
	}

	if quantity, err := resource.ParseQuantity(resources.CPURequests); err == nil {
		requests[coreV1.ResourceCPU] = quantity
	}

	if quantity, err := resource.ParseQuantity(resources.MemoryLimits); err == nil {
		limits[coreV1.ResourceMemory] = quantity
	}

	if quantity, err := resource.ParseQuantity(resources.MemoryRequests); err == nil {
		requests[coreV1.ResourceMemory] = quantity
	}

	return coreV1.ResourceRequirements{
		Limits:   limits,
		Requests: requests,
	}
}
