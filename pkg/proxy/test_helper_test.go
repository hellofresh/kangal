package proxy

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"
	"k8s.io/client-go/kubernetes"

	testhelper "github.com/hellofresh/kangal/pkg/controller"
	clientSetV "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned"
)

// Request contains request body and contentType prepared in createRequestBody func
type Request struct {
	body        io.Reader
	contentType string
}

func createRequestWrapper(requestFiles map[string]string, distributedPods string, loadtestType string, tagsString string, overwrite bool) (*Request, error) {
	request := &Request{}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	if err := writer.WriteField("distributedPods", distributedPods); err != nil {
		return nil, fmt.Errorf("error adding pod nr: %w", err)
	}

	if err := writer.WriteField("tags", tagsString); err != nil {
		return nil, fmt.Errorf("error adding tags: %w", err)
	}

	if err := writer.WriteField("type", loadtestType); err != nil {
		return nil, fmt.Errorf("error adding loadtest type: %w", err)
	}

	if overwrite {
		if err := writer.WriteField("overwrite", "true"); err != nil {
			return nil, fmt.Errorf("error adding loadtest overwrite: %w", err)
		}
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
		return nil, err
	}

	request.body = body
	request.contentType = writer.FormDataContentType()

	return request, nil
}

func kubeTestClient() clientSetV.Clientset {
	if len(os.Getenv("KUBECONFIG")) == 0 {
		log.Println("Skipping kube config builder, KUBECONFIG is missed")
		return clientSetV.Clientset{}
	}
	config, err := testhelper.BuildConfig()

	clientSet, err := clientSetV.NewForConfig(config)
	if err != nil {
		log.Fatal(err)
	}

	return *clientSet
}

func kubeClient(t *testing.T) *kubernetes.Clientset {
	t.Helper()

	config, err := testhelper.BuildConfig()
	require.NoError(t, err)

	cSet, err := kubernetes.NewForConfig(config)
	require.NoError(t, err)

	return cSet
}

func parseBody(res *http.Response) (createdLoadTestName string) {
	var dat LoadTestStatus

	defer res.Body.Close()
	respbody, err := ioutil.ReadAll(res.Body)
	if err != nil {
		log.Fatal("Unable to read the response body:", err)
	}

	unmarshalErr := json.Unmarshal(respbody, &dat)
	if unmarshalErr != nil {
		log.Fatal(fmt.Sprintf("The response body was unable to be unmarshaled: %s", string(respbody)), err)
	}

	return dat.Namespace
}
