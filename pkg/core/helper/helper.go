package helper

import (
	"encoding/csv"
	"io"
	"strings"

	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
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

// ReadEnvs reads data from csv file to save it as a map for creating a secret
func ReadEnvs(envVars string) (map[string]string, error) {
	m := make(map[string]string)
	reader := csv.NewReader(strings.NewReader(envVars))
	for {
		line, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		if len(line) != 2 {
			return nil, ErrInvalidCSVFormat
		}
		m[line[0]] = line[1]
	}
	return m, nil
}
