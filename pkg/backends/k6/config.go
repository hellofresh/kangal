package k6

// Config specific to k6 backend
type Config struct {
	ImageName      string `envconfig:"K6_IMAGE_NAME" default:"grafana/k6"`
	ImageTag       string `envconfig:"K6_IMAGE_TAG" default:"latest"`
	CPULimits      string `envconfig:"K6_CPU_LIMITS"`
	CPURequests    string `envconfig:"K6_CPU_REQUESTS"`
	MemoryLimits   string `envconfig:"K6_MEMORY_LIMITS"`
	MemoryRequests string `envconfig:"K6_MEMORY_REQUESTS"`
}
