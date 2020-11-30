package proxy

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"
	"time"

	"github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
	"github.com/semihs/goform"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/durationpb"

	apisLoadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/apis/loadtest/v1"
	grpcProxyV2 "github.com/hellofresh/kangal/pkg/proxy/rpc/pb/grpc/proxy/v2"
)

const (
	defaultMaxMemory = 32 << 20 // 32 MB
)

type createTLProxyModel struct {
	EnvVars         *goform.File `goform:"envVars"`
	TestData        *goform.File `goform:"testData"`
	TestFile        *goform.File `goform:"testFile"`
	DistributedPods int32        `goform:"distributedPods"`
	Type            string       `goform:"type"`
	Overwrite       bool         `goform:"overwrite"`
	TargetURL       string       `goform:"targetUrl"`
	Tags            string       `goform:"tags"`
	Duration        string       `goform:"duration"`
}

// multipartFormWrapper is used to convert "multipart/form-data" request to "application/json" that is handled by
// grpc-gateway. We need this because there is no way to handle file uploads out-of-the-box. Currently this middleware
// is used to handle the one and the only POST request - Load Test creation.
//
// While the solution is not so bad in general (it works ;)) the fact that we need to have manually-maintained
// entities and code around these entities (form, model, explicit fields conversion) is bad.
// One of the steps to turn this API from  experimental to production-ready is to do the following:
// TODO: generate model(s), form(s) and transformation code for "multipart/form-data" requests from ProtoBuf definition
var multipartFormWrapper = func(h *runtime.ServeMux) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.ToLower(strings.Split(r.Header.Get("Content-Type"), ";")[0]) != "multipart/form-data" {
			h.ServeHTTP(w, r)
			return
		}

		// we expect default set of marshalers here - need them to output errors if any
		_, outboundMarshaler := runtime.MarshalerForRequest(h, r)

		if err := r.ParseMultipartForm(defaultMaxMemory); err != nil {
			runtime.HTTPError(r.Context(), h, outboundMarshaler, w, r, status.Errorf(codes.InvalidArgument, "%v", err))
			return
		}

		form := getLTCreateForm()

		form.BindFromPost(r)

		if !form.IsValid() {
			var buf bytes.Buffer
			for _, e := range form.GetElements() {
				for _, ee := range e.GetErrors() {
					buf.WriteString(fmt.Sprintf("%s: %s; ", e.GetName(), fmt.Sprintf(ee.Message, ee.Args...)))
				}
			}

			runtime.HTTPError(r.Context(), h, outboundMarshaler, w, r, status.Error(codes.InvalidArgument, buf.String()))
			return
		}

		var (
			m   createTLProxyModel
			err error
		)

		form.MapTo(&m)

		createRq := &grpcProxyV2.CreateRequest{
			DistributedPods: m.DistributedPods,
			Type:            grpcProxyV2.LoadTestType(grpcProxyV2.LoadTestType_value[m.Type]),
			Overwrite:       m.Overwrite,
			TargetUrl:       m.TargetURL,
		}

		if err := readFileContents(createRq, m); err != nil {
			runtime.HTTPError(r.Context(), h, outboundMarshaler, w, r, status.Errorf(codes.InvalidArgument, "%v", err))
			return
		}

		if m.Duration != "" {
			durationTime, err := time.ParseDuration(m.Duration)
			if err != nil {
				runtime.HTTPError(r.Context(), h, outboundMarshaler, w, r, status.Errorf(codes.InvalidArgument, "%v", err))
				return
			}

			createRq.Duration = durationpb.New(durationTime)
		}

		createRq.Tags, err = apisLoadTestV1.LoadTestTagsFromString(m.Tags)
		if err != nil {
			runtime.HTTPError(r.Context(), h, outboundMarshaler, w, r, status.Errorf(codes.InvalidArgument, "%v", err))
			return
		}

		jsonBody, err := protojson.Marshal(createRq)
		if err != nil {
			runtime.HTTPError(r.Context(), h, outboundMarshaler, w, r, status.Errorf(codes.InvalidArgument, "%v", err))
			return
		}

		r.Body = ioutil.NopCloser(bytes.NewReader(jsonBody))
		r.ContentLength = int64(len(jsonBody))
		r.Header.Set("Content-Type", "application/json")

		h.ServeHTTP(w, r)
	})
}

func readFileContents(createRq *grpcProxyV2.CreateRequest, m createTLProxyModel) error {
	var err error

	createRq.EnvVars, err = ioutil.ReadAll(m.EnvVars.Binary)
	if err != nil {
		return err
	}

	createRq.TestData, err = ioutil.ReadAll(m.TestData.Binary)
	if err != nil {
		return err
	}

	createRq.TestFile, err = ioutil.ReadAll(m.TestFile.Binary)
	if err != nil {
		return err
	}

	return nil
}

func getLTCreateForm() *goform.Form {
	ltTypes := make([]*goform.ValueOption, 0, len(grpcProxyV2.LoadTestType_value))
	for k := range grpcProxyV2.LoadTestType_value {
		ltTypes = append(ltTypes, &goform.ValueOption{
			Value: k,
			Label: k,
		})
	}

	envVars := goform.NewFileElement("envVars", "envVars", []*goform.Attribute{}, []goform.ValidatorInterface{}, []goform.FilterInterface{}, "")
	testData := goform.NewFileElement("testData", "testData", []*goform.Attribute{}, []goform.ValidatorInterface{}, []goform.FilterInterface{}, "")
	testFile := goform.NewFileElement("testFile", "testFile", []*goform.Attribute{}, []goform.ValidatorInterface{}, []goform.FilterInterface{}, "")
	distributedPods := goform.NewNumberElement("distributedPods", "distributedPods", []*goform.Attribute{}, []goform.ValidatorInterface{}, []goform.FilterInterface{})
	ltType := goform.NewSelectElement("type", "type", []*goform.Attribute{}, ltTypes, []goform.ValidatorInterface{}, []goform.FilterInterface{})
	overwrite := goform.NewCheckboxElement("overwrite", "overwrite", []*goform.Attribute{}, []goform.ValidatorInterface{}, []goform.FilterInterface{})
	targetURL := goform.NewTextElement("targetUrl", "targetUrl", []*goform.Attribute{}, []goform.ValidatorInterface{}, []goform.FilterInterface{})
	tags := goform.NewTextElement("tags", "tags", []*goform.Attribute{}, []goform.ValidatorInterface{}, []goform.FilterInterface{})
	duration := goform.NewTextElement("duration", "duration", []*goform.Attribute{}, []goform.ValidatorInterface{}, []goform.FilterInterface{})

	form := goform.NewGoForm()
	form.Add(envVars)
	form.Add(testData)
	form.Add(testFile)
	form.Add(distributedPods)
	form.Add(ltType)
	form.Add(overwrite)
	form.Add(targetURL)
	form.Add(tags)
	form.Add(duration)

	return form
}
