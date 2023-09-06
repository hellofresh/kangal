package jmeter

import (
	"time"
)

// Config specific to JMeter backend
type Config struct {
	MasterImageName         string        `envconfig:"JMETER_MASTER_IMAGE_NAME" default:"hellofresh/kangal-jmeter-master"`
	MasterImageTag          string        `envconfig:"JMETER_MASTER_IMAGE_TAG" default:"latest"`
	MasterCPULimits         string        `envconfig:"JMETER_MASTER_CPU_LIMITS"`
	MasterCPURequests       string        `envconfig:"JMETER_MASTER_CPU_REQUESTS"`
	MasterMemoryLimits      string        `envconfig:"JMETER_MASTER_MEMORY_LIMITS"`
	MasterMemoryRequests    string        `envconfig:"JMETER_MASTER_MEMORY_REQUESTS"`
	WorkerImageName         string        `envconfig:"JMETER_WORKER_IMAGE_NAME" default:"hellofresh/kangal-jmeter-worker"`
	WorkerImageTag          string        `envconfig:"JMETER_WORKER_IMAGE_TAG" default:"latest"`
	WorkerCPULimits         string        `envconfig:"JMETER_WORKER_CPU_LIMITS"`
	WorkerCPURequests       string        `envconfig:"JMETER_WORKER_CPU_REQUESTS"`
	WorkerMemoryLimits      string        `envconfig:"JMETER_WORKER_MEMORY_LIMITS"`
	WorkerMemoryRequests    string        `envconfig:"JMETER_WORKER_MEMORY_REQUESTS"`
	TestDataDecompressImage string        `envconfig:"JMETER_TESTDATA_DECOMPRESS_IMAGE" default:"alpine:latest"`
	RemoteCustomDataImage   string        `envconfig:"JMETER_WORKER_REMOTE_CUSTOM_DATA_IMAGE" default:"rclone/rclone:latest"`
	WaitForResourceTimeout  time.Duration `envconfig:"WAIT_FOR_RESOURCE_TIMEOUT" default:"30s"`
}
