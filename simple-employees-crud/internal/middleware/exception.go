package middleware

import (
	"fmt"
	"net/http"
	"runtime/debug"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"simple-employees-crud/pkg/apperror"
	"simple-employees-crud/pkg/logger"
	"simple-employees-crud/pkg/response"
)

// Exception is the centralized error-handling middleware.
//
// Handlers must NEVER write error responses directly.  Instead they call:
//
//	_ = c.Error(err)    // attach the error to the context
//	c.Abort()           // stop handler chain
//	return
//
// This middleware runs after c.Next() returns and converts any attached
// *apperror.AppError into a properly structured JSON response.
// Non-AppError values (e.g. raw errors from third-party code) are mapped to
// a generic 500 Internal Server Error — technical details are logged but
// never leaked to the client.
func Exception() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		// Nothing to handle if the handler wrote its own response or produced
		// no errors.
		if len(c.Errors) == 0 {
			return
		}

		// Use only the last error (the one closest to the root cause).
		err := c.Errors.Last().Err

		if appErr, ok := err.(*apperror.AppError); ok {
			// Known application error — send the structured response.
			// Log 5xx errors with full context; 4xx at a lower level.
			if appErr.StatusCode >= http.StatusInternalServerError {
				logger.L().Error("internal error",
					zap.String("path", c.FullPath()),
					zap.String("method", c.Request.Method),
					zap.String("code", string(appErr.Code)),
					zap.Error(err),
				)
			} else {
				logger.L().Warn("client error",
					zap.String("path", c.FullPath()),
					zap.String("method", c.Request.Method),
					zap.String("code", string(appErr.Code)),
					zap.String("message", appErr.Message),
				)
			}
			response.Err(c, appErr)
			return
		}

		// Unexpected / untyped error — log and return a safe 500.
		logger.L().Error("unhandled error",
			zap.String("path", c.FullPath()),
			zap.String("method", c.Request.Method),
			zap.Error(err),
		)
		response.Err(c, apperror.NewInternal("an unexpected error occurred"))
	}
}

// Recovery wraps Gin's built-in panic recovery with structured zap logging.
// It must be registered BEFORE Exception so that panics are converted to
// errors that Exception can format consistently.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if rec := recover(); rec != nil {
				// Capture full stack trace for diagnostics.
				stack := debug.Stack()

				logger.L().Error("panic recovered",
					zap.String("path", c.FullPath()),
					zap.String("method", c.Request.Method),
					zap.Any("panic", rec),
					zap.ByteString("stack", stack),
				)

				_ = c.Error(apperror.NewInternal(fmt.Sprintf("internal server error: %v", rec)))
				c.Abort()
			}
		}()
		c.Next()
	}
}

// RequestLogger logs every inbound request and its response status using zap.
func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Next()

		logger.L().Info("request",
			zap.String("method", c.Request.Method),
			zap.String("path", c.Request.URL.Path),
			zap.String("query", c.Request.URL.RawQuery),
			zap.Int("status", c.Writer.Status()),
			zap.String("client_ip", c.ClientIP()),
			zap.Int("bytes_written", c.Writer.Size()),
		)
	}
}
