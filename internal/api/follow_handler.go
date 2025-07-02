package api

import (
	"net/http"
	"strconv"

	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/services"
	"github.com/gin-gonic/gin"
)

type FollowHandler struct {
	followService *services.FollowService
}

// NewFollowHandler 创建关注处理器实例
func NewFollowHandler() *FollowHandler {
	return &FollowHandler{
		followService: services.NewFollowService(),
	}
}

// AddFollow 添加关注
// @Summary 添加关注
// @Description 用户添加对事件的关注
// @Tags follows
// @Accept json
// @Produce json
// @Param request body models.AddFollowRequest true "关注请求"
// @Success 201 {object} map[string]interface{} "success"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Failure 401 {object} map[string]interface{} "unauthorized"
// @Failure 409 {object} map[string]interface{} "already following"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/follows [post]
func (h *FollowHandler) AddFollow(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.AddFollowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.followService.AddFollow(userID.(uint), req.EventID)
	if err != nil {
		if err.Error() == "already following" {
			c.JSON(http.StatusConflict, gin.H{"error": "Already following this event"})
			return
		}
		if err.Error() == "event not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Event not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Event followed successfully"})
}

// RemoveFollow 取消关注
// @Summary 取消关注
// @Description 用户取消对事件的关注
// @Tags follows
// @Accept json
// @Produce json
// @Param request body models.RemoveFollowRequest true "取消关注请求"
// @Success 200 {object} map[string]interface{} "success"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Failure 401 {object} map[string]interface{} "unauthorized"
// @Failure 404 {object} map[string]interface{} "follow not found"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/follows [delete]
func (h *FollowHandler) RemoveFollow(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.RemoveFollowRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := h.followService.RemoveFollow(userID.(uint), req.EventID)
	if err != nil {
		if err.Error() == "follow not found" {
			c.JSON(http.StatusNotFound, gin.H{"error": "Follow relationship not found"})
			return
		}
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Event unfollowed successfully"})
}

// GetFollows 获取用户关注的事件列表（包含统计信息）
// @Summary 获取关注列表和统计
// @Description 获取用户关注的事件列表，支持分页，同时返回统计信息
// @Tags follows
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} map[string]interface{} "success"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Failure 401 {object} map[string]interface{} "unauthorized"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/follows [get]
func (h *FollowHandler) GetFollows(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	// 获取查询参数
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 获取关注列表和统计信息
	follows, total, err := h.followService.GetUserFollowsWithPagination(userID.(uint), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"follows":     follows,
		"total_count": total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// CheckFollow 检查是否已关注某个事件
// @Summary 检查是否已关注
// @Description 检查用户是否已关注指定事件
// @Tags follows
// @Accept json
// @Produce json
// @Param event_id query uint true "事件ID"
// @Success 200 {object} map[string]interface{} "success"
// @Failure 400 {object} map[string]interface{} "bad request"
// @Failure 401 {object} map[string]interface{} "unauthorized"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/follows/check [get]
func (h *FollowHandler) CheckFollow(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Unauthorized"})
		return
	}

	var req models.CheckFollowRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	isFollowing, err := h.followService.CheckFollow(userID.(uint), req.EventID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"is_following": isFollowing})
}

// GetAvailableEvents 获取可关注的事件列表
// @Summary 获取可关注的事件列表
// @Description 获取可关注的事件列表，支持分页
// @Tags follows
// @Accept json
// @Produce json
// @Param page query int false "页码" default(1)
// @Param page_size query int false "每页数量" default(10)
// @Success 200 {object} map[string]interface{} "success"
// @Failure 500 {object} map[string]interface{} "internal server error"
// @Router /api/v1/follows/targets [get]
func (h *FollowHandler) GetAvailableEvents(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))

	// 参数验证
	if page < 1 {
		page = 1
	}
	if pageSize < 1 || pageSize > 100 {
		pageSize = 10
	}

	// 获取事件列表
	events, total, err := h.followService.GetAvailableEvents(page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"events":      events,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}
