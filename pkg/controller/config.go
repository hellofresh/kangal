package controller

import (
	"time"

	"github.com/hellofresh/kangal/pkg/core/observability"
	"github.com/hellofresh/kangal/pkg/report"
)

// Config is the possible Kangal Controller configurations
type Config struct {
	Debug    bool `envconfig:"DEBUG"`
	HTTPPort int  `envconfig:"WEB_HTTP_PORT" default:"8080"`
	Logger   observability.LoggerConfig
	// CleanUpThresholdEnvVar is used if we want to increase the amount of time a
	// load test lives for, the default is 1 hour. (ex. 5h)
	CleanUpThreshold time.Duration `envconfig:"CLEANUP_THRESHOLD" default:"1h"`
	// S3 compatible configuration access keys and endpoints needed to store load test reports
	Report report.Config

	MasterURL            string
	KubeConfig           string
	NamespaceAnnotations map[string]string
	PodAnnotations       map[string]string
}
