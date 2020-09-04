package proxy

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/render"
	"go.uber.org/zap"
	kube "k8s.io/client-go/kubernetes"
	restClient "k8s.io/client-go/rest"

	loadtest "github.com/hellofresh/kangal/pkg/controller"
	cHttp "github.com/hellofresh/kangal/pkg/core/http"
	mPkg "github.com/hellofresh/kangal/pkg/core/middleware"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned/typed/loadtest/v1"
)

// LoadTestGetLogsHandler gets the logs for the requested load test
func LoadTestGetLogsHandler(kubeClient kube.Interface, loadTestClient loadTestV1.LoadTestInterface) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		logger := mPkg.GetLogger(r.Context())
		ltID := chi.URLParam(r, loadTestID)
		logger.Info("Retrieving info for loadtest", zap.Any("ltID", ltID))

		ctx, cancel := context.WithTimeout(context.Background(), loadtest.KubeTimeout)
		defer cancel()
		loadTest, err := loadtest.GetLoadtestCR(ctx, loadTestClient, ltID, logger)
		if err != nil {
			logger.Error("Could not get load test info with error:", zap.Error(err))
			render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest, err.Error()))
			return
		}

		namespace := loadTest.Status.Namespace
		// if no namespace was created we can not get errors
		// TODO: maybe this should just be empty and not an error?
		if namespace == "" {
			render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest, "got empty request"))
			return
		}

		ctxJMeterLogs, cancelJMeterLogs := context.WithTimeout(context.Background(), loadtest.KubeTimeout)
		defer cancelJMeterLogs()
		logsRequest, err := loadtest.GetMasterPodLogs(ctxJMeterLogs, kubeClient, namespace, logger)
		if err != nil {
			logger.Error("Could not get load test logs request:", zap.Error(err))
			render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest, err.Error()))
			return
		}

		logs, err := doRequest(logsRequest)
		if err != nil {
			logger.Error("Could not get load test logs:", zap.Error(err))
			render.Render(w, r, cHttp.ErrResponse(http.StatusBadRequest, err.Error()))
			return
		}

		io.WriteString(w, string(logs))
		return
	}
}

func doRequest(req *restClient.Request) ([]byte, error) {
	stream, err := req.Stream(context.Background())
	if err != nil {
		return []byte{}, fmt.Errorf("error in opening stream: %w", err)
	}
	defer stream.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, stream)
	if err != nil {
		return []byte{}, errors.New("error in copy information from podLogs to buf")
	}

	return buf.Bytes(), nil
}
