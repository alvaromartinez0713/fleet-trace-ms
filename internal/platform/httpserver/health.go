package httpserver

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

const (
	livenessPath          = "/healthz"
	readinessPath         = "/readyz"
	readinessCheckTimeout = 2 * time.Second
)

// liveness reports that the process is up and serving HTTP.
func liveness(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

// readiness runs every check and returns 503 when any of them fails.
// The response lists failed check names only. Error details go to the log, not to the client.
func readiness(checks []ReadinessCheck, logger *slog.Logger) gin.HandlerFunc {
	return func(c *gin.Context) {
		var failed []string
		for _, check := range checks {
			ctx, cancel := context.WithTimeout(c.Request.Context(), readinessCheckTimeout)
			err := check.Check(ctx)
			cancel()
			if err != nil {
				failed = append(failed, check.Name)
				logger.WarnContext(c.Request.Context(), "readiness check failed",
					slog.String("check", check.Name),
					slog.Any("error", err),
				)
			}
		}

		if len(failed) > 0 {
			c.JSON(http.StatusServiceUnavailable, gin.H{"status": "unavailable", "failed": failed})
			return
		}
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	}
}
