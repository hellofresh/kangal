package jmeter

// Config specific to JMeter backend
type Config struct {
	MasterCPULimits      string `envconfig:"JMETER_MASTER_CPU_LIMITS" default:"2000m"`
	MasterCPURequests    string `envconfig:"JMETER_MASTER_CPU_REQUESTS" default:"1000m"`
	MasterMemoryLimits   string `envconfig:"JMETER_MASTER_MEMORY_LIMITS" default:"4Gi"`
	MasterMemoryRequests string `envconfig:"JMETER_MASTER_MEMORY_REQUESTS" default:"4Gi"`
	WorkerCPULimits      string `envconfig:"JMETER_WORKER_CPU_LIMITS" default:"2000m"`
	WorkerCPURequests    string `envconfig:"JMETER_WORKER_CPU_REQUESTS" default:"1000m"`
	WorkerMemoryLimits   string `envconfig:"JMETER_WORKER_MEMORY_LIMITS" default:"4Gi"`
	WorkerMemoryRequests string `envconfig:"JMETER_WORKER_MEMORY_REQUESTS" default:"4Gi"`
}
