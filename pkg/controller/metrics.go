package controller

import (
	"fmt"
	"net/http"

	"github.com/go-chi/chi"
	"github.com/go-chi/chi/middleware"
	"github.com/go-chi/render"
	"go.opencensus.io/plugin/ochttp"
	"go.uber.org/zap"

	cHttp "github.com/hellofresh/kangal/pkg/core/http"
	mPkg "github.com/hellofresh/kangal/pkg/core/middleware"
)

//RunMetricsServer starts Prometheus metrics server
func RunMetricsServer(cfg Config, rr Runner, stopChan chan struct{}) error {
	r := chi.NewRouter()
	// Define Middleware
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(mPkg.Recovery)
	r.Use(render.SetContentType(render.ContentTypeJSON))

	// Register Routes
	r.Get("/", cHttp.LivenessHandler("Kangal Controller"))
	r.Get("/status", cHttp.LivenessHandler("Kangal Controller"))
	r.Handle("/metrics", rr.Exporter)

	// Run HTTP Server
	address := fmt.Sprintf(":%d", cfg.HTTPPort)
	rr.Logger.Info("Running HTTP server...", zap.String("address", address))

	go func() {
		// Try and run http server, fail on error
		if err := http.ListenAndServe(address, &ochttp.Handler{Handler: r}); err != nil {
			rr.Logger.Error("Failed to run HTTP server", zap.Error(err))
			close(stopChan)
		}
	}()

	return nil
}
