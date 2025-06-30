package services

import (
	"errors"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/utils"
	"github.com/mmcdole/gofeed"
	"gorm.io/gorm"
)

type RSSService struct {
	db     *gorm.DB
	parser *gofeed.Parser
}

func NewRSSService() *RSSService {
	return &RSSService{
		db:     database.GetDB(),
		parser: gofeed.NewParser(),
	}
}

// CreateRSSSource 创建RSS源
func (s *RSSService) CreateRSSSource(req *models.CreateRSSSourceRequest) (*models.RSSSourceResponse, error) {
	// 检查URL是否已存在
	var existingSource models.RSSSource
	if err := s.db.Where("url = ?", req.URL).First(&existingSource).Error; err == nil {
		return nil, errors.New("RSS source with this URL already exists")
	}

	// 测试RSS源是否可访问
	_, err := s.parser.ParseURL(req.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse RSS feed: %v", err)
	}

	// 创建RSS源
	source := models.RSSSource{
		Name:        req.Name,
		URL:         req.URL,
		Category:    req.Category,
		Language:    req.Language,
		Description: req.Description,
		Tags:        utils.SliceToJSON(req.Tags),
		Priority:    req.Priority,
		UpdateFreq:  req.UpdateFreq,
		IsActive:    true,
	}

	// 设置默认值
	if source.Language == "" {
		source.Language = "zh"
	}
	if source.Priority == 0 {
		source.Priority = 1
	}
	if source.UpdateFreq == 0 {
		source.UpdateFreq = 60
	}

	if err := s.db.Create(&source).Error; err != nil {
		return nil, err
	}

	return s.convertToRSSSourceResponse(&source), nil
}

// GetRSSSources 获取RSS源列表
func (s *RSSService) GetRSSSources(page, limit int, category string, isActive *bool) (*models.NewsListResponse, error) {
	var sources []models.RSSSource
	var total int64

	db := s.db.Model(&models.RSSSource{})

	// 添加筛选条件
	if category != "" {
		db = db.Where("category = ?", category)
	}
	if isActive != nil {
		db = db.Where("is_active = ?", *isActive)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页查询
	offset := (page - 1) * limit
	if err := db.Order("priority DESC, created_at DESC").Offset(offset).Limit(limit).Find(&sources).Error; err != nil {
		return nil, err
	}

	// 转换响应格式
	var responses []models.NewsItemResponse
	for _, source := range sources {
		// 这里将RSS源信息转换为新闻响应格式以复用现有结构
		response := models.NewsItemResponse{
			ID:          source.ID,
			Title:       source.Name,
			Description: source.Description,
			Category:    source.Category,
			Tags:        source.Tags,
			CreatedAt:   source.CreatedAt,
			UpdatedAt:   source.UpdatedAt,
		}
		responses = append(responses, response)
	}

	return &models.NewsListResponse{
		Total: total,
		News:  responses,
	}, nil
}

// UpdateRSSSource 更新RSS源
func (s *RSSService) UpdateRSSSource(id uint, req *models.UpdateRSSSourceRequest) (*models.RSSSourceResponse, error) {
	var source models.RSSSource
	if err := s.db.First(&source, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("RSS source not found")
		}
		return nil, err
	}

	// 更新字段
	if req.Name != "" {
		source.Name = req.Name
	}
	if req.URL != "" {
		// 检查新URL是否已存在
		var existingSource models.RSSSource
		if err := s.db.Where("url = ? AND id != ?", req.URL, id).First(&existingSource).Error; err == nil {
			return nil, errors.New("RSS source with this URL already exists")
		}

		// 测试新URL是否可访问
		_, err := s.parser.ParseURL(req.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to parse RSS feed: %v", err)
		}

		source.URL = req.URL
	}
	if req.Category != "" {
		source.Category = req.Category
	}
	if req.Language != "" {
		source.Language = req.Language
	}
	if req.IsActive != nil {
		source.IsActive = *req.IsActive
	}
	if req.Description != "" {
		source.Description = req.Description
	}
	if req.Tags != nil {
		source.Tags = utils.SliceToJSON(req.Tags)
	}
	if req.Priority > 0 {
		source.Priority = req.Priority
	}
	if req.UpdateFreq > 0 {
		source.UpdateFreq = req.UpdateFreq
	}

	if err := s.db.Save(&source).Error; err != nil {
		return nil, err
	}

	return s.convertToRSSSourceResponse(&source), nil
}

// DeleteRSSSource 删除RSS源
func (s *RSSService) DeleteRSSSource(id uint) error {
	var source models.RSSSource
	if err := s.db.First(&source, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("RSS source not found")
		}
		return err
	}

	// 软删除RSS源
	if err := s.db.Delete(&source).Error; err != nil {
		return err
	}

	return nil
}

// FetchRSSFeed 抓取单个RSS源的内容
func (s *RSSService) FetchRSSFeed(sourceID uint) (*models.RSSFetchStats, error) {
	log.Printf("[RSS DEBUG] Starting to fetch RSS feed for source ID: %d", sourceID)
	
	var source models.RSSSource
	if err := s.db.First(&source, sourceID).Error; err != nil {
		log.Printf("[RSS ERROR] Failed to find RSS source %d: %v", sourceID, err)
		return nil, err
	}

	log.Printf("[RSS DEBUG] Found RSS source: %s (URL: %s, Active: %v)", source.Name, source.URL, source.IsActive)

	if !source.IsActive {
		log.Printf("[RSS WARNING] RSS source %s is not active", source.Name)
		return nil, errors.New("RSS source is not active")
	}

	startTime := time.Now()
	stats := &models.RSSFetchStats{
		SourceID:   source.ID,
		SourceName: source.Name,
		FetchTime:  startTime,
	}

	// 解析RSS feed
	log.Printf("[RSS DEBUG] Parsing RSS feed from URL: %s", source.URL)
	feed, err := s.parser.ParseURL(source.URL)
	if err != nil {
		log.Printf("[RSS ERROR] Failed to parse RSS feed %s: %v", source.URL, err)
		// 增加错误计数
		s.db.Model(&source).UpdateColumn("error_count", gorm.Expr("error_count + 1"))
		return nil, fmt.Errorf("failed to parse RSS feed: %v", err)
	}

	log.Printf("[RSS DEBUG] Successfully parsed RSS feed, found %d items", len(feed.Items))

	stats.TotalItems = len(feed.Items)

	// 处理每个新闻条目
	for _, item := range feed.Items {
		newsItem, isNew, err := s.processNewsItem(&source, item)
		if err != nil {
			log.Printf("Error processing news item: %v", err)
			stats.ErrorItems++
			continue
		}

		if isNew {
			stats.NewItems++
		} else {
			stats.UpdatedItems++
		}

		// 自动计算热度
		if newsItem != nil {
			s.calculateNewsHotness(newsItem.ID)
		}
	}

	// 更新RSS源统计信息
	s.db.Model(&source).Updates(map[string]interface{}{
		"last_fetched": time.Now(),
		"fetch_count":  gorm.Expr("fetch_count + 1"),
	})

	stats.Duration = time.Since(startTime).String()
	return stats, nil
}

// FetchAllRSSFeeds 抓取所有活跃RSS源的内容
func (s *RSSService) FetchAllRSSFeeds() (*models.RSSFetchResult, error) {
	var sources []models.RSSSource
	if err := s.db.Where("is_active = ?", true).Find(&sources).Error; err != nil {
		return nil, err
	}

	result := &models.RSSFetchResult{
		Success: true,
		Stats:   make([]models.RSSFetchStats, 0),
	}

	successCount := 0
	for _, source := range sources {
		stats, err := s.FetchRSSFeed(source.ID)
		if err != nil {
			log.Printf("Failed to fetch RSS feed for source %s: %v", source.Name, err)
			result.Stats = append(result.Stats, models.RSSFetchStats{
				SourceID:   source.ID,
				SourceName: source.Name,
				ErrorItems: 1,
				FetchTime:  time.Now(),
			})
			continue
		}

		result.Stats = append(result.Stats, *stats)
		successCount++
	}

	if successCount == 0 {
		result.Success = false
		result.Message = "All RSS feeds failed to fetch"
	} else if successCount < len(sources) {
		result.Message = fmt.Sprintf("Partially successful: %d/%d sources fetched", successCount, len(sources))
	} else {
		result.Message = fmt.Sprintf("Successfully fetched %d RSS sources", successCount)
	}

	return result, nil
}

// processNewsItem 处理单个新闻条目
func (s *RSSService) processNewsItem(source *models.RSSSource, item *gofeed.Item) (*models.News, bool, error) {
	log.Printf("[RSS DEBUG] Processing news item: %s", item.Title)
	
	// 检查是否已存在
	var existingItem models.News
	isNew := false

	// 优先使用GUID，其次使用Link
	identifier := item.GUID
	if identifier == "" {
		identifier = item.Link
	}

	log.Printf("[RSS DEBUG] Checking for existing item with identifier: %s, link: %s", identifier, item.Link)

	err := s.db.Where("guid = ? OR link = ?", identifier, item.Link).First(&existingItem).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		isNew = true
		existingItem = models.News{}
		log.Printf("[RSS DEBUG] Item is new, will create")
	} else if err != nil {
		log.Printf("[RSS ERROR] Database error when checking existing item: %v", err)
		return nil, false, err
	} else {
		log.Printf("[RSS DEBUG] Item already exists with ID: %d, will update", existingItem.ID)
	}

	// 解析发布时间
	var publishedAt time.Time
	if item.PublishedParsed != nil {
		publishedAt = *item.PublishedParsed
	} else if item.UpdatedParsed != nil {
		publishedAt = *item.UpdatedParsed
	} else {
		publishedAt = time.Now()
	}

	// 提取图片URL
	imageURL := ""
	if item.Image != nil {
		imageURL = item.Image.URL
	}

	// 提取作者信息
	author := ""
	if item.Author != nil {
		author = item.Author.Name
	}

	// 提取分类
	categories := make([]string, 0)
	for _, cat := range item.Categories {
		categories = append(categories, cat)
	}
	categoryStr := strings.Join(categories, ",")
	if categoryStr == "" {
		categoryStr = source.Category
	}

	// 更新或创建新闻条目
	newsItem := models.News{
		RSSSourceID: &source.ID,
		SourceType:  models.NewsTypeRSS,
		Title:       item.Title,
		Link:        item.Link,
		Description: item.Description,
		Content:     item.Content,
		Author:      author,
		Category:    categoryStr,
		Tags:        utils.SliceToJSON(categories),
		PublishedAt: publishedAt,
		GUID:        identifier,
		ImageURL:    imageURL,
		Status:      "published",
		IsProcessed: false,
		Source:      source.Name,
		Language:    source.Language,
		IsActive:    true,
	}

	if isNew {
		newsItem.ID = 0 // 确保是新记录
		log.Printf("[RSS DEBUG] Creating new news item: %s", newsItem.Title)
		if err := s.db.Create(&newsItem).Error; err != nil {
			log.Printf("[RSS ERROR] Failed to create news item: %v", err)
			return nil, false, err
		}
		log.Printf("[RSS DEBUG] Successfully created news item with ID: %d", newsItem.ID)
	} else {
		// 更新现有记录
		newsItem.ID = existingItem.ID
		newsItem.ViewCount = existingItem.ViewCount
		newsItem.LikeCount = existingItem.LikeCount
		newsItem.CommentCount = existingItem.CommentCount
		newsItem.ShareCount = existingItem.ShareCount
		newsItem.HotnessScore = existingItem.HotnessScore
		newsItem.CreatedBy = existingItem.CreatedBy

		log.Printf("[RSS DEBUG] Updating existing news item ID: %d", newsItem.ID)
		if err := s.db.Save(&newsItem).Error; err != nil {
			log.Printf("[RSS ERROR] Failed to update news item: %v", err)
			return nil, false, err
		}
		log.Printf("[RSS DEBUG] Successfully updated news item ID: %d", newsItem.ID)
	}

	return &newsItem, isNew, nil
}

// GetNews 获取新闻列表
func (s *RSSService) GetNews(query *models.NewsQueryRequest) (*models.NewsListResponse, error) {
	var news []models.News
	var total int64

	db := s.db.Model(&models.News{}).Preload("RSSSource").Where("source_type = ?", models.NewsTypeRSS)

	// 添加筛选条件
	if query.RSSSourceID > 0 {
		db = db.Where("rss_source_id = ?", query.RSSSourceID)
	}
	if query.Category != "" {
		db = db.Where("category LIKE ?", "%"+query.Category+"%")
	}
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}
	if query.Search != "" {
		searchTerm := "%" + query.Search + "%"
		db = db.Where("title LIKE ? OR description LIKE ? OR content LIKE ?", searchTerm, searchTerm, searchTerm)
	}

	// 日期范围筛选
	if query.StartDate != "" {
		if startDate, err := time.Parse("2006-01-02", query.StartDate); err == nil {
			db = db.Where("published_at >= ?", startDate)
		}
	}
	if query.EndDate != "" {
		if endDate, err := time.Parse("2006-01-02", query.EndDate); err == nil {
			db = db.Where("published_at <= ?", endDate.Add(24*time.Hour))
		}
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// 排序
	orderBy := "published_at DESC"
	switch query.SortBy {
	case "hotness":
		orderBy = "hotness_score DESC, published_at DESC"
	case "views":
		orderBy = "view_count DESC, published_at DESC"
	case "published_at":
		orderBy = "published_at DESC"
	}

	// 分页
	offset := (query.Page - 1) * query.Limit
	if err := db.Order(orderBy).Offset(offset).Limit(query.Limit).Find(&news).Error; err != nil {
		return nil, err
	}

	// 转换响应格式
	var newsResponses []models.NewsItemResponse
	for _, item := range news {
		// 将 NewsResponse 转换为 NewsItemResponse 格式以保持API兼容性
		newsResp := item.ToResponse()
		newsItemResp := models.NewsItemResponse{
			ID:           newsResp.ID,
			RSSSourceID:  *newsResp.RSSSourceID,
			Title:        newsResp.Title,
			Link:         newsResp.Link,
			Description:  newsResp.Description,
			Content:      newsResp.Content,
			Author:       newsResp.Author,
			Category:     newsResp.Category,
			Tags:         newsResp.Tags,
			PublishedAt:  newsResp.PublishedAt,
			GUID:         newsResp.GUID,
			ImageURL:     newsResp.ImageURL,
			ViewCount:    newsResp.ViewCount,
			LikeCount:    newsResp.LikeCount,
			CommentCount: newsResp.CommentCount,
			ShareCount:   newsResp.ShareCount,
			HotnessScore: newsResp.HotnessScore,
			Status:       newsResp.Status,
			IsProcessed:  newsResp.IsProcessed,
			CreatedAt:    newsResp.CreatedAt,
			UpdatedAt:    newsResp.UpdatedAt,
			RSSSource:    newsResp.RSSSource,
		}
		newsResponses = append(newsResponses, newsItemResp)
	}

	return &models.NewsListResponse{
		Total: total,
		News:  newsResponses,
	}, nil
}

// GetNewsItem 获取单个新闻详情
func (s *RSSService) GetNewsItem(id uint) (*models.NewsItemResponse, error) {
	var newsItem models.News
	if err := s.db.Preload("RSSSource").Where("source_type = ?", models.NewsTypeRSS).First(&newsItem, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("news item not found")
		}
		return nil, err
	}

	// 增加浏览量
	s.db.Model(&newsItem).UpdateColumn("view_count", gorm.Expr("view_count + 1"))
	newsItem.ViewCount++

	// 重新计算热度
	s.calculateNewsHotness(newsItem.ID)

	// 转换为NewsItemResponse格式
	newsResp := newsItem.ToResponse()
	response := models.NewsItemResponse{
		ID:           newsResp.ID,
		RSSSourceID:  *newsResp.RSSSourceID,
		Title:        newsResp.Title,
		Link:         newsResp.Link,
		Description:  newsResp.Description,
		Content:      newsResp.Content,
		Author:       newsResp.Author,
		Category:     newsResp.Category,
		Tags:         newsResp.Tags,
		PublishedAt:  newsResp.PublishedAt,
		GUID:         newsResp.GUID,
		ImageURL:     newsResp.ImageURL,
		ViewCount:    newsResp.ViewCount,
		LikeCount:    newsResp.LikeCount,
		CommentCount: newsResp.CommentCount,
		ShareCount:   newsResp.ShareCount,
		HotnessScore: newsResp.HotnessScore,
		Status:       newsResp.Status,
		IsProcessed:  newsResp.IsProcessed,
		CreatedAt:    newsResp.CreatedAt,
		UpdatedAt:    newsResp.UpdatedAt,
		RSSSource:    newsResp.RSSSource,
	}
	return &response, nil
}

// calculateNewsHotness 计算新闻热度
func (s *RSSService) calculateNewsHotness(newsID uint) error {
	var newsItem models.News
	if err := s.db.First(&newsItem, newsID).Error; err != nil {
		return err
	}

	// 计算热度分值（类似事件热度计算）
	viewScore := float64(newsItem.ViewCount) / 1000.0 * 8.0
	if viewScore > 10 {
		viewScore = 10
	}

	likeScore := float64(newsItem.LikeCount) / 100.0
	if likeScore > 10 {
		likeScore = 10
	}

	commentScore := float64(newsItem.CommentCount) / 10.0
	if commentScore > 10 {
		commentScore = 10
	}

	shareScore := float64(newsItem.ShareCount) / 5.0
	if shareScore > 10 {
		shareScore = 10
	}

	// 时间因素
	hours := time.Since(newsItem.PublishedAt).Hours()
	timeScore := 10.0
	if hours > 1 {
		timeScore = 9.0
	}
	if hours > 6 {
		timeScore = 8.0
	}
	if hours > 24 {
		timeScore = 6.0
	}
	if hours > 72 {
		timeScore = 4.0
	}
	if hours > 168 {
		timeScore = 2.0
	}

	// 综合计算
	finalScore := viewScore*0.2 + likeScore*0.3 + commentScore*0.25 + shareScore*0.15 + timeScore*0.1
	if finalScore > 10 {
		finalScore = 10
	}

	// 更新热度分值
	return s.db.Model(&newsItem).UpdateColumn("hotness_score", finalScore).Error
}

// 转换函数
func (s *RSSService) convertToRSSSourceResponse(source *models.RSSSource) *models.RSSSourceResponse {
	return &models.RSSSourceResponse{
		ID:          source.ID,
		Name:        source.Name,
		URL:         source.URL,
		Category:    source.Category,
		Language:    source.Language,
		IsActive:    source.IsActive,
		LastFetched: source.LastFetched,
		FetchCount:  source.FetchCount,
		ErrorCount:  source.ErrorCount,
		Description: source.Description,
		Tags:        source.Tags,
		Priority:    source.Priority,
		UpdateFreq:  source.UpdateFreq,
		CreatedAt:   source.CreatedAt,
		UpdatedAt:   source.UpdatedAt,
	}
}

// convertToNewsItemResponse 函数已删除，现在直接使用 News.ToResponse() 方法
