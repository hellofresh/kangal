package report

import (
	"errors"
	"net/http"
	"strings"
	"time"

	"github.com/minio/minio-go/v6"
	"github.com/minio/minio-go/v6/pkg/credentials"

	report "github.com/hellofresh/kangal/pkg/report/minio"
)

var (
	minioClient *minio.Client
	bucketName  string
	fs          http.FileSystem
	expires     time.Duration
)

//InitObjectStorageClient inits new minio backend client to work with S3 compatible storages
func InitObjectStorageClient(cfg Config) error {
	// minio doesn't like http schema endpoints - dial tcp: too many colons in address
	endpoint := strings.Replace(cfg.AWSEndpointURL, "https://", "", -1)
	endpoint = strings.Replace(endpoint, "http://", "", -1)

	if cfg.AWSBucketName == "" {
		return errors.New("bucket name is empty")
	}
	bucketName = cfg.AWSBucketName

	var err error

	var awsCredProviders = []credentials.Provider{
		&credentials.EnvAWS{},
		&credentials.FileAWSCredentials{},
		&credentials.IAM{
			Client: &http.Client{
				Timeout: time.Second * 5,
			},
		},
		&credentials.EnvMinio{},
	}

	if cfg.AWSAccessKeyID != "" && cfg.AWSSecretAccessKey != "" {
		awsCredProviders = []credentials.Provider{
			&credentials.Static{
				Value: credentials.Value{
					AccessKeyID:     cfg.AWSAccessKeyID,
					SecretAccessKey: cfg.AWSSecretAccessKey,
				},
			},
		}
	}
	creds := credentials.NewChainCredentials(awsCredProviders)

	// Init object storage (S3 compatible) client
	minioClient, err = minio.NewWithCredentials(endpoint, creds, cfg.AWSUseHTTPS, cfg.AWSRegion)
	if err != nil {
		return err
	}
	// Init file system - in our case it is Minio client which exposes object storage bucket
	fs = &report.MinioFileSystem{Client: minioClient, Bucket: bucketName}
	// Init PreSigned URL expiration time
	if "" == cfg.AWSPresignedExpires {
		cfg.AWSPresignedExpires = "30m" // defaults to 30 minutes
	}
	expires, err = time.ParseDuration(cfg.AWSPresignedExpires)
	if nil != err {
		return err
	}
	return nil
}
