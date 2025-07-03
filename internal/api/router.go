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
	adminHandler := NewAdminHandler()
	userHandler := NewUserHandler()
	eventHandler := NewEventHandler()
	rssHandler := NewRSSHandler()
	newsHandler := NewNewsHandler()
	commentHandler := NewCommentHandler()
	messageHandler := NewMessageHandler()
	followHandler := NewFollowHandler()

	// API v1 routes
	v1 := r.Group("/api/v1")
	{
		// public routes
		auth := v1.Group("/auth")
		{
			auth.POST("/register", userHandler.Register)
			auth.POST("/login", userHandler.Login)

			auth.POST("/admin-login", adminHandler.AdminLogin) // 管理员登录
			// auth.POST("/refresh", userHandler.RefreshToken)  // TODO: 实现token刷新
			// auth.POST("/logout", userHandler.Logout)         // TODO: 实现登出
		}

		// user routes
		user := v1.Group("/user")
		user.Use(middleware.AuthMiddleware())
		{
			user.GET("/profile", userHandler.GetProfile)
			user.PUT("/profile", userHandler.UpdateProfile)
			user.POST("/change-password", userHandler.ChangePassword)
			// 用户自删除账户
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

			// 需要身份验证的路由
			authNews := news.Group("")
			authNews.Use(middleware.AuthMiddleware())
			{
				authNews.POST("", newsHandler.CreateNews)       // 创建新闻
				authNews.PUT("/:id", newsHandler.UpdateNews)    // 更新新闻
				authNews.DELETE("/:id", newsHandler.DeleteNews) // 删除新闻
			}
		}

		// comment routes
		comments := v1.Group("/comments")
		{
			// 公开路由
			comments.GET("/:id", commentHandler.GetCommentByID)                // 根据ID获取单条评论
			comments.GET("/news/:news_id", commentHandler.GetCommentsByNewsID) // 根据新闻ID获取评论列表
			comments.GET("/user/:user_id", commentHandler.GetCommentsByUserID) // 根据用户ID获取评论列表
			comments.POST("/anonymous", commentHandler.CreateAnonymousComment) // 创建匿名评论

			// 需要身份验证的路由
			authComments := comments.Group("")
			authComments.Use(middleware.AuthMiddleware())
			{
				authComments.POST("", commentHandler.CreateComment)            // 创建评论
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
			// 公开路由
			rss.GET("/news", rssHandler.GetNews)
			rss.GET("/news/hot", rssHandler.GetHotNews)
			rss.GET("/news/latest", rssHandler.GetLatestNews)
			rss.GET("/news/category/:category", rssHandler.GetNewsByCategory)
			rss.GET("/news/:id", rssHandler.GetNewsItem)

			// 管理员路由
			adminRSS := rss.Group("")
			adminRSS.Use(middleware.AuthMiddleware())
			adminRSS.Use(middleware.RoleMiddleware(middleware.RoleAdmin))
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
		admin.Use(middleware.AdminAuthMiddleware())
		{
			// system stats
			admin.GET("/stats", adminHandler.GetSystemStats)

			// 用户管理
			users := admin.Group("/users")
			{
				users.GET("", adminHandler.GetAllUsers)          // 获取所有用户（带过滤）
				users.GET("/active", userHandler.GetActiveUsers) // 获取活跃用户（保持兼容）
				users.GET("/:id", adminHandler.GetUserByID)      // 获取指定用户

				users.PUT("/:id", adminHandler.UpdateUser)    // 更新用户信息
				users.DELETE("/:id", adminHandler.DeleteUser) // 管理员删除用户（硬删除）
				// 保留原有的单独角色和状态更新接口
				// users.PUT("/:id/role", userHandler.UpdateUserRole)     // 更新用户角色
				// users.PUT("/:id/status", userHandler.UpdateUserStatus) // 更新用户状态
			}

			// 事件管理
			events := admin.Group("/events")
			{
				events.GET("", adminHandler.GetAllEvents)       // 获取所有事件
				events.PUT("/:id", adminHandler.UpdateEvent)    // 更新事件
				events.DELETE("/:id", adminHandler.DeleteEvent) // 删除事件
			}

			// 新闻管理
			news := admin.Group("/news")
			{
				news.GET("", adminHandler.GetAllNews)        // 获取所有新闻
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
			rssAdmin := admin.Group("/rss-sources")
			{
				rssAdmin.GET("", adminHandler.GetAllRSSSources)            // 获取所有RSS源
				rssAdmin.POST("", adminHandler.CreateRSSSource)            // 创建RSS源
				rssAdmin.PUT("/:id", adminHandler.UpdateRSSSource)         // 更新RSS源
				rssAdmin.DELETE("/:id", adminHandler.DeleteRSSSource)      // 删除RSS源
				rssAdmin.POST("/:id/fetch", adminHandler.FetchRSSFeed)     // 手动抓取RSS源
				rssAdmin.POST("/fetch-all", adminHandler.FetchAllRSSFeeds) // 抓取所有RSS源
			}

			// 消息管理
			messageAdmin := admin.Group("/messages")
			{
				messageAdmin.POST("", messageHandler.CreateMessage) // 创建系统消息
			}
		}

	}

	return r
}
