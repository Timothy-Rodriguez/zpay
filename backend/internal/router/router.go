package router

import (
	"zpay/internal/handler"
	"zpay/internal/middleware"
	"zpay/internal/model"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/contrib/instrumentation/github.com/gin-gonic/gin/otelgin"
)

func SetupRouter(app *model.App) *gin.Engine {
	router := gin.Default()

	// Initilize middleware
	authMiddleware := middleware.NewAuthHndler(app)
	loggingMiddleware := middleware.NewLoggingMiddleware(app)
	corsMiddleware := middleware.NewCORSHandler(app, middleware.DefaultCORSConfig())

	// Initialize handlers
	userHandler := handler.NewUserHandler(app)
	transactionHandler := handler.NewTranactionHandler(app)

	// CORS must run before any handler so preflight OPTIONS requests are
	// answered without authentication.
	router.Use(corsMiddleware.CORS())

	// Use logging middleware for all endpoints
	router.Use(loggingMiddleware.Logger())

	// Use tracing here so all HTTP request has root span
	router.Use(otelgin.Middleware("zpay-backend"))

	// Initialize group
	root := router.Group("/")

	// public endpoint group
	public := root.Group("/")
	{
		// exposed prometheus metrics
		public.GET("/metrics", gin.WrapH(promhttp.Handler()))

		public.GET("/", handler.Public)
		public.POST("/signup", userHandler.CreateUser)
		public.POST("/login", userHandler.LoginUser)
		public.GET("/refresh", userHandler.RefreshToken)
	}

	// authenticated endpoint group
	auth := root.Group("/")
	auth.Use(authMiddleware.AuthMiddleware())
	{
		auth.GET("/get-balance", transactionHandler.GetBalance)
		auth.GET("/get-transactions", transactionHandler.GetTransactions)
		auth.POST("/payment", transactionHandler.ProcessTransaction)
		auth.POST("/logout", userHandler.LogoutUser)
	}

	return router
}
