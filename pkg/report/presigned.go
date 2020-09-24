package report

import (
	"net/url"
)

// NewPreSignedPutURL returns a signed URL that allows to upload a single file
func NewPreSignedPutURL(loadTestName string) *url.URL {
	if nil == minioClient {
		return nil
	}

	presignedURL, _ := minioClient.PresignedPutObject(bucketName, loadTestName, expires)

	return presignedURL
}
