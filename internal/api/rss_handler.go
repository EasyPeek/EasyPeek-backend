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

	// 处理前端字段映射
	if updateFreq, exists := c.GetPostForm("update_freq"); exists {
		if freq, err := strconv.Atoi(updateFreq); err == nil {
			req.UpdateFreq = freq
		}
	}

	source, err := h.rssService.CreateRSSSource(&req)
	if err != nil {
		utils.BadRequest(c, err.Error())
		return
	}

	// 转换响应数据，映射字段名
	responseData := map[string]interface{}{
		"id":              source.ID,
		"name":            source.Name,
		"url":             source.URL,
		"category":        source.Category,
		"description":     source.Description,
		"is_active":       source.IsActive,
		"fetch_interval":  source.UpdateFreq, // 映射字段名
		"last_fetch_time": source.LastFetched,
		"fetch_count":     source.FetchCount,
		"error_count":     source.ErrorCount,
		"priority":        source.Priority,
		"language":        source.Language,
		"tags":            source.Tags,
		"created_at":      source.CreatedAt,
		"updated_at":      source.UpdatedAt,
	}

	c.JSON(http.StatusCreated, utils.Response{
		Code:    201,
		Message: "RSS source created successfully",
		Data:    responseData,
	})
}

// GetAllRSSSources 获取RSS源列表
func (h *RSSHandler) GetAllRSSSources(c *gin.Context) {
	pageStr := c.DefaultQuery("page", "1")
	sizeStr := c.DefaultQuery("size", "10")

	page, err := strconv.Atoi(pageStr)
	if err != nil || page < 1 {
		page = 1
	}

	size, err := strconv.Atoi(sizeStr)
	if err != nil || size < 1 || size > 100 {
		size = 10
	}

	sources, total, err := h.rssService.GetAllRSSSources(page, size)
	if err != nil {
		utils.InternalServerError(c, "Failed to get RSS sources")
		return
	}

	var RSSSourceResponse []interface{}
	for _, source := range sources {
		RSSSourceResponse = append(RSSSourceResponse, source.ToResponse())
	}

	utils.SuccessWithPagination(c, RSSSourceResponse, total, page, size)
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

	utils.Success(c, source.ToResponse())
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

// GetRSSCategories 获取RSS源分类列表
func (h *RSSHandler) GetRSSCategories(c *gin.Context) {
	categories, err := h.rssService.GetRSSCategories()
	if err != nil {
		utils.InternalServerError(c, "Failed to get RSS categories")
		return
	}

	utils.Success(c, map[string]interface{}{
		"categories": categories,
	})
}

// GetRSSStats 获取RSS源统计信息
func (h *RSSHandler) GetRSSStats(c *gin.Context) {
	stats, err := h.rssService.GetRSSStats()
	if err != nil {
		utils.InternalServerError(c, "Failed to get RSS statistics")
		return
	}

	utils.Success(c, stats)
}
