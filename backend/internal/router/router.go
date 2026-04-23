package router

import (
	"zpay/internal/handler"
	"zpay/internal/model"

	"github.com/gin-gonic/gin"
)

func SetupRouter(app *model.App) *gin.Engine {
	router := gin.Default()

	// Initialize handlers
	userHandler := handler.NewUserHandler(app)

	// Initialize group
	root := router.Group("/")

	// public endpoint group
	public := root.Group("/")
	{
		public.GET("/", handler.Public)
		public.POST("/signup", userHandler.CreateUser)
	}

	return router
}
