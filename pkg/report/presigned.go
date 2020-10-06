package report

import (
	"fmt"
	"net/url"
)

// NewPreSignedPutURL returns a signed URL that allows to upload a single file
func NewPreSignedPutURL(loadTestName string) *url.URL {
	if nil == minioClient {
		return nil
	}

	presignedURL, err := minioClient.PresignedPutObject(bucketName, loadTestName, expires)
	if nil != err {
		fmt.Printf("failed to presign url: %s", err.Error())
	}

	return presignedURL
}
