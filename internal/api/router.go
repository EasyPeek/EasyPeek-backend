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
	eventHandler := NewEventHandler()

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

		// event routes
		events := v1.Group("/events")
		{
			// 公开路由
			events.GET("", eventHandler.GetEvents)
			events.GET("/hot", eventHandler.GetHotEvents)
			events.GET("/trending", eventHandler.GetTrendingEvents)
			events.GET("/categories", eventHandler.GetEventCategories)
			events.GET("/category/:category", eventHandler.GetEventsByCategory)
			events.GET("/tags", eventHandler.GetPopularTags)
			events.GET("/:id", eventHandler.GetEvent)
			events.GET("/status/:status", eventHandler.GetEventsByStatus)
			events.POST("/:id/view", eventHandler.IncrementViewCount)

			// 需要身份验证的路由
			authEvents := events.Group("")
			authEvents.Use(middleware.AuthMiddleware())
			{
				authEvents.POST("", eventHandler.CreateEvent)
				authEvents.PUT("/:id", eventHandler.UpdateEvent)
				authEvents.DELETE("/:id", eventHandler.DeleteEvent)
			}

			// 管理员专用路由
			adminEvents := events.Group("")
			adminEvents.Use(middleware.AuthMiddleware())
			adminEvents.Use(middleware.AdminMiddleware())
			{
				adminEvents.PUT("/:id/tags", eventHandler.UpdateEventTags)
			}

			// 系统内部路由（需要特殊权限）
			systemEvents := events.Group("")
			systemEvents.Use(middleware.AuthMiddleware())
			// TODO: 添加系统权限中间件
			{
				systemEvents.PUT("/:id/hotness", eventHandler.UpdateEventHotness)
			}
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
	}

	return r
}
