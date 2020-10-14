package report

import (
	"errors"
	"net/url"
)

var ErrNoMinioClient = errors.New("Minio client not initialized")

// newPreSignedPutURL returns a signed URL that allows to upload a single file
func newPreSignedPutURL(loadTestName string) (*url.URL, error) {
	if nil == minioClient {
		return nil, ErrNoMinioClient
	}

	return minioClient.PresignedPutObject(bucketName, loadTestName, expires)
}
