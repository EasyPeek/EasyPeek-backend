package api

import (
	"net/http"
	"strconv"

	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/services"
	"github.com/EasyPeek/EasyPeek-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

type RSSHandler struct {
	rssService *services.RSSService
}

func NewRSSHandler() *RSSHandler {
	return &RSSHandler{
		rssService: services.NewRSSService(),
	}
}

// CreateRSSSource 创建RSS源
func (h *RSSHandler) CreateRSSSource(c *gin.Context) {
	var req models.CreateRSSSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	source, err := h.rssService.CreateRSSSource(&req)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	c.JSON(http.StatusCreated, utils.Response{
		Code:    201,
		Message: "RSS source created successfully",
		Data:    source,
	})
}

// GetRSSSources 获取RSS源列表
func (h *RSSHandler) GetRSSSources(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	limitStr := c.DefaultQuery("limit", "10")
	category := c.Query("category")
	isActiveStr := c.Query("is_active")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit < 1 || limit > 100 {
		limit = 10
	}

	var isActive *bool
	if isActiveStr != "" {
		if isActiveStr == "true" {
			active := true
			isActive = &active
		} else if isActiveStr == "false" {
			active := false
			isActive = &active
		}
	}

	sources, err := h.rssService.GetRSSSources(page, limit, category, isActive)
	if err != nil {
		utils.InternalServerError(c, "Failed to get RSS sources")
		return
	}

	utils.Success(c, sources)
}

// UpdateRSSSource 更新RSS源
func (h *RSSHandler) UpdateRSSSource(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid RSS source ID")
		return
	}

	var req models.UpdateRSSSourceRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "Invalid request body: "+err.Error())
		return
	}

	source, err := h.rssService.UpdateRSSSource(uint(id), &req)
	if err != nil {
		if err.Error() == "RSS source not found" {
			utils.NotFound(c, "RSS source not found")
			return
		}
		utils.BadRequest(c, err.Error())
		return
	}

	utils.Success(c, source)
}

// DeleteRSSSource 删除RSS源
func (h *RSSHandler) DeleteRSSSource(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid RSS source ID")
		return
	}

	err = h.rssService.DeleteRSSSource(uint(id))
	if err != nil {
		if err.Error() == "RSS source not found" {
			utils.NotFound(c, "RSS source not found")
			return
		}
		utils.InternalServerError(c, "Failed to delete RSS source")
		return
	}

	utils.Success(c, gin.H{"message": "RSS source deleted successfully"})
}

// FetchRSSFeed 手动抓取RSS源
func (h *RSSHandler) FetchRSSFeed(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid RSS source ID")
		return
	}

	stats, err := h.rssService.FetchRSSFeed(uint(id))
	if err != nil {
		utils.InternalServerError(c, "Failed to fetch RSS feed: "+err.Error())
		return
	}

	utils.Success(c, stats)
}

// FetchAllRSSFeeds 抓取所有RSS源
func (h *RSSHandler) FetchAllRSSFeeds(c *gin.Context) {
	result, err := h.rssService.FetchAllRSSFeeds()
	if err != nil {
		utils.InternalServerError(c, "Failed to fetch RSS feeds: "+err.Error())
		return
	}

	utils.Success(c, result)
}
