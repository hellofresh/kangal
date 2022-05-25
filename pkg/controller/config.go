package controller

import (
	"time"

	"github.com/hellofresh/kangal/pkg/core/observability"
	"github.com/hellofresh/kangal/pkg/kubernetes"
)

// Config is the possible Kangal Controller configurations
type Config struct {
	HTTPPort int `envconfig:"WEB_HTTP_PORT" default:"8080"`
	Logger   observability.LoggerConfig

	// CleanUpThresholdEnvVar is used if we want to increase the amount of time a
	// load test lives for, the default is 1 hour. (ex. 5h)
	CleanUpThreshold time.Duration `envconfig:"CLEANUP_THRESHOLD" default:"1h"`

	// S3 compatible configuration access keys and endpoints needed to store load test reports
	KangalProxyURL string `envconfig:"KANGAL_PROXY_URL" default:""`

	// KubeClientTimeout specifies timeout for each operation done by kube client
	KubeClientTimeout time.Duration `envconfig:"KUBE_CLIENT_TIMEOUT" default:"5s"`

	// SyncHandlerTimeout specifies the time limit for each sync operation
	SyncHandlerTimeout time.Duration `envconfig:"SYNC_HANDLER_TIMEOUT" default:"60s"`

	MasterURL            string
	KubeConfig           string
	NamespaceAnnotations map[string]string
	PodAnnotations       map[string]string
	NodeSelectors        map[string]string
	Tolerations          kubernetes.Tolerations
}
