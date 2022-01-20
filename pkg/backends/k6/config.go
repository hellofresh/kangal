package k6

// Config specific to k6 backend
type Config struct {
	Image          string `envconfig:"K6_IMAGE"`
	ImageName      string `envconfig:"K6_IMAGE_NAME"`
	ImageTag       string `envconfig:"K6_IMAGE_TAG"`
	CPULimits      string `envconfig:"K6_CPU_LIMITS"`
	CPURequests    string `envconfig:"K6_CPU_REQUESTS"`
	MemoryLimits   string `envconfig:"K6_MEMORY_LIMITS"`
	MemoryRequests string `envconfig:"K6_MEMORY_REQUESTS"`
}
