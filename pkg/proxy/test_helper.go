package proxy

import (
	"bytes"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
)

// Request contains request body and contentType prepared in createRequestBody func
type Request struct {
	body        io.Reader
	contentType string
}

func createRequestBody(requestFiles map[string]string, distributedPods string, loadtestType string) (*Request, error) {
	request := &Request{}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("distributedPods", distributedPods); err != nil {
		return request, fmt.Errorf("error adding pod nr: %w", err)
	}

	if err := writer.WriteField("type", loadtestType); err != nil {
		return request, fmt.Errorf("error adding loadtest type: %w", err)
	}

	for key, val := range requestFiles {
		file, err := os.Open(val)
		if err != nil {
			return request, err
		}

		part, err := writer.CreateFormFile(key, filepath.Base(val))
		if err != nil {
			return request, err
		}

		_, _ = io.Copy(part, file)
		_ = file.Close()
	}

	err := writer.Close()
	if err != nil {
		return request, err
	}

	request.body = body
	request.contentType = writer.FormDataContentType()

	return request, nil
}
