package api

import (
	"github.com/EasyPeek/EasyPeek-backend/internal/middleware"
	"github.com/gin-gonic/gin"
)

func SetupRoutes() *gin.Engine {
	r := gin.Default()

	// add cors middleware
	r.Use(middleware.CORSMiddleware())

	// health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "ok",
			"message": "EasyPeek backend is running",
		})
	})

	// initialize handler
	userHandler := NewUserHandler()
	newsHandler := NewNewsHandler()

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// public routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", userHandler.Register)
			auth.POST("/login", userHandler.Login)
			// auth.POST(refresh)
			// auth.POST(logout)
		}

		// user routes
		user := v1.Group("/user")
		user.Use(middleware.AuthMiddleware())
		{
			user.GET("/profile", userHandler.GetProfile)
			user.PUT("/profile", userHandler.UpdateProfile)
			user.POST("/change-password", userHandler.ChangePassword)
		}

		// admin routes
		admin := v1.Group("/admin")
		admin.Use(middleware.AuthMiddleware())
		admin.Use(middleware.AdminMiddleware())
		{
			admin.GET("/users", userHandler.GetUsers)
			admin.GET("/users/:id", userHandler.GetUser)
			admin.DELETE("/users/:id", userHandler.DeleteUser)
		}

		// news routes
		news := v1.Group("/news")
		{
			news.POST("", middleware.AuthMiddleware(), newsHandler.CreateNews)
			news.GET("/:id", newsHandler.GetNewsByID)
			news.GET("", newsHandler.GetAllNews)
			news.PUT("/:id", middleware.AuthMiddleware(), newsHandler.UpdateNews)
			news.DELETE("/:id", middleware.AuthMiddleware(), newsHandler.DeleteNews)
			news.GET("/search", newsHandler.SearchNews)
		}
	}

	return r
}
