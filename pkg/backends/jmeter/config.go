package jmeter

// Config specific to JMeter backend
type Config struct {
	MasterCPULimits      string `envconfig:"JMETER_MASTER_CPU_LIMITS"`
	MasterCPURequests    string `envconfig:"JMETER_MASTER_CPU_REQUESTS"`
	MasterMemoryLimits   string `envconfig:"JMETER_MASTER_MEMORY_LIMITS"`
	MasterMemoryRequests string `envconfig:"JMETER_MASTER_MEMORY_REQUESTS"`
	WorkerCPULimits      string `envconfig:"JMETER_WORKER_CPU_LIMITS"`
	WorkerCPURequests    string `envconfig:"JMETER_WORKER_CPU_REQUESTS"`
	WorkerMemoryLimits   string `envconfig:"JMETER_WORKER_MEMORY_LIMITS"`
	WorkerMemoryRequests string `envconfig:"JMETER_WORKER_MEMORY_REQUESTS"`
}
