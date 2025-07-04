package api

import (
	"net/http"
	"strconv"

	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/services"
	"github.com/EasyPeek/EasyPeek-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

type EventHandler struct {
	eventService *services.EventService
	newsService  *services.NewsService
}

func NewEventHandler() *EventHandler {
	return &EventHandler{
		eventService: services.NewEventService(),
		newsService:  services.NewNewsService(),
	}
}

// GetEvents 获取事件列表
// @Summary 获取事件列表
// @Description 获取事件列表，支持分页、状态筛选、分类筛选、搜索和排序
// @Tags events
// @Produce json
// @Param status query string false "事件状态" Enums(进行中, 已结束)
// @Param category query string false "事件分类"
// @Param search query string false "搜索关键词"
// @Param sort_by query string false "排序方式" Enums(time, hotness, views)
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(10)
// @Success 200 {object} utils.Response{data=models.EventListResponse}
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/events [get]
func (h *EventHandler) GetEvents(c *gin.Context) {
	var query models.EventQueryRequest

	// 绑定查询参数
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.BadRequest(c, "Invalid query parameters")
		return
	}

	// 获取事件列表
	events, err := h.eventService.GetEvents(&query)
	if err != nil {
		utils.InternalServerError(c, "Failed to get events")
		return
	}

	utils.Success(c, events)
}

// GetEvent 根据ID获取事件
// @Summary 根据ID获取事件
// @Description 根据ID获取单个事件详情（会增加浏览量并更新热度）
// @Tags events
// @Produce json
// @Param id path int true "事件ID"
// @Success 200 {object} utils.Response{data=models.EventResponse}
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/events/{id} [get]
func (h *EventHandler) GetEvent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid event ID")
		return
	}

	event, err := h.eventService.ViewEvent(uint(id))
	if err != nil {
		if err.Error() == "event not found" {
			utils.NotFound(c, "Event not found")
			return
		}
		utils.InternalServerError(c, "Failed to get event")
		return
	}

	utils.Success(c, event)
}

// CreateEvent 创建事件
// @Summary 创建事件
// @Description 创建新的事件
// @Tags events
// @Accept json
// @Produce json
// @Param event body models.CreateEventRequest true "事件信息"
// @Success 201 {object} utils.Response{data=models.EventResponse}
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Security BearerAuth
// @Router /api/v1/events [post]
func (h *EventHandler) CreateEvent(c *gin.Context) {
	var req models.CreateEventRequest

	// 绑定请求体
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request body")
		return
	}

	// 创建事件
	event, err := h.eventService.CreateEvent(&req)
	if err != nil {
		utils.InternalServerError(c, "Failed to create event")
		return
	}

	c.JSON(http.StatusCreated, utils.Response{
		Code:    201,
		Message: "Event created successfully",
		Data:    event,
	})
}

// UpdateEvent 更新事件
// @Summary 更新事件
// @Description 更新指定ID的事件信息
// @Tags events
// @Accept json
// @Produce json
// @Param id path int true "事件ID"
// @Param event body models.UpdateEventRequest true "更新的事件信息"
// @Success 200 {object} utils.Response{data=models.EventResponse}
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Security BearerAuth
// @Router /api/v1/events/{id} [put]
func (h *EventHandler) UpdateEvent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid event ID")
		return
	}

	var req models.UpdateEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request body")
		return
	}

	event, err := h.eventService.UpdateEvent(uint(id), &req)
	if err != nil {
		if err.Error() == "event not found" {
			utils.NotFound(c, "Event not found")
			return
		}
		utils.InternalServerError(c, "Failed to update event")
		return
	}

	utils.Success(c, event)
}

// DeleteEvent 删除事件
// @Summary 删除事件
// @Description 删除指定ID的事件（软删除）
// @Tags events
// @Produce json
// @Param id path int true "事件ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Security BearerAuth
// @Router /api/v1/events/{id} [delete]
func (h *EventHandler) DeleteEvent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid event ID")
		return
	}

	err = h.eventService.DeleteEvent(uint(id))
	if err != nil {
		if err.Error() == "event not found" {
			utils.NotFound(c, "Event not found")
			return
		}
		utils.InternalServerError(c, "Failed to delete event")
		return
	}

	utils.Success(c, gin.H{"message": "Event deleted successfully"})
}

// GetEventsByStatus 根据状态获取事件
// @Summary 根据状态获取事件
// @Description 根据状态获取事件列表
// @Tags events
// @Produce json
// @Param status path string true "事件状态" Enums(进行中, 已结束)
// @Success 200 {object} utils.Response{data=[]models.EventResponse}
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/events/status/{status} [get]
func (h *EventHandler) GetEventsByStatus(c *gin.Context) {
	status := c.Param("status")

	// 验证状态值
	if status != "进行中" && status != "已结束" {
		utils.BadRequest(c, "Invalid status")
		return
	}

	events, err := h.eventService.GetEventsByStatus(status)
	if err != nil {
		utils.InternalServerError(c, "Failed to get events by status")
		return
	}

	utils.Success(c, events)
}

// GetHotEvents 获取热点事件列表
// @Summary 获取热点事件列表
// @Description 获取当前热点事件，按热度排序
// @Tags events
// @Produce json
// @Param limit query int false "限制数量" default(10)
// @Success 200 {object} utils.Response{data=[]models.EventResponse}
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/events/hot [get]
func (h *EventHandler) GetHotEvents(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	events, err := h.eventService.GetHotEvents(limit)
	if err != nil {
		utils.InternalServerError(c, "Failed to get hot events")
		return
	}

	utils.Success(c, events)
}

// GetEventCategories 获取事件分类列表
// @Summary 获取事件分类列表
// @Description 获取所有可用的事件分类
// @Tags events
// @Produce json
// @Success 200 {object} utils.Response{data=[]string}
// @Failure 500 {object} utils.Response
// @Router /api/v1/events/categories [get]
func (h *EventHandler) GetEventCategories(c *gin.Context) {
	categories, err := h.eventService.GetEventCategories()
	if err != nil {
		utils.InternalServerError(c, "Failed to get categories")
		return
	}

	utils.Success(c, categories)
}

// IncrementViewCount 增加事件浏览次数
// @Summary 增加事件浏览次数
// @Description 记录事件被查看，增加浏览次数
// @Tags events
// @Produce json
// @Param id path int true "事件ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/events/{id}/view [post]
func (h *EventHandler) IncrementViewCount(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid event ID")
		return
	}

	err = h.eventService.IncrementViewCount(uint(id))
	if err != nil {
		utils.InternalServerError(c, "Failed to increment view count")
		return
	}

	utils.Success(c, gin.H{"message": "View count incremented"})
}

// GetEventsByCategory 按分类获取事件列表
// @Summary 按分类获取事件列表
// @Description 根据分类获取事件列表，支持分页和排序
// @Tags events
// @Produce json
// @Param category path string true "事件分类"
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(10)
// @Param sort_by query string false "排序方式" Enums(time, hotness, views)
// @Success 200 {object} utils.Response{data=models.EventListResponse}
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/events/category/{category} [get]
func (h *EventHandler) GetEventsByCategory(c *gin.Context) {
	category := c.Param("category")
	if category == "" {
		utils.BadRequest(c, "Category is required")
		return
	}

	var query models.EventQueryRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.BadRequest(c, "Invalid query parameters")
		return
	}

	events, err := h.eventService.GetEventsByCategory(category, &query)
	if err != nil {
		utils.InternalServerError(c, "Failed to get events by category")
		return
	}

	utils.Success(c, events)
}

// GetPopularTags 获取热门标签
// @Summary 获取热门标签
// @Description 获取使用频率最高的标签列表
// @Tags events
// @Produce json
// @Param limit query int false "返回标签数量" default(50)
// @Param min_count query int false "标签使用次数最小值" default(1)
// @Success 200 {object} utils.Response{data=[]models.TagResponse}
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/events/tags [get]
func (h *EventHandler) GetPopularTags(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}

	minCountStr := c.DefaultQuery("min_count", "1")
	minCount, err := strconv.Atoi(minCountStr)
	if err != nil || minCount < 0 {
		minCount = 1
	}

	tags, err := h.eventService.GetPopularTags(limit, minCount)
	if err != nil {
		utils.InternalServerError(c, "Failed to get popular tags")
		return
	}

	utils.Success(c, tags)
}

// GetTrendingEvents 获取趋势事件
// @Summary 获取趋势事件
// @Description 获取上升趋势最快的事件列表
// @Tags events
// @Produce json
// @Param limit query int false "返回事件数量" default(10)
// @Param time_range query string false "趋势计算时间范围" Enums(1h, 6h, 24h, 7d) default(24h)
// @Success 200 {object} utils.Response{data=[]models.TrendingEventResponse}
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/events/trending [get]
func (h *EventHandler) GetTrendingEvents(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	timeRange := c.DefaultQuery("time_range", "24h")
	validRanges := map[string]bool{"1h": true, "6h": true, "24h": true, "7d": true}
	if !validRanges[timeRange] {
		timeRange = "24h"
	}

	events, err := h.eventService.GetTrendingEvents(limit, timeRange)
	if err != nil {
		utils.InternalServerError(c, "Failed to get trending events")
		return
	}

	utils.Success(c, events)
}

// UpdateEventTags 更新事件标签
// @Summary 更新事件标签
// @Description 管理员更新事件标签，支持替换、添加、删除操作
// @Tags events
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "事件ID"
// @Param request body models.UpdateTagsRequest true "标签更新请求"
// @Success 200 {object} utils.Response{data=models.Event}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 403 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/events/{id}/tags [put]
func (h *EventHandler) UpdateEventTags(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid event ID")
		return
	}

	var req models.UpdateTagsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request body")
		return
	}

	// 设置默认操作为替换
	if req.Operation == "" {
		req.Operation = "replace"
	}

	event, err := h.eventService.UpdateEventTags(uint(id), req.Tags, req.Operation)
	if err != nil {
		if err.Error() == "record not found" {
			utils.NotFound(c, "Event not found")
			return
		}
		utils.InternalServerError(c, "Failed to update event tags")
		return
	}

	utils.Success(c, event)
}

// UpdateEventHotness 更新事件热度
// @Summary 更新事件热度
// @Description 系统内部接口，用于更新事件热度分值
// @Tags events
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "事件ID"
// @Param request body models.UpdateHotnessRequest true "热度更新请求"
// @Success 200 {object} utils.Response{data=models.HotnessCalculationResult}
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/events/{id}/hotness [put]
func (h *EventHandler) UpdateEventHotness(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid event ID")
		return
	}

	var req models.UpdateHotnessRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request body")
		return
	}

	// 如果设置了手动分值，直接更新
	if req.HotnessScore != nil {
		err := h.eventService.UpdateHotnessScore(uint(id), *req.HotnessScore)
		if err != nil {
			utils.InternalServerError(c, "Failed to update hotness score")
			return
		}
		utils.Success(c, gin.H{"message": "Hotness score updated"})
		return
	}

	// 否则使用自动计算
	autoCalculate := true
	if req.AutoCalculate != nil {
		autoCalculate = *req.AutoCalculate
	}

	if autoCalculate {
		result, err := h.eventService.CalculateHotness(uint(id), req.Factors)
		if err != nil {
			if err.Error() == "record not found" {
				utils.NotFound(c, "Event not found")
				return
			}
			utils.InternalServerError(c, "Failed to calculate hotness")
			return
		}
		utils.Success(c, result)
	} else {
		utils.BadRequest(c, "Either provide hotness_score or set auto_calculate to true")
	}
}

// LikeEvent 点赞或取消点赞事件
// @Summary 点赞或取消点赞事件
// @Description 用户点赞或取消点赞事件
// @Tags events
// @Security BearerAuth
// @Accept json
// @Produce json
// @Param id path int true "事件ID"
// @Param request body models.LikeActionRequest true "点赞操作请求"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/events/{id}/like [post]
func (h *EventHandler) LikeEvent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid event ID")
		return
	}

	var req models.LikeActionRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request body")
		return
	}

	// 获取用户ID
	userID, exists := c.Get("user_id")
	if !exists {
		utils.Unauthorized(c, "User not found")
		return
	}

	switch req.Action {
	case "like":
		err = h.eventService.LikeEvent(uint(id), userID.(uint))
		if err != nil {
			utils.InternalServerError(c, "Failed to like event")
			return
		}
		utils.Success(c, gin.H{"message": "Event liked successfully"})
	case "unlike":
		err = h.eventService.UnlikeEvent(uint(id), userID.(uint))
		if err != nil {
			utils.InternalServerError(c, "Failed to unlike event")
			return
		}
		utils.Success(c, gin.H{"message": "Event unliked successfully"})
	default:
		utils.BadRequest(c, "Invalid action. Use 'like' or 'unlike'")
	}
}

// ShareEvent 分享事件
// @Summary 分享事件
// @Description 记录事件分享，增加分享计数
// @Tags events
// @Produce json
// @Param id path int true "事件ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/events/{id}/share [post]
func (h *EventHandler) ShareEvent(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid event ID")
		return
	}

	err = h.eventService.IncrementShareCount(uint(id))
	if err != nil {
		utils.InternalServerError(c, "Failed to record share")
		return
	}

	utils.Success(c, gin.H{"message": "Share recorded successfully"})
}

// AddComment 添加评论
// @Summary 添加评论
// @Description 为事件添加评论，增加评论计数
// @Tags events
// @Security BearerAuth
// @Produce json
// @Param id path int true "事件ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 401 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/events/{id}/comment [post]
func (h *EventHandler) AddComment(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid event ID")
		return
	}

	// 这里简化处理，只增加评论计数
	// 实际应用中应该保存评论内容到评论表
	err = h.eventService.IncrementCommentCount(uint(id))
	if err != nil {
		utils.InternalServerError(c, "Failed to add comment")
		return
	}

	utils.Success(c, gin.H{"message": "Comment added successfully"})
}

// GetEventStats 获取事件统计信息
// @Summary 获取事件统计信息
// @Description 获取事件的互动统计信息（浏览、点赞、评论、分享、热度）
// @Tags events
// @Produce json
// @Param id path int true "事件ID"
// @Success 200 {object} utils.Response{data=models.InteractionStatsResponse}
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/events/{id}/stats [get]
func (h *EventHandler) GetEventStats(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid event ID")
		return
	}

	stats, err := h.eventService.GetEventStats(uint(id))
	if err != nil {
		if err.Error() == "record not found" {
			utils.NotFound(c, "Event not found")
			return
		}
		utils.InternalServerError(c, "Failed to get event stats")
		return
	}

	utils.Success(c, stats)
}

// GenerateEventsFromNews 从新闻自动生成事件
// @Summary 从新闻自动生成事件
// @Description 基于现有新闻数据自动生成事件，会自动聚类相似新闻并建立关联
// @Tags events
// @Security BearerAuth
// @Produce json
// @Success 200 {object} utils.Response{data=services.EventGenerationResult}
// @Failure 401 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/events/generate [post]
func (h *EventHandler) GenerateEventsFromNews(c *gin.Context) {
	// 调用事件生成服务
	result, err := h.eventService.GenerateEventsFromNews()
	if err != nil {
		utils.InternalServerError(c, err.Error())
		return
	}

	utils.Success(c, result)
}

// GetNewsByEventID 根据事件ID获取相关新闻
// @Summary 根据事件ID获取相关新闻
// @Description 获取指定事件相关联的所有新闻列表
// @Tags events
// @Produce json
// @Param id path int true "事件ID"
// @Success 200 {object} utils.Response{data=[]models.NewsResponse}
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/events/{id}/news [get]
func (h *EventHandler) GetNewsByEventID(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid event ID")
		return
	}

	// 先检查事件是否存在
	_, err = h.eventService.GetEventByID(uint(id))
	if err != nil {
		if err.Error() == "event not found" {
			utils.NotFound(c, "Event not found")
			return
		}
		utils.InternalServerError(c, "Failed to verify event")
		return
	}

	// 获取相关新闻
	newsList, err := h.newsService.GetNewsByEventID(uint(id))
	if err != nil {
		utils.InternalServerError(c, "Failed to get news by event ID")
		return
	}

	// 转换为响应格式
	var newsResponses []models.NewsResponse
	for _, news := range newsList {
		newsResponses = append(newsResponses, news.ToResponse())
	}

	utils.Success(c, newsResponses)
}

// UpdateEventStats 更新事件统计信息
// @Summary 更新事件统计信息
// @Description 更新指定事件的统计信息，包括新闻数、浏览量、热度等，并重新生成标签
// @Tags events
// @Produce json
// @Param id path int true "事件ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Security BearerAuth
// @Router /api/v1/events/{id}/stats/update [post]
func (h *EventHandler) UpdateEventStats(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid event ID")
		return
	}

	err = h.eventService.UpdateEventStats(uint(id))
	if err != nil {
		utils.InternalServerError(c, "Failed to update event stats: "+err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "Event stats updated successfully"})
}

// UpdateAllEventStats 更新所有事件统计信息
// @Summary 更新所有事件统计信息
// @Description 批量更新所有事件的统计信息，包括新闻数、浏览量、热度等，并重新生成标签
// @Tags events
// @Produce json
// @Success 200 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Security BearerAuth
// @Router /api/v1/events/stats/update-all [post]
func (h *EventHandler) UpdateAllEventStats(c *gin.Context) {
	err := h.eventService.UpdateAllEventStats()
	if err != nil {
		utils.InternalServerError(c, "Failed to update all event stats: "+err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "All event stats updated successfully"})
}

// RefreshEventHotness 刷新事件热度
// @Summary 刷新事件热度
// @Description 刷新指定事件的热度评分，包括更新统计信息和重新计算热度
// @Tags events
// @Produce json
// @Param id path int true "事件ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Security BearerAuth
// @Router /api/v1/events/{id}/hotness/refresh [post]
func (h *EventHandler) RefreshEventHotness(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid event ID")
		return
	}

	err = h.eventService.RefreshEventHotness(uint(id))
	if err != nil {
		utils.InternalServerError(c, "Failed to refresh event hotness: "+err.Error())
		return
	}

	utils.Success(c, gin.H{"message": "Event hotness refreshed successfully"})
}

// BatchUpdateEventStats 批量更新指定事件统计信息
// @Summary 批量更新指定事件统计信息
// @Description 批量更新指定事件的统计信息
// @Tags events
// @Accept json
// @Produce json
// @Param eventIds body []uint true "事件ID列表"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Security BearerAuth
// @Router /api/v1/events/stats/batch-update [post]
func (h *EventHandler) BatchUpdateEventStats(c *gin.Context) {
	var eventIDs []uint
	if err := c.ShouldBindJSON(&eventIDs); err != nil {
		utils.BadRequest(c, "Invalid request body")
		return
	}

	if len(eventIDs) == 0 {
		utils.BadRequest(c, "Event IDs list cannot be empty")
		return
	}

	err := h.eventService.BatchUpdateEventStats(eventIDs)
	if err != nil {
		utils.InternalServerError(c, "Failed to batch update event stats: "+err.Error())
		return
	}

	utils.Success(c, gin.H{
		"message":       "Event stats updated successfully",
		"updated_count": len(eventIDs),
	})
}
