package router

import (
	"zpay/internal/handler"
	"zpay/internal/middleware"
	"zpay/internal/model"

	"github.com/gin-gonic/gin"
)

func SetupRouter(app *model.App) *gin.Engine {
	router := gin.Default()

	// Initilize middleware
	middleware := middleware.NewAuthHndler(app)

	// Initialize handlers
	userHandler := handler.NewUserHandler(app)
	transactionHandler := handler.NewTranactionHandler(app)

	// Initialize group
	root := router.Group("/")

	// public endpoint group
	public := root.Group("/")
	{
		public.GET("/", handler.Public)
		public.POST("/signup", userHandler.CreateUser)
		public.POST("/login", userHandler.LoginUser)
	}

	// authenticated endpoint group
	auth := root.Group("/")
	auth.Use(middleware.AuthMiddleware())
	{
		auth.POST("/payment", transactionHandler.ProcessTransaction)
	}

	return router
}
