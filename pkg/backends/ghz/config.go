package ghz

// Config specific to ghz backend
type Config struct {
	ImageName      string `envconfig:"GHZ_IMAGE_NAME" default:"hellofresh/kangal-ghz"`
	ImageTag       string `envconfig:"GHZ_IMAGE_TAG" default:"latest"`
	CPULimits      string `envconfig:"GHZ_CPU_LIMITS"`
	CPURequests    string `envconfig:"GHZ_CPU_REQUESTS"`
	MemoryLimits   string `envconfig:"GHZ_MEMORY_LIMITS"`
	MemoryRequests string `envconfig:"GHZ_MEMORY_REQUESTS"`
}
