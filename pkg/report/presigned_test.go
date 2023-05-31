package report

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewPreSignedPutURL(t *testing.T) {
	err := InitObjectStorageClient(Config{
		AWSAccessKeyID:     "access-key-id",
		AWSSecretAccessKey: "secret-access-key",
		AWSRegion:          "region",
		AWSEndpointURL:     "localhost:80",
		AWSBucketName:      "bucket-name",
	})
	assert.Nil(t, err)

	loadTestName := "fake-loadtest"
	url, err := newPreSignedPutURL(context.Background(), loadTestName)

	assert.NoError(t, err)
	assert.NotNil(t, url)
	assert.Contains(t, url.String(), loadTestName)
}

func TestNilClientNewPreSignedPutURL(t *testing.T) {
	minioClient = nil
	loadTestName := "fake-loadtest"
	url, err := newPreSignedPutURL(context.Background(), loadTestName)
	assert.EqualError(t, err, ErrNoMinioClient.Error())
	assert.Nil(t, url)
}
