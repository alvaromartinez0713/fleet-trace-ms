package httpserver

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// unmatchedRoute is the route label used when no route matched the request.
const unmatchedRoute = "unmatched"

// requestLogger logs one record per request after the handler runs.
// Records for 5xx responses use the error level, all others use info. Health probes are not logged,
// because orchestrators call them often and they would flood the log.
func requestLogger(logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		c.Next()

		route := c.FullPath()
		if route == livenessPath || route == readinessPath {
			return
		}
		if route == "" {
			route = unmatchedRoute
		}

		status := c.Writer.Status()
		level := slog.LevelInfo
		if status >= http.StatusInternalServerError {
			level = slog.LevelError
		}

		logger.LogAttrs(c.Request.Context(), level, "http request",
			slog.String("method", c.Request.Method),
			slog.String("route", route),
			slog.Int("status", status),
			slog.Int64("duration_ms", time.Since(start).Milliseconds()),
			slog.String("client_ip", c.ClientIP()),
		)
	}
}
