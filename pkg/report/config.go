package report

// Config is the report package related basic config
type Config struct {
	// S3 compatible configuration access keys and endpoints needed to store load test reports
	AWSAccessKeyID     string `envconfig:"AWS_ACCESS_KEY_ID" default:""`
	AWSSecretAccessKey string `envconfig:"AWS_SECRET_ACCESS_KEY" default:""`
	AWSRegion          string `envconfig:"AWS_DEFAULT_REGION" default:""`
	AWSEndpointURL     string `envconfig:"AWS_ENDPOINT_URL" default:""`
	AWSBucketName      string `envconfig:"AWS_BUCKET_NAME" default:""`
}
