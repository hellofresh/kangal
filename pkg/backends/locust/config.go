package locust

// Config specific to Locust backend
type Config struct {
	Image                string `envconfig:"LOCUST_IMAGE"`
	ImageName            string `envconfig:"LOCUST_IMAGE_NAME" default:"locustio/locust"`
	ImageTag             string `envconfig:"LOCUST_IMAGE_TAG" default:"latest"`
	MasterCPULimits      string `envconfig:"LOCUST_MASTER_CPU_LIMITS"`
	MasterCPURequests    string `envconfig:"LOCUST_MASTER_CPU_REQUESTS"`
	MasterMemoryLimits   string `envconfig:"LOCUST_MASTER_MEMORY_LIMITS"`
	MasterMemoryRequests string `envconfig:"LOCUST_MASTER_MEMORY_REQUESTS"`
	WorkerCPULimits      string `envconfig:"LOCUST_WORKER_CPU_LIMITS"`
	WorkerCPURequests    string `envconfig:"LOCUST_WORKER_CPU_REQUESTS"`
	WorkerMemoryLimits   string `envconfig:"LOCUST_WORKER_MEMORY_LIMITS"`
	WorkerMemoryRequests string `envconfig:"LOCUST_WORKER_MEMORY_REQUESTS"`
}
