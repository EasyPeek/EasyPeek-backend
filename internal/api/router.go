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
	rssHandler := NewRSSHandler()

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
			events.GET("/:id/news", eventHandler.GetNewsByEventID)
			events.GET("/:id/stats", eventHandler.GetEventStats)
			events.GET("/status/:status", eventHandler.GetEventsByStatus)
			events.POST("/:id/view", eventHandler.IncrementViewCount)
			events.POST("/:id/share", eventHandler.ShareEvent)

			// 需要身份验证的路由
			authEvents := events.Group("")
			authEvents.Use(middleware.AuthMiddleware())
			{
				authEvents.POST("", eventHandler.CreateEvent)
				authEvents.PUT("/:id", eventHandler.UpdateEvent)
				authEvents.DELETE("/:id", eventHandler.DeleteEvent)
				authEvents.POST("/:id/like", eventHandler.LikeEvent)
				authEvents.POST("/:id/comment", eventHandler.AddComment)
			}

			// 管理员专用路由
			adminEvents := events.Group("")
			adminEvents.Use(middleware.AuthMiddleware())
			adminEvents.Use(middleware.AdminMiddleware())
			{
				adminEvents.PUT("/:id/tags", eventHandler.UpdateEventTags)
				adminEvents.POST("/generate", eventHandler.GenerateEventsFromNews)
			}

			// 系统内部路由（需要特殊权限）
			systemEvents := events.Group("")
			systemEvents.Use(middleware.AuthMiddleware())
			// TODO: 添加系统权限中间件
			{
				systemEvents.PUT("/:id/hotness", eventHandler.UpdateEventHotness)
			}
		}

		// RSS routes
		rss := v1.Group("/rss")
		{
			// 公开路由
			rss.GET("/news", rssHandler.GetNews)
			rss.GET("/news/hot", rssHandler.GetHotNews)
			rss.GET("/news/latest", rssHandler.GetLatestNews)
			rss.GET("/news/category/:category", rssHandler.GetNewsByCategory)
			rss.GET("/news/:id", rssHandler.GetNewsItem)

			// 管理员路由
			adminRSS := rss.Group("")
			adminRSS.Use(middleware.AuthMiddleware())
			adminRSS.Use(middleware.AdminMiddleware())
			{
				adminRSS.GET("/sources", rssHandler.GetRSSSources)
				adminRSS.POST("/sources", rssHandler.CreateRSSSource)
				adminRSS.PUT("/sources/:id", rssHandler.UpdateRSSSource)
				adminRSS.DELETE("/sources/:id", rssHandler.DeleteRSSSource)
				adminRSS.POST("/sources/:id/fetch", rssHandler.FetchRSSFeed)
				adminRSS.POST("/fetch-all", rssHandler.FetchAllRSSFeeds)
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
