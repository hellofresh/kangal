package ghz

// Config specific to ghz backend
type Config struct {
	Image          string `envconfig:"GHZ_IMAGE"`
	ImageName      string `envconfig:"GHZ_IMAGE_NAME"`
	ImageTag       string `envconfig:"GHZ_IMAGE_TAG"`
	CPULimits      string `envconfig:"GHZ_CPU_LIMITS"`
	CPURequests    string `envconfig:"GHZ_CPU_REQUESTS"`
	MemoryLimits   string `envconfig:"GHZ_MEMORY_LIMITS"`
	MemoryRequests string `envconfig:"GHZ_MEMORY_REQUESTS"`
}
