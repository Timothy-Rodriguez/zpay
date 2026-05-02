package middleware

import (
	"net/http"
	"strconv"
	"strings"
	"time"
	"zpay/internal/model"

	"github.com/gin-gonic/gin"
)

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	ExposedHeaders   []string
	AllowCredentials bool
	MaxAge           time.Duration
}

func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowedOrigins: []string{
			"http://localhost:5173",
			"http://127.0.0.1:5173",
		},
		AllowedMethods: []string{
			http.MethodGet,
			http.MethodPost,
			http.MethodPut,
			http.MethodPatch,
			http.MethodDelete,
			http.MethodOptions,
		},
		AllowedHeaders: []string{
			"Origin",
			"Content-Type",
			"Accept",
			"Authorization",
			"X-IDEMPOTENCY-KEY",
		},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}
}

type CORSHandler struct {
	App    *model.App
	Config CORSConfig
}

func NewCORSHandler(app *model.App, cfg CORSConfig) *CORSHandler {
	return &CORSHandler{App: app, Config: cfg}
}

func (h *CORSHandler) CORS() gin.HandlerFunc {
	allowAll := len(h.Config.AllowedOrigins) == 1 && h.Config.AllowedOrigins[0] == "*"
	allowed := make(map[string]struct{}, len(h.Config.AllowedOrigins))
	for _, o := range h.Config.AllowedOrigins {
		allowed[strings.ToLower(strings.TrimSpace(o))] = struct{}{}
	}

	allowMethods := strings.Join(h.Config.AllowedMethods, ", ")
	allowHeaders := strings.Join(h.Config.AllowedHeaders, ", ")
	maxAge := strconv.Itoa(int(h.Config.MaxAge.Seconds()))

	return func(c *gin.Context) {
		origin := c.GetHeader("Origin")

		// Always advertise that responses vary by Origin so caches behave.
		c.Writer.Header().Add("Vary", "Origin")

		if origin != "" {
			if allowAll && !h.Config.AllowCredentials {
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			} else if _, ok := allowed[strings.ToLower(origin)]; ok || allowAll {
				// Echo the requesting origin so credentials can be used.
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
				if h.Config.AllowCredentials {
					c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
				}
			}
		}

		// Preflight handling.
		if c.Request.Method == http.MethodOptions {
			c.Writer.Header().Set("Access-Control-Allow-Methods", allowMethods)
			if reqHeaders := c.GetHeader("Access-Control-Request-Headers"); reqHeaders != "" {
				c.Writer.Header().Set("Access-Control-Allow-Headers", reqHeaders)
			} else if allowHeaders != "" {
				c.Writer.Header().Set("Access-Control-Allow-Headers", allowHeaders)
			}
			c.Writer.Header().Set("Access-Control-Max-Age", maxAge)
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}
