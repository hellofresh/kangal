package proxy

import (
	"bytes"
	"encoding/json"
	"io"
	"io/ioutil"
	"log"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
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

func createRequestWrapper(t *testing.T, requestFiles map[string]string, distributedPods string, loadtestType string, tagsString string, overwrite bool, masterImage string, workerImage string) *Request {
	t.Helper()

	request := &Request{}
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	err := writer.WriteField("distributedPods", distributedPods)
	require.NoError(t, err)

	err = writer.WriteField("tags", tagsString)
	require.NoError(t, err)

	err = writer.WriteField("type", loadtestType)
	require.NoError(t, err)

	err = writer.WriteField("masterImage", masterImage)
	require.NoError(t, err)

	err = writer.WriteField("workerImage", workerImage)
	require.NoError(t, err)

	if overwrite {
		err = writer.WriteField("overwrite", "true")
		require.NoError(t, err)
	}

	for key, val := range requestFiles {
		file, err := os.Open(val)
		require.NoError(t, err)

		part, err := writer.CreateFormFile(key, filepath.Base(val))
		require.NoError(t, err)

		_, err = io.Copy(part, file)
		require.NoError(t, err)

		err = file.Close()
		require.NoError(t, err)
	}

	err = writer.Close()
	require.NoError(t, err)

	request.body = body
	request.contentType = writer.FormDataContentType()

	return request
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

func parseBody(t *testing.T, res *http.Response) (createdLoadTestName string) {
	t.Helper()

	var dat LoadTestStatus

	defer func() {
		err := res.Body.Close()
		assert.NoError(t, err)
	}()

	respBody, err := ioutil.ReadAll(res.Body)
	require.NoError(t, err)

	err = json.Unmarshal(respBody, &dat)
	require.NoError(t, err)

	return dat.Namespace
}
