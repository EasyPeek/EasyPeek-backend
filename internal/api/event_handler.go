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
}

func NewEventHandler() *EventHandler {
	return &EventHandler{
		eventService: services.NewEventService(),
	}
}

// GetEvents 获取事件列表
// @Summary 获取事件列表
// @Description 获取事件列表，支持分页、状态筛选和搜索
// @Tags events
// @Produce json
// @Param status query string false "事件状态" Enums(进行中, 已结束)
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(10)
// @Param search query string false "搜索关键词"
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
// @Description 根据ID获取单个事件详情
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

	event, err := h.eventService.GetEventByID(uint(id))
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
