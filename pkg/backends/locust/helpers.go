package locust

import (
	"encoding/csv"
	"io"
	"strings"

	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

func buildResourceRequirements(cpuLimit, cpuRequest, memLimit, memRequest string) coreV1.ResourceRequirements {
	limits := make(map[coreV1.ResourceName]resource.Quantity)
	requests := make(map[coreV1.ResourceName]resource.Quantity)

	if quantity, err := resource.ParseQuantity(cpuLimit); err == nil {
		limits[coreV1.ResourceCPU] = quantity
	}

	if quantity, err := resource.ParseQuantity(cpuRequest); err == nil {
		requests[coreV1.ResourceCPU] = quantity
	}

	if quantity, err := resource.ParseQuantity(memLimit); err == nil {
		limits[coreV1.ResourceMemory] = quantity
	}

	if quantity, err := resource.ParseQuantity(memRequest); err == nil {
		requests[coreV1.ResourceMemory] = quantity
	}

	return coreV1.ResourceRequirements{
		Limits:   limits,
		Requests: requests,
	}
}

func readEnvs(envVars string) (map[string]string, error) {
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
