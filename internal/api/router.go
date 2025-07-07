package api

import (
	"github.com/EasyPeek/EasyPeek-backend/internal/middleware"
	"github.com/EasyPeek/EasyPeek-backend/internal/services"
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
	adminHandler := NewAdminHandler()
	userHandler := NewUserHandler()
	eventHandler := NewEventHandler()
	rssHandler := NewRSSHandler()
	newsHandler := NewNewsHandler()
	commentHandler := NewCommentHandler()
	messageHandler := NewMessageHandler()
	followHandler := NewFollowHandler()

	// initialize services for AI handler
	newsService := services.NewNewsService()
	aiHandler := NewAIHandler(newsService)

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// public routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", userHandler.Register)
			auth.POST("/login", userHandler.Login)
			auth.POST("/logout", userHandler.Logout)

			auth.POST("/admin-login", adminHandler.AdminLogin)
			auth.POST("/admin-logout", adminHandler.AdminLogout)
		}

		// user routes
		user := v1.Group("/user")
		user.Use(middleware.AuthMiddleware())
		{

			user.GET("/profile", userHandler.GetProfile)
			user.PUT("/profile", userHandler.UpdateProfile)
			user.POST("/change-password", userHandler.ChangePassword)
			user.DELETE("/me", userHandler.DeleteSelf)
		}

		// message routes
		messages := v1.Group("/messages")
		messages.Use(middleware.AuthMiddleware())
		{
			messages.GET("", messageHandler.GetMessages)                                             // 获取消息列表
			messages.GET("/unread-count", messageHandler.GetUnreadCount)                             // 获取未读消息数量
			messages.GET("/followed-events-news", messageHandler.GetFollowedEventsLatestNews)        // 获取关注事件的最新新闻
			messages.GET("/followed-events-recent-news", messageHandler.GetFollowedEventsRecentNews) // 获取关注事件的最近新闻
			messages.PUT("/:id/read", messageHandler.MarkAsRead)                                     // 标记消息已读
			messages.PUT("/read-all", messageHandler.MarkAllAsRead)                                  // 标记全部已读
			messages.DELETE("/:id", messageHandler.DeleteMessage)                                    // 删除消息
		}

		// follow routes
		follows := v1.Group("/follows")
		follows.Use(middleware.AuthMiddleware())
		{
			follows.POST("", followHandler.AddFollow)        // 添加关注
			follows.DELETE("", followHandler.RemoveFollow)   // 取消关注
			follows.GET("", followHandler.GetFollows)        // 获取关注列表（包含统计信息）
			follows.GET("/check", followHandler.CheckFollow) // 检查是否已关注
			// follows.GET("/stats", followHandler.GetFollowStats)   // 已废弃：获取关注统计，请使用 GET /follows
			follows.GET("/events", followHandler.GetAvailableEvents) // 获取可关注的事件列表
		}

		// news routes
		news := v1.Group("/news")
		{

			// 公开路由 - 前端可以直接访问
			news.GET("", newsHandler.GetAllNews)                           // 获取所有新闻列表（带分页）
			news.GET("/hot", newsHandler.GetHotNews)                       // 获取热门新闻
			news.GET("/latest", newsHandler.GetLatestNews)                 // 获取最新新闻
			news.GET("/category/:category", newsHandler.GetNewsByCategory) // 按分类获取新闻
			news.GET("/:id", newsHandler.GetNewsByID)                      // 根据ID获取单条新闻
			news.GET("/search", newsHandler.SearchNews)                    // 搜索新闻
			news.POST("/:id/view", newsHandler.IncrementNewsView)          // 增加浏览量

			// 需要身份验证的路由
			authNews := news.Group("")
			authNews.Use(middleware.AuthMiddleware())
			{
				authNews.POST("", newsHandler.CreateNews)                // 创建新闻
				authNews.PUT("/:id", newsHandler.UpdateNews)             // 更新新闻
				authNews.DELETE("/:id", newsHandler.DeleteNews)          // 删除新闻
				authNews.POST("/:id/like", newsHandler.LikeNews)         // 点赞/取消点赞新闻
				authNews.GET("/:id/like", newsHandler.GetNewsLikeStatus) // 获取点赞状态
			}
		}

		// comment routes
		comments := v1.Group("/comments")
		{
			// 公开路由
			comments.GET("/:id", commentHandler.GetCommentByID)                // 根据ID获取单条评论
			comments.POST("/anonymous", commentHandler.CreateAnonymousComment) // 创建匿名评论

			// 支持可选认证的路由（已登录用户可以获取个人点赞状态）
			comments.GET("/news/:news_id", middleware.OptionalAuthMiddleware(), commentHandler.GetCommentsByNewsID) // 根据新闻ID获取评论列表
			comments.GET("/user/:user_id", middleware.OptionalAuthMiddleware(), commentHandler.GetCommentsByUserID) // 根据用户ID获取评论列表

			// 需要身份验证的路由
			authComments := comments.Group("")
			authComments.Use(middleware.AuthMiddleware())
			{
				authComments.POST("", commentHandler.CreateComment)            // 创建评论
				authComments.POST("/reply", commentHandler.ReplyToComment)     // 回复评论
				authComments.DELETE("/:id", commentHandler.DeleteComment)      // 删除评论
				authComments.POST("/:id/like", commentHandler.LikeComment)     // 点赞评论
				authComments.DELETE("/:id/like", commentHandler.UnlikeComment) // 取消点赞评论
			}
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
			adminEvents.Use(middleware.RoleMiddleware(middleware.RoleAdmin))
			{
				adminEvents.PUT("/:id/tags", eventHandler.UpdateEventTags)
				adminEvents.POST("/generate", eventHandler.GenerateEventsFromNews)
				adminEvents.POST("/:id/stats/update", eventHandler.UpdateEventStats)        // 更新单个事件统计信息
				adminEvents.POST("/stats/update-all", eventHandler.UpdateAllEventStats)     // 更新所有事件统计信息
				adminEvents.POST("/:id/hotness/refresh", eventHandler.RefreshEventHotness)  // 刷新事件热度
				adminEvents.POST("/stats/batch-update", eventHandler.BatchUpdateEventStats) // 批量更新事件统计信息
			}

			// 系统内部路由（需要系统权限或管理员权限）
			systemEvents := events.Group("")
			systemEvents.Use(middleware.AuthMiddleware())
			systemEvents.Use(middleware.RequireSystemOrAdmin())
			{
				systemEvents.PUT("/:id/hotness", eventHandler.UpdateEventHotness)
			}
		}

		// RSS routes
		rss := v1.Group("/rss")
		{
			// 管理员路由
			adminRSS := rss.Group("")
			adminRSS.Use(middleware.AuthMiddleware())
			adminRSS.Use(middleware.RoleMiddleware(middleware.RoleAdmin))
			{
				adminRSS.GET("/sources", rssHandler.GetAllRSSSources)
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
		admin.Use(middleware.AdminAuthMiddleware())
		{
			// System Stats
			admin.GET("/stats", adminHandler.GetSystemStats)

			// User Management
			users := admin.Group("/users")
			{
				users.GET("", adminHandler.GetAllUsers)

				// get user by username or email
				users.GET("/:id", adminHandler.GetUserByID)   // get user by ID
				users.PUT("/:id", adminHandler.UpdateUser)    // 更新用户信息
				users.DELETE("/:id", adminHandler.DeleteUser) // delete user
			}

			// 事件管理
			events := admin.Group("/events")
			{
				events.GET("", adminHandler.GetAllEvents)       // 获取所有事件
				events.POST("", adminHandler.CreateEvent)       // 创建事件
				events.PUT("/:id", adminHandler.UpdateEvent)    // 更新事件
				events.DELETE("/:id", adminHandler.DeleteEvent) // 删除事件
			}

			// 新闻管理
			news := admin.Group("/news")
			{
				news.GET("", adminHandler.GetAllNews)        // 获取所有新闻
				news.POST("", adminHandler.CreateNews)       // 创建新闻
				news.PUT("/:id", adminHandler.UpdateNews)    // 更新新闻
				news.DELETE("/:id", adminHandler.DeleteNews) // 删除新闻
			}

			// 评论管理
			comments := admin.Group("/comments")
			{
				comments.GET("", commentHandler.GetAllComments)            // 获取所有评论
				comments.DELETE("/:id", commentHandler.AdminDeleteComment) // 管理员删除评论（硬删除）
			}

			// RSS源管理
			rssAdmin := admin.Group("/rss")
			{
				rssAdmin.GET("", rssHandler.GetAllRSSSources)                          // 获取所有RSS源
				rssAdmin.POST("", rssHandler.CreateRSSSource)                          // 创建RSS源
				rssAdmin.PUT("/:id", rssHandler.UpdateRSSSource)                       // 更新RSS源
				rssAdmin.DELETE("/:id", rssHandler.DeleteRSSSource)                    // 删除RSS源
				rssAdmin.POST("/:id/fetch", rssHandler.FetchRSSFeed)                   // 手动抓取RSS源
				rssAdmin.POST("/fetch-all", rssHandler.FetchAllRSSFeeds)               // 抓取所有RSS源
				rssAdmin.POST("/batch-analyze", aiHandler.BatchAnalyzeUnprocessedNews) // 批量AI分析未处理新闻
				rssAdmin.GET("/categories", rssHandler.GetRSSCategories)               // 获取RSS分类列表
				rssAdmin.GET("/stats", rssHandler.GetRSSStats)                         // 获取RSS统计信息
			}

			// 消息管理
			messageAdmin := admin.Group("/messages")
			{
				messageAdmin.POST("", messageHandler.CreateMessage) // 创建系统消息
			}
		}

		// AI routes
		ai := v1.Group("/ai")
		{
			// 公开路由
			ai.POST("/analyze", aiHandler.AnalyzeNews)                                   // 分析新闻
			ai.POST("/analyze-event", aiHandler.AnalyzeEvent)                            // 分析事件
			ai.GET("/analysis", aiHandler.GetAnalysis)                                   // 获取分析结果
			ai.POST("/batch-analyze", aiHandler.BatchAnalyzeNews)                        // 批量分析指定新闻
			ai.POST("/batch-analyze-unprocessed", aiHandler.BatchAnalyzeUnprocessedNews) // 批量分析未处理新闻
			ai.GET("/stats", aiHandler.GetAnalysisStats)                                 // 获取分析统计
			ai.POST("/summarize", aiHandler.SummarizeNews)                               // 快速摘要
		}

	}

	return r
}
