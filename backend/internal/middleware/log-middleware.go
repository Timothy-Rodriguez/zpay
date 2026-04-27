package middleware

import (
	"strconv"
	"time"
	"zpay/internal/constants"
	"zpay/internal/model"
	"zpay/internal/pkg"

	"github.com/gin-gonic/gin"
)

type LoggingMiddleware struct {
	App *model.App
}

func NewLoggingMiddleware(app *model.App) *LoggingMiddleware {
	return &LoggingMiddleware{App: app}
}

func (m *LoggingMiddleware) Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		// prometheus in-flight requests
		pkg.HTTPInFlightRequests.Inc()
		defer pkg.HTTPInFlightRequests.Dec()

		requestID := c.GetHeader("X-Request-ID")
		if requestID != "" {
			c.Set("request_id", requestID)
			c.Writer.Header().Set("X-Request-ID", requestID)
		}

		c.Next()

		path := c.FullPath()
		if path == "" {
			path = c.Request.URL.Path
		}

		// prometheus total requests & duration
		statusCode := c.Writer.Status()
		durationSeconds := time.Since(start).Seconds()

		pkg.HTTPRequestsTotal.WithLabelValues(
			c.Request.Method,
			path,
			strconv.Itoa(statusCode),
		).Inc()

		pkg.HTTPRequestDurationSeconds.WithLabelValues(
			c.Request.Method,
			path,
		).Observe(durationSeconds)

		fields := []interface{}{
			"http_method", c.Request.Method,
			"http_path", path,
			"http_status", c.Writer.Status(),
			"latency_ms", time.Since(start).Milliseconds(),
			"client_ip", c.ClientIP(),
		}

		if requestID != "" {
			fields = append(fields, "request_id", requestID)
		}

		if email, ok := c.Get(constants.ClaimsEmail); ok {
			fields = append(fields, "user_email", email)
		}

		if len(c.Errors) > 0 {
			fields = append(fields, "errors", c.Errors.String())
			m.App.Logger.Error("request_completed", fields...)
			return
		}

		m.App.Logger.Info("request_completed", fields...)
	}
}
