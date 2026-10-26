// Package httpserver builds the Gin router and the HTTP server shared by all services.
// It is the only platform package that imports Gin.
package httpserver

import (
	"context"
	"log/slog"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

// ReadinessCheck is a dependency probe run by GET /readyz.
type ReadinessCheck struct {
	Name  string
	Check func(ctx context.Context) error
}

// Options configures the router.
type Options struct {
	ServiceName string
	Logger      *slog.Logger // Defaults to slog.Default() when nil.
	Tracing     bool         // Adds the otelgin middleware when true.
	Readiness   []ReadinessCheck
}

// NewRouter returns a Gin engine with recovery, optional tracing and request logging middleware,
// plus the GET /healthz and GET /readyz endpoints. It does not set the global Gin mode. The caller decides that.
func NewRouter(opts Options) *gin.Engine {
	logger := opts.Logger
	if logger == nil {
		logger = slog.Default()
	}

	engine := gin.New()
	engine.Use(recoveryMiddleware(logger))
	if opts.Tracing {
		engine.Use(otelgin.Middleware(opts.ServiceName))
	}
	engine.Use(requestLogger(logger))

	engine.GET(livenessPath, liveness)
	engine.GET(readinessPath, readiness(opts.Readiness, logger))

	return engine
}
