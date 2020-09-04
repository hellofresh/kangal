package report

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInitObjectStorageClient(t *testing.T) {
	type args struct {
		cfg Config
	}
	tests := []struct {
		name              string
		args              args
		wantErr           bool
		minioClientIsNil  bool
		bucketNameIsEmpty bool
	}{
		{
			name: "Test with empty bucket name",
			args: args{
				cfg: Config{
					AWSAccessKeyID:     "access-key-id",
					AWSSecretAccessKey: "secret-access-key",
					AWSRegion:          "region",
					AWSEndpointURL:     "localhost:80",
					AWSBucketName:      "",
				},
			},
			wantErr:           true,
			minioClientIsNil:  true,
			bucketNameIsEmpty: true,
		},
		{
			name: "Test with correct input data",
			args: args{
				cfg: Config{
					AWSAccessKeyID:     "access-key-id",
					AWSSecretAccessKey: "secret-access-key",
					AWSRegion:          "region",
					AWSEndpointURL:     "localhost:80",
					AWSBucketName:      "some-bucket",
				},
			},
			wantErr:           false,
			minioClientIsNil:  false,
			bucketNameIsEmpty: false,
		},
		{
			name: "Test with endpoint schema - avoid dial tcp: too many colons in address",
			args: args{
				cfg: Config{
					AWSAccessKeyID:     "access-key-id",
					AWSSecretAccessKey: "secret-access-key",
					AWSRegion:          "region",
					AWSEndpointURL:     "https://localhost:80",
					AWSBucketName:      "some-bucket",
				},
			},
			wantErr:           false,
			minioClientIsNil:  false,
			bucketNameIsEmpty: false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			minioClient = nil
			bucketName = ""
			if err := InitObjectStorageClient(tt.args.cfg); (err != nil) != tt.wantErr {
				t.Errorf("InitObjectStorageClient() error = %v, wantErr %v", err, tt.wantErr)
			}

			if tt.minioClientIsNil {
				assert.Nil(t, minioClient)
			}
			if tt.bucketNameIsEmpty {
				assert.Empty(t, bucketName)
			}
		})
	}
}
