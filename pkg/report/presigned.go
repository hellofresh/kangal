package report

import (
	"errors"
	"net/url"
)

// ErrNoMinioClient is returned when the package was not initialized with `InitObjectStorageClient`
var ErrNoMinioClient = errors.New("minio client not initialized")

// newPreSignedPutURL returns a signed URL that allows to upload a single file
func newPreSignedPutURL(loadTestName string) (*url.URL, error) {
	if nil == minioClient {
		return nil, ErrNoMinioClient
	}

	return minioClient.PresignedPutObject(bucketName, loadTestName, expires)
}
