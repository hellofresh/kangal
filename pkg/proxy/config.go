package proxy

import (
	"github.com/hellofresh/kangal/pkg/core/observability"
	"github.com/hellofresh/kangal/pkg/report"
)

// Config is the possible Proxy configurations
type Config struct {
	Debug           bool `envconfig:"DEBUG"`
	HTTPPort        int  `envconfig:"WEB_HTTP_PORT" default:"8080"`
	Logger          observability.LoggerConfig
	OpenAPI         OpenAPIConfig
	Report          report.Config
	MaxLoadTestsRun int
	MasterURL       string
}

// OpenAPIConfig is the OpenAPI specification-specific parameters
type OpenAPIConfig struct {
	SpecPath          string `envconfig:"OPEN_API_SPEC_PATH" default:"/etc/kangal"`
	SpecFile          string `envconfig:"OPEN_API_SPEC_FILE" default:"openapi.json"`
	ServerURL         string `envconfig:"OPEN_API_SERVER_URL"`
	ServerDescription string `envconfig:"OPEN_API_SERVER_DESCRIPTION"`
	UIUrl             string `envconfig:"OPEN_API_UI_URL"`

	AccessControlAllowOrigin  []string `envconfig:"OPEN_API_CORS_ALLOW_ORIGIN" default:"*"`
	AccessControlAllowHeaders []string `envconfig:"OPEN_API_CORS_ALLOW_HEADERS" default:"Content-Type,api_key,Authorization"`
}
