package api

import (
	"net/http"
	"strconv"

	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/services"
	"github.com/gin-gonic/gin"
)

type MessageHandler struct {
	messageService *services.MessageService
}

// NewMessageHandler 创建消息处理器实例
func NewMessageHandler() *MessageHandler {
	return &MessageHandler{
		messageService: services.NewMessageService(),
	}
}

// GetMessages 获取用户消息列表（包含未读数量）
// @Summary 获取用户消息列表和未读数量
// @Description 获取当前用户的消息列表，支持分页和类型筛选，同时返回未读消息数量
// @Tags messages
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Param type query string false "消息类型" Enums(system,like,comment,follow,news_update,event_update)
// @Success 200 {object} map[string]interface{} "success"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Failure 401 {object} map[string]interface{} "unauthorized"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/messages [get]
func (h *MessageHandler) GetMessages(c *gin.Context) {
	// 从JWT中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// 获取查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	msgType := c.Query("type")

	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 获取消息列表
	messages, total, err := h.messageService.GetUserMessages(userID.(uint), page, pageSize, msgType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// 获取未读消息数量
	unreadCount, err := h.messageService.GetUnreadCount(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"messages":     messages,
		"total":        total,
		"unread_count": unreadCount,
		"page":         page,
		"page_size":    pageSize,
		"total_pages":  (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// GetUnreadCount 获取未读消息数量（已废弃，请使用 GetMessages 接口）
// @Summary 获取未读消息数量（已废弃）
// @Description 此接口已废弃，请使用 GET /api/v1/messages 接口获取消息列表和未读数量
// @Tags messages
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "success"
// @Failure 401 {object} map[string]interface{} "unauthorized"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/messages/unread-count [get]
// @deprecated
func (h *MessageHandler) GetUnreadCount(c *gin.Context) {
	// 从JWT中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// 获取未读消息数量
	count, err := h.messageService.GetUnreadCount(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data": gin.H{
			"unread_count": count,
		},
	})
}

// MarkAsRead 标记消息为已读
// @Summary 标记消息为已读
// @Description 标记指定消息为已读状态
// @Tags messages
// @Accept json
// @Produce json
// @Param id path int true "消息ID"
// @Success 200 {object} map[string]interface{} "success"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Failure 401 {object} map[string]interface{} "unauthorized"
// @Failure 404 {object} map[string]interface{} "message not found"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/messages/{id}/read [put]
func (h *MessageHandler) MarkAsRead(c *gin.Context) {
	// 从JWT中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// 获取消息ID
	messageIDStr := c.Param("id")
	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	// 标记消息为已读
	err = h.messageService.MarkAsRead(userID.(uint), uint(messageID))
	if err != nil {
		if err.Error() == "message not found or access denied" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Message marked as read",
	})
}

// MarkAllAsRead 标记所有消息为已读
// @Summary 标记所有消息为已读
// @Description 标记当前用户的所有未读消息为已读状态
// @Tags messages
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "success"
// @Failure 401 {object} map[string]interface{} "unauthorized"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/messages/read-all [put]
func (h *MessageHandler) MarkAllAsRead(c *gin.Context) {
	// 从JWT中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// 标记所有消息为已读
	err := h.messageService.MarkAllAsRead(userID.(uint))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "All messages marked as read",
	})
}

// DeleteMessage 删除消息
// @Summary 删除消息
// @Description 删除指定的消息
// @Tags messages
// @Accept json
// @Produce json
// @Param id path int true "消息ID"
// @Success 200 {object} map[string]interface{} "success"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Failure 401 {object} map[string]interface{} "unauthorized"
// @Failure 404 {object} map[string]interface{} "message not found"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/messages/{id} [delete]
func (h *MessageHandler) DeleteMessage(c *gin.Context) {
	// 从JWT中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	// 获取消息ID
	messageIDStr := c.Param("id")
	messageID, err := strconv.ParseUint(messageIDStr, 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid message ID"})
		return
	}

	// 删除消息
	err = h.messageService.DeleteMessage(userID.(uint), uint(messageID))
	if err != nil {
		if err.Error() == "message not found or access denied" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Message not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Message deleted successfully",
	})
}

// CreateMessage 创建消息（管理员功能）
// @Summary 创建消息
// @Description 管理员创建系统消息
// @Tags messages
// @Accept json
// @Produce json
// @Param request body models.CreateMessageRequest true "消息创建请求"
// @Success 201 {object} map[string]interface{} "success"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Failure 401 {object} map[string]interface{} "unauthorized"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/admin/messages [post]
func (h *MessageHandler) CreateMessage(c *gin.Context) {
	// 从JWT中获取发送者ID
	senderID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var req models.CreateMessageRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// 创建消息
	senderIDPtr := new(uint)
	*senderIDPtr = senderID.(uint)
	err := h.messageService.CreateMessage(
		req.UserID,
		req.Type,
		req.Title,
		req.Content,
		req.RelatedType,
		req.RelatedID,
		senderIDPtr,
	)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"success": true,
		"message": "Message created successfully",
	})
}

// GetFollowedEventsLatestNews 获取用户关注事件的最新新闻
// @Summary 获取用户关注事件的最新新闻
// @Description 获取当前用户关注的所有事件的最新一篇新闻
// @Tags messages
// @Accept json
// @Produce json
// @Param limit query int false "限制返回数量" default(20)
// @Success 200 {object} map[string]interface{} "success"
// @Failure 401 {object} map[string]interface{} "unauthorized"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/messages/followed-events-news [get]
func (h *MessageHandler) GetFollowedEventsLatestNews(c *gin.Context) {
	// 从JWT中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "User not authenticated",
			"data":    nil,
		})
		return
	}

	// 获取查询参数
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))

	// 参数验证
	if limit < 1 {
		limit = 20
	}
	if limit > 50 {
		limit = 50
	}

	// 获取关注事件的最新新闻
	result, err := h.messageService.GetFollowedEventsLatestNews(userID.(uint), limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data":    result,
	})
}

// GetFollowedEventsRecentNews 获取用户关注事件的最近新闻
// @Summary 获取用户关注事件的最近新闻
// @Description 获取用户关注事件在指定时间范围内的最新新闻，用于个人中心通知显示
// @Tags messages
// @Accept json
// @Produce json
// @Param hours query int false "获取几小时内的新闻" default(24)
// @Param create_notifications query bool false "是否为新闻创建通知消息" default(false)
// @Success 200 {object} map[string]interface{} "success"
// @Failure 401 {object} map[string]interface{} "unauthorized"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/messages/followed-events-recent-news [get]
func (h *MessageHandler) GetFollowedEventsRecentNews(c *gin.Context) {
	// 从JWT中获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{
			"code":    401,
			"message": "User not authenticated",
			"data":    nil,
		})
		return
	}

	// 获取查询参数
	hours, _ := strconv.Atoi(c.DefaultQuery("hours", "24"))
	createNotifications := c.DefaultQuery("create_notifications", "false") == "true"

	// 参数验证
	if hours < 1 {
		hours = 24
	}
	if hours > 168 { // 最多7天
		hours = 168
	}

	// 获取关注事件的最近新闻
	recentNews, err := h.messageService.GetFollowedEventsRecentNews(userID.(uint), hours)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    500,
			"message": err.Error(),
			"data":    nil,
		})
		return
	}

	// 如果需要，创建通知消息
	if createNotifications && len(recentNews) > 0 {
		err = h.messageService.CreateNewsUpdateNotifications(userID.(uint), hours)
		if err != nil {
			// 创建通知失败不影响数据返回，只记录日志
			c.Header("X-Notification-Warning", "Failed to create notifications: "+err.Error())
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    200,
		"message": "success",
		"data": gin.H{
			"recent_news":           recentNews,
			"count":                 len(recentNews),
			"hours_range":           hours,
			"notifications_created": createNotifications,
		},
	})
}
