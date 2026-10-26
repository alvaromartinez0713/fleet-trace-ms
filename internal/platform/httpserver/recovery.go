package httpserver

import (
	"log/slog"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
)

// recoveryMiddleware turns a handler panic into a 500 response and logs the panic with its stack.
// The client receives no panic details. A nil writer disables Gin's own plain-text report,
// so the panic is logged once, through slog.
func recoveryMiddleware(logger *slog.Logger) gin.HandlerFunc {
	return gin.CustomRecoveryWithWriter(nil, func(c *gin.Context, recovered any) {
		logger.ErrorContext(c.Request.Context(), "panic recovered",
			slog.String("method", c.Request.Method),
			slog.String("route", c.FullPath()),
			slog.Any("panic", recovered),
			slog.String("stack", string(debug.Stack())),
		)
		c.AbortWithStatus(http.StatusInternalServerError)
	})
}
