package cmd

import (
	"github.com/spf13/cobra"
)

// GlobalConfig holds the config values that are common to all applications
type GlobalConfig struct {
	Log   LogConfig

	// S3 compatible configuration access keys and endpoints needed to store load test reports
	AWSAccessKeyID     string `envconfig:"AWS_ACCESS_KEY_ID" default:""`
	AWSSecretAccessKey string `envconfig:"AWS_SECRET_ACCESS_KEY" default:""`
	AWSRegion          string `envconfig:"AWS_DEFAULT_REGION" default:""`
	AWSEndpointURL     string `envconfig:"AWS_ENDPOINT_URL" default:""`
	AWSBucketName      string `envconfig:"AWS_BUCKET_NAME" default:""`
}

// LogConfig is logging basic config
type LogConfig struct {
	Level string `envconfig:"LOG_LEVEL" default:"info"`
	Type  string `envconfig:"LOG_TYPE" default:"kangal"`
}

// NewRootCmd creates a new instance of the root command
func NewRootCmd(version string) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "kangal",
		Short:   "Kangal is an application for creating environments for performance testing",
		Version: version,
	}

	cmd.AddCommand(NewProxyCmd())
	cmd.AddCommand(NewControllerCmd())

	return cmd
}
