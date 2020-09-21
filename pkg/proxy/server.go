package proxy

import (
	"context"
	"fmt"
	"net/http"

	"contrib.go.opencensus.io/exporter/prometheus"
	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"go.opencensus.io/plugin/ochttp"
	"go.uber.org/zap"
	"k8s.io/client-go/kubernetes"

	cHttp "github.com/hellofresh/kangal/pkg/core/http"
	mPkg "github.com/hellofresh/kangal/pkg/core/middleware"
	loadTestV1 "github.com/hellofresh/kangal/pkg/kubernetes/generated/clientset/versioned/typed/loadtest/v1"
	"github.com/hellofresh/kangal/pkg/report"
)

// Runner encapsulates all Kangal Proxy API server dependencies
type Runner struct {
	Exporter       *prometheus.Exporter
	KubeClient     kubernetes.Interface
	LoadTestClient loadTestV1.LoadTestInterface
	Logger         *zap.Logger
}

// RunServer runs Kangal proxy API
func RunServer(ctx context.Context, cfg Config, rr Runner) error {

	proxyHandler := NewProxy(cfg.MaxLoadTestsRun, rr.LoadTestClient, rr.KubeClient)
	// Start instrumented server
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(mPkg.NewLogger(rr.Logger).Handler)
	r.Use(mPkg.NewRequestLogger().Handler)
	r.Use(mPkg.Recovery)
	r.Use(render.SetContentType(render.ContentTypeJSON))
	r.Use(OpenAPISpecCORSMiddleware(cfg.OpenAPI))

	r.Get("/status", cHttp.LivenessHandler("Kangal Proxy"))
	r.Handle("/metrics", rr.Exporter)

	// ---------------------------------------------------------------------- //
	// LoadTest Proxy CRUD
	// ---------------------------------------------------------------------- //
	loadtestRoute := "/load-test"
	loadtestRouteWithID := fmt.Sprintf("%s/{id}", loadtestRoute)

	r.Method(http.MethodPost,
		loadtestRoute,
		ochttp.WithRouteTag(http.HandlerFunc(proxyHandler.Create), loadtestRoute),
	)

	r.Method(http.MethodGet,
		loadtestRouteWithID,
		ochttp.WithRouteTag(http.HandlerFunc(proxyHandler.Get), loadtestRouteWithID),
	)

	r.Method(http.MethodDelete,
		loadtestRouteWithID,
		ochttp.WithRouteTag(http.HandlerFunc(proxyHandler.Delete), loadtestRouteWithID),
	)

	// ---------------------------------------------------------------------- //
	// LoadTest API Documentation
	// ---------------------------------------------------------------------- //
	r.Get("/", OpenAPIUIHandler(cfg.OpenAPI))
	r.Get("/openapi", OpenAPISpecHandler(cfg.OpenAPI))

	r.Get("/load-test/{id}/logs", proxyHandler.GetLogs)

	// ---------------------------------------------------------------------- //
	// LoadTest reports
	// ---------------------------------------------------------------------- //

	// we need to do redirects because we serve html reports
	redirectReportHandler := func(w http.ResponseWriter, r *http.Request) {
		// FileServer removes end slashes from the path and redirects to .. - as a workaround we changed dashboard index.html to main.html
		url := fmt.Sprintf("%s/main.html", r.URL.Host+r.URL.Path)
		http.Redirect(w, r, url, http.StatusMovedPermanently)
	}
	r.Get("/load-test/{id}/report", redirectReportHandler)
	r.Get("/load-test/{id}/report/", redirectReportHandler)
	r.Get("/load-test/{id}/report/*", report.ShowHandler())

	address := fmt.Sprintf(":%d", cfg.HTTPPort)
	rr.Logger.Info("Running HTTP server...", zap.String("address", address))

	// Try and run http server, fail on error
	err := http.ListenAndServe(address, &ochttp.Handler{Handler: r})
	if err != nil {
		return fmt.Errorf("failed to run HTTP server: %w", err)
	}
	return nil
}
