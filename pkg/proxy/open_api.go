package proxy

import (
	"io/ioutil"
	"net/http"
	"os"
	"path/filepath"

	"github.com/rs/cors"
	"github.com/tidwall/sjson"
	"go.uber.org/zap"

	mPkg "github.com/hellofresh/kangal/pkg/core/middleware"
)

// OpenAPISpecHandler returns a http handler for OpenAPI Spec
func OpenAPISpecHandler(cfg OpenAPIConfig) func(w http.ResponseWriter, r *http.Request) {
	openAPISpec := filepath.Join(cfg.SpecPath, cfg.SpecFile)

	// use OpenAPI spec as is
	if cfg.ServerURL == "" {
		return func(w http.ResponseWriter, r *http.Request) {
			http.ServeFile(w, r, openAPISpec)
		}
	}

	// override "servers" part of the spec with private destination for easier apps configuring
	_, err := os.Stat(openAPISpec)
	if err != nil {
		return func(w http.ResponseWriter, r *http.Request) {
			mPkg.GetLogger(r.Context()).Error("Can not read OpenAPI spec file", zap.Error(err), zap.String("path", openAPISpec))

			switch {
			case os.IsNotExist(err):
				http.Error(w, "OpenAPI spec file not found", http.StatusNotFound)
			case os.IsPermission(err):
				http.Error(w, "OpenAPI spec file is not accessible", http.StatusForbidden)
			default:
				http.Error(w, "Can not read OpenAPI spec file", http.StatusInternalServerError)
			}
		}
	}

	spec, err := ioutil.ReadFile(openAPISpec)
	if err != nil {
		return func(w http.ResponseWriter, r *http.Request) {
			mPkg.GetLogger(r.Context()).Error("Failed to read OpenAPI spec file", zap.Error(err), zap.String("path", openAPISpec))
			http.Error(w, "Failed to read OpenAPI spec file", http.StatusInternalServerError)
		}
	}

	value, err := sjson.SetBytes(spec, "servers.0.url", cfg.ServerURL)
	if err != nil {
		return func(w http.ResponseWriter, r *http.Request) {
			mPkg.GetLogger(r.Context()).Error("Could not set custom server URL", zap.Error(err))
			http.Error(w, "Could not set custom server URL", http.StatusInternalServerError)
		}
	}

	if cfg.ServerDescription != "" {
		value, err = sjson.SetBytes(value, "servers.0.description", cfg.ServerDescription)
		if err != nil {
			return func(w http.ResponseWriter, r *http.Request) {
				mPkg.GetLogger(r.Context()).Error("Could not set custom server description", zap.Error(err))
				http.Error(w, "Could not set custom server description", http.StatusInternalServerError)
			}
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", mimeJSON)
		w.Write(value)
	}
}

// OpenAPIUIHandler returns a http handler for UI built out of OpenAPI Spec
func OpenAPIUIHandler(cfg OpenAPIConfig) func(w http.ResponseWriter, r *http.Request) {
	if cfg.UIUrl == "" {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "OpenAPI UI URL is not set, check service configuration if you maintain it or contact maintainers otherwise", http.StatusNotFound)
		}
	}

	return func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, cfg.UIUrl, http.StatusFound)
	}
}

// OpenAPISpecCORSMiddleware returns a middleware to handle CORS requests for OpenAPI spec
func OpenAPISpecCORSMiddleware(cfg OpenAPIConfig) func(http.Handler) http.Handler {
	return cors.New(cors.Options{
		AllowedOrigins:     cfg.AccessControlAllowOrigin,
		AllowedMethods:     []string{"OPTIONS", "HEAD", "GET"},
		AllowedHeaders:     cfg.AccessControlAllowHeaders,
		ExposedHeaders:     []string{"*"},
		OptionsPassthrough: true,
		AllowCredentials:   true,
	}).Handler
}
