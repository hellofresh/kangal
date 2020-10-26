package jmeter

// Config specific to JMeter backend
type Config struct {
	MasterImageName      string `envconfig:"JMETER_MASTER_IMAGE_NAME"`
	MasterImageTag       string `envconfig:"JMETER_MASTER_IMAGE_TAG"`
	MasterCPULimits      string `envconfig:"JMETER_MASTER_CPU_LIMITS"`
	MasterCPURequests    string `envconfig:"JMETER_MASTER_CPU_REQUESTS"`
	MasterMemoryLimits   string `envconfig:"JMETER_MASTER_MEMORY_LIMITS"`
	MasterMemoryRequests string `envconfig:"JMETER_MASTER_MEMORY_REQUESTS"`
	WorkerImageName      string `envconfig:"JMETER_WORKER_IMAGE_NAME"`
	WorkerImageTag       string `envconfig:"JMETER_WORKER_IMAGE_TAG"`
	WorkerCPULimits      string `envconfig:"JMETER_WORKER_CPU_LIMITS"`
	WorkerCPURequests    string `envconfig:"JMETER_WORKER_CPU_REQUESTS"`
	WorkerMemoryLimits   string `envconfig:"JMETER_WORKER_MEMORY_LIMITS"`
	WorkerMemoryRequests string `envconfig:"JMETER_WORKER_MEMORY_REQUESTS"`
}
