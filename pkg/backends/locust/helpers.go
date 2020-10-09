package locust

import (
	"encoding/csv"
	"fmt"
	"io"
	"os"
	"strings"

	"go.uber.org/zap"
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
