package backends_test

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/hellofresh/kangal/pkg/backends"
)

func TestBuildResourceRequirements(t *testing.T) {
	req := backends.BuildResourceRequirements(backends.Resources{
		CPULimits:      "500m",
		CPURequests:    "250m",
		MemoryLimits:   "128Mi",
		MemoryRequests: "64Mi",
	})
	assert.Equal(t, int(2), len(req.Limits))
	assert.Equal(t, int(2), len(req.Requests))
}
