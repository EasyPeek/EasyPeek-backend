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
// @Summary 创建RSS源
// @Description 创建新的RSS源配置
// @Tags rss
// @Accept json
// @Produce json
// @Param source body models.CreateRSSSourceRequest true "RSS源信息"
// @Success 201 {object} utils.Response{data=models.RSSSourceResponse}
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Security BearerAuth
// @Router /api/v1/rss/sources [post]
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
// @Summary 获取RSS源列表
// @Description 获取RSS源列表，支持分页和筛选
// @Tags rss
// @Produce json
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(10)
// @Param category query string false "分类筛选"
// @Param is_active query bool false "是否启用筛选"
// @Success 200 {object} utils.Response{data=models.NewsListResponse}
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/rss/sources [get]
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
// @Summary 更新RSS源
// @Description 更新指定ID的RSS源信息
// @Tags rss
// @Accept json
// @Produce json
// @Param id path int true "RSS源ID"
// @Param source body models.UpdateRSSSourceRequest true "更新的RSS源信息"
// @Success 200 {object} utils.Response{data=models.RSSSourceResponse}
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Security BearerAuth
// @Router /api/v1/rss/sources/{id} [put]
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
// @Summary 删除RSS源
// @Description 删除指定ID的RSS源（软删除）
// @Tags rss
// @Produce json
// @Param id path int true "RSS源ID"
// @Success 200 {object} utils.Response
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Security BearerAuth
// @Router /api/v1/rss/sources/{id} [delete]
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
// @Summary 手动抓取RSS源
// @Description 手动触发指定RSS源的内容抓取
// @Tags rss
// @Produce json
// @Param id path int true "RSS源ID"
// @Success 200 {object} utils.Response{data=models.RSSFetchStats}
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Security BearerAuth
// @Router /api/v1/rss/sources/{id}/fetch [post]
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
// @Summary 抓取所有RSS源
// @Description 手动触发所有活跃RSS源的内容抓取
// @Tags rss
// @Produce json
// @Success 200 {object} utils.Response{data=models.RSSFetchResult}
// @Failure 500 {object} utils.Response
// @Security BearerAuth
// @Router /api/v1/rss/fetch-all [post]
func (h *RSSHandler) FetchAllRSSFeeds(c *gin.Context) {
	result, err := h.rssService.FetchAllRSSFeeds()
	if err != nil {
		utils.InternalServerError(c, "Failed to fetch RSS feeds: "+err.Error())
		return
	}

	utils.Success(c, result)
}

// GetNews 获取新闻列表
// @Summary 获取新闻列表
// @Description 获取新闻列表，支持分页、筛选、搜索和排序
// @Tags rss
// @Produce json
// @Param rss_source_id query int false "RSS源ID"
// @Param category query string false "分类筛选"
// @Param status query string false "状态筛选"
// @Param search query string false "搜索关键词"
// @Param sort_by query string false "排序方式" Enums(published_at, hotness, views)
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(10)
// @Param start_date query string false "开始日期 YYYY-MM-DD"
// @Param end_date query string false "结束日期 YYYY-MM-DD"
// @Success 200 {object} utils.Response{data=models.NewsListResponse}
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/rss/news [get]
func (h *RSSHandler) GetNews(c *gin.Context) {
	var query models.NewsQueryRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.BadRequest(c, "Invalid query parameters: "+err.Error())
		return
	}

	news, err := h.rssService.GetNews(&query)
	if err != nil {
		utils.InternalServerError(c, "Failed to get news")
		return
	}

	utils.Success(c, news)
}

// GetNewsItem 获取新闻详情
// @Summary 获取新闻详情
// @Description 根据ID获取单个新闻详情（会增加浏览量）
// @Tags rss
// @Produce json
// @Param id path int true "新闻ID"
// @Success 200 {object} utils.Response{data=models.NewsItemResponse}
// @Failure 400 {object} utils.Response
// @Failure 404 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/rss/news/{id} [get]
func (h *RSSHandler) GetNewsItem(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "Invalid news ID")
		return
	}

	newsItem, err := h.rssService.GetNewsItem(uint(id))
	if err != nil {
		if err.Error() == "news item not found" {
			utils.NotFound(c, "News item not found")
			return
		}
		utils.InternalServerError(c, "Failed to get news item")
		return
	}

	utils.Success(c, newsItem)
}

// GetHotNews 获取热门新闻
// @Summary 获取热门新闻
// @Description 获取当前热门新闻，按热度排序
// @Tags rss
// @Produce json
// @Param limit query int false "限制数量" default(10)
// @Success 200 {object} utils.Response{data=[]models.NewsItemResponse}
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/rss/news/hot [get]
func (h *RSSHandler) GetHotNews(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	query := &models.NewsQueryRequest{
		SortBy: "hotness",
		Page:   1,
		Limit:  limit,
		Status: "published",
	}

	news, err := h.rssService.GetNews(query)
	if err != nil {
		utils.InternalServerError(c, "Failed to get hot news")
		return
	}

	utils.Success(c, news.News)
}

// GetLatestNews 获取最新新闻
// @Summary 获取最新新闻
// @Description 获取最新发布的新闻，按发布时间排序
// @Tags rss
// @Produce json
// @Param limit query int false "限制数量" default(10)
// @Success 200 {object} utils.Response{data=[]models.NewsItemResponse}
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/rss/news/latest [get]
func (h *RSSHandler) GetLatestNews(c *gin.Context) {
	limitStr := c.DefaultQuery("limit", "10")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 10
	}

	query := &models.NewsQueryRequest{
		SortBy: "published_at",
		Page:   1,
		Limit:  limit,
		Status: "published",
	}

	news, err := h.rssService.GetNews(query)
	if err != nil {
		utils.InternalServerError(c, "Failed to get latest news")
		return
	}

	utils.Success(c, news.News)
}

// GetNewsByCategory 按分类获取新闻
// @Summary 按分类获取新闻
// @Description 根据分类获取新闻列表
// @Tags rss
// @Produce json
// @Param category path string true "新闻分类"
// @Param page query int false "页码" default(1)
// @Param limit query int false "每页数量" default(10)
// @Param sort_by query string false "排序方式" Enums(published_at, hotness, views)
// @Success 200 {object} utils.Response{data=models.NewsListResponse}
// @Failure 400 {object} utils.Response
// @Failure 500 {object} utils.Response
// @Router /api/v1/rss/news/category/{category} [get]
func (h *RSSHandler) GetNewsByCategory(c *gin.Context) {
	category := c.Param("category")
	if category == "" {
		utils.BadRequest(c, "Category is required")
		return
	}

	var query models.NewsQueryRequest
	if err := c.ShouldBindQuery(&query); err != nil {
		utils.BadRequest(c, "Invalid query parameters: "+err.Error())
		return
	}

	query.Category = category
	query.Status = "published"

	news, err := h.rssService.GetNews(&query)
	if err != nil {
		utils.InternalServerError(c, "Failed to get news by category")
		return
	}

	utils.Success(c, news)
}