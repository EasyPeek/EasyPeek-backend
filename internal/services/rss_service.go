package services

import (
	"errors"
	"fmt"
	"html"
	"log"
	"math/rand"
	"net/http"
	"regexp"
	"strings"
	"time"
	"unicode/utf8"

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

// NewsListResponse 新闻列表响应结构
type NewsListResponse struct {
	Total int64                 `json:"total"`
	News  []models.NewsResponse `json:"news"`
}

// NewsQueryRequest 新闻查询请求结构
type NewsQueryRequest struct {
	Page        int    `json:"page" form:"page"`
	Limit       int    `json:"limit" form:"limit"`
	RSSSourceID uint   `json:"rss_source_id" form:"rss_source_id"`
	Category    string `json:"category" form:"category"`
	Status      string `json:"status" form:"status"`
	Search      string `json:"search" form:"search"`
	StartDate   string `json:"start_date" form:"start_date"`
	EndDate     string `json:"end_date" form:"end_date"`
	SortBy      string `json:"sort_by" form:"sort_by"`
}

func NewRSSService() *RSSService {
	parser := gofeed.NewParser()

	client := &http.Client{
		Timeout: 30 * time.Second,
		Transport: &http.Transport{
			MaxIdleConns:       10,
			IdleConnTimeout:    30 * time.Second,
			DisableCompression: false,
		},
	}

	parser.Client = client
	parser.UserAgent = "EasyPeek/1.0 RSS Reader (Mozilla/5.0 compatible)"

	return &RSSService{
		db:     database.GetDB(),
		parser: parser,
	}
}

// CreateRSSSource
func (s *RSSService) CreateRSSSource(req *models.CreateRSSSourceRequest) (*models.RSSSource, error) {
	var existingSource models.RSSSource
	if err := s.db.Where("url = ?", req.URL).First(&existingSource).Error; err == nil {
		return nil, errors.New("RSS source with this URL already exists")
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

	return &source, nil
}

// GetRSSSources
func (s *RSSService) GetAllRSSSources(page, limit int) ([]models.RSSSource, int64, error) {
	var sources []models.RSSSource
	var total int64

	db := s.db.Model(&models.RSSSource{})

	if err := db.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	if err := db.Order("id ASC").Offset(offset).Limit(limit).Find(&sources).Error; err != nil {
		return nil, 0, err
	}

	return sources, total, nil
}

// UpdateRSSSource
func (s *RSSService) UpdateRSSSource(id uint, req *models.UpdateRSSSourceRequest) (*models.RSSSource, error) {
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

	if err := s.db.Save(&source).Error; err != nil {
		return nil, err
	}

	return &source, nil
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

// FetchRSSFeed 抓取单个RSS源的内容（带重试机制）
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

	// 解析RSS feed（带重试机制）
	feed, err := s.fetchRSSWithRetry(source.URL, 3)
	if err != nil {
		log.Printf("[RSS ERROR] Failed to parse RSS feed %s after retries: %v", source.URL, err)
		// 增加错误计数
		s.db.Model(&source).UpdateColumn("error_count", gorm.Expr("error_count + 1"))

		// 如果错误次数过多，可以考虑暂时禁用该源
		if source.ErrorCount >= 10 {
			log.Printf("[RSS WARNING] RSS source %s has too many errors (%d), consider reviewing", source.Name, source.ErrorCount+1)
		}

		return nil, fmt.Errorf("failed to parse RSS feed: %v", err)
	}

	log.Printf("[RSS DEBUG] Successfully parsed RSS feed, found %d items", len(feed.Items))

	stats.TotalItems = len(feed.Items)

	// 处理每个新闻条目
	for i, item := range feed.Items {
		log.Printf("[RSS DEBUG] Processing item %d/%d: %s", i+1, len(feed.Items), item.Title)

		newsItem, isNew, err := s.processNewsItem(&source, item)
		if err != nil {
			log.Printf("[RSS ERROR] Error processing news item '%s': %v", item.Title, err)
			stats.ErrorItems++
			continue
		}

		if isNew {
			stats.NewItems++
			log.Printf("[RSS DEBUG] Created new news item: %s", item.Title)
		} else {
			stats.UpdatedItems++
			log.Printf("[RSS DEBUG] Updated existing news item: %s", item.Title)
		}

		// 计算新闻热度
		if newsItem != nil {
			if err := s.calculateNewsHotness(newsItem.ID); err != nil {
				log.Printf("[RSS WARNING] Failed to calculate hotness for news item %d: %v", newsItem.ID, err)
			}
		}
	}

	// 更新RSS源统计信息
	updateData := map[string]interface{}{
		"last_fetched": time.Now(),
		"fetch_count":  gorm.Expr("fetch_count + 1"),
	}

	// 如果这次抓取成功，重置错误计数
	if stats.ErrorItems < len(feed.Items)/2 { // 如果错误条目少于总数的一半
		updateData["error_count"] = 0
	}

	s.db.Model(&source).Updates(updateData)

	stats.Duration = time.Since(startTime).String()
	log.Printf("[RSS DEBUG] RSS fetch completed for %s: %d new, %d updated, %d errors in %s",
		source.Name, stats.NewItems, stats.UpdatedItems, stats.ErrorItems, stats.Duration)

	return stats, nil
}

// fetchRSSWithRetry 带重试机制的RSS抓取
func (s *RSSService) fetchRSSWithRetry(url string, maxRetries int) (*gofeed.Feed, error) {
	var lastError error

	for attempt := 1; attempt <= maxRetries; attempt++ {
		log.Printf("[RSS DEBUG] Attempting to fetch RSS from %s (attempt %d/%d)", url, attempt, maxRetries)

		feed, err := s.parser.ParseURL(url)
		if err == nil {
			log.Printf("[RSS DEBUG] Successfully fetched RSS from %s on attempt %d", url, attempt)
			return feed, nil
		}

		lastError = err
		log.Printf("[RSS WARNING] Attempt %d failed for %s: %v", attempt, url, err)

		// 检查错误类型
		if strings.Contains(err.Error(), "timeout") {
			log.Printf("[RSS WARNING] Timeout error for %s, will retry", url)
		} else if strings.Contains(err.Error(), "no such host") || strings.Contains(err.Error(), "connection refused") {
			log.Printf("[RSS WARNING] Network error for %s: %v", url, err)
		} else if strings.Contains(err.Error(), "404") || strings.Contains(err.Error(), "403") {
			log.Printf("[RSS WARNING] HTTP error for %s: %v, will not retry", url, err)
			break // 不重试HTTP错误
		}

		// 等待后重试（指数退避）
		if attempt < maxRetries {
			waitTime := time.Duration(attempt) * 2 * time.Second
			log.Printf("[RSS DEBUG] Waiting %v before retry...", waitTime)
			time.Sleep(waitTime)
		}
	}

	return nil, fmt.Errorf("failed after %d attempts, last error: %v", maxRetries, lastError)
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

// generateNewsStats 为新闻生成合理的统计数据
func (s *RSSService) generateNewsStats(publishedAt time.Time, source *models.RSSSource) (int64, int64, int64, int64, float64) {
	// 根据发布时间计算时间权重（越新的新闻基础数据越低）
	now := time.Now()
	hoursOld := now.Sub(publishedAt).Hours()

	// 时间权重：0-24小时内权重最低，随时间增加
	timeWeight := 1.0
	if hoursOld <= 24 {
		timeWeight = 0.3 + (hoursOld / 24 * 0.4) // 0.3-0.7
	} else if hoursOld <= 168 { // 一周内
		timeWeight = 0.7 + ((hoursOld - 24) / 144 * 0.3) // 0.7-1.0
	} else {
		timeWeight = 1.0 + (hoursOld-168)/168*0.5 // 1.0-1.5
	}

	// 根据RSS源的重要性调整基础权重
	sourceWeight := 1.0
	switch source.Priority {
	case 1, 2: // 高优先级源
		sourceWeight = 1.2
	case 3, 4: // 中优先级源
		sourceWeight = 1.0
	default: // 低优先级源
		sourceWeight = 0.8
	}

	// 生成基础统计数据
	baseViews := int64(rand.Intn(200) + 50) // 50-250 基础浏览量
	viewCount := int64(float64(baseViews) * timeWeight * sourceWeight)

	// 点赞数约为浏览量的 3-8%
	likeRate := 0.03 + rand.Float64()*0.05 // 3%-8%
	likeCount := int64(float64(viewCount) * likeRate)

	// 评论数约为浏览量的 0.5-2%
	commentRate := 0.005 + rand.Float64()*0.015 // 0.5%-2%
	commentCount := int64(float64(viewCount) * commentRate)

	// 分享数约为浏览量的 1-3%
	shareRate := 0.01 + rand.Float64()*0.02 // 1%-3%
	shareCount := int64(float64(viewCount) * shareRate)

	// 计算热度分数 (0-10分)
	hotnessScore := float64(viewCount)*0.001 +
		float64(likeCount)*0.01 +
		float64(commentCount)*0.05 +
		float64(shareCount)*0.02

	// 添加时间衰减
	if hoursOld <= 24 {
		hotnessScore *= 1.5 // 24小时内热度加成
	} else if hoursOld <= 168 {
		hotnessScore *= (1.5 - (hoursOld-24)/144*0.5) // 逐渐衰减
	} else {
		hotnessScore *= 1.0 - (hoursOld-168)/168*0.3 // 继续衰减
	}

	// 限制热度分数在0-10之间
	if hotnessScore > 10 {
		hotnessScore = 10
	}
	if hotnessScore < 0 {
		hotnessScore = 0
	}

	log.Printf("[RSS DEBUG] Generated stats for news - Views: %d, Likes: %d, Comments: %d, Shares: %d, Hotness: %.2f",
		viewCount, likeCount, commentCount, shareCount, hotnessScore)

	return viewCount, likeCount, commentCount, shareCount, hotnessScore
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
		author = s.cleanUTF8(item.Author.Name)
	}

	// 提取分类
	categories := make([]string, len(item.Categories))
	for i, cat := range item.Categories {
		categories[i] = s.cleanUTF8(cat)
	}
	categoryStr := strings.Join(categories, ",")
	if categoryStr == "" {
		categoryStr = source.Category
	}

	// 处理内容字段 - 优先使用Content，如果为空则使用Description
	content := item.Content
	if content == "" && item.Description != "" {
		content = item.Description
		log.Printf("[RSS DEBUG] Using description as content for item: %s", item.Title)
	}

	// 清理所有文本字段的UTF-8编码
	title := s.cleanUTF8(item.Title)
	description := s.cleanUTF8(item.Description)
	content = s.cleanUTF8(content)
	link := s.cleanUTF8(item.Link)
	guid := s.cleanUTF8(identifier)

	// 提取摘要 - 从Description或Content中提取前200个字符作为摘要
	summary := ""
	if description != "" {
		summary = s.extractSummary(description, 200)
	} else if content != "" {
		summary = s.extractSummary(content, 200)
	}

	// 生成或补充统计数据
	var viewCount, likeCount, commentCount, shareCount int64
	var hotnessScore float64

	if isNew {
		// 新新闻：生成完整的统计数据
		viewCount, likeCount, commentCount, shareCount, hotnessScore = s.generateNewsStats(publishedAt, source)
		log.Printf("[RSS DEBUG] Generated stats for new news: %s", title)
	} else {
		// 现有新闻：检查是否需要补充统计数据
		viewCount = existingItem.ViewCount
		likeCount = existingItem.LikeCount
		commentCount = existingItem.CommentCount
		shareCount = existingItem.ShareCount
		hotnessScore = existingItem.HotnessScore

		// 如果统计数据为0或过低，生成合理的数据
		if viewCount == 0 && likeCount == 0 && commentCount == 0 {
			log.Printf("[RSS DEBUG] Existing news has no stats, generating new stats for: %s", title)
			viewCount, likeCount, commentCount, shareCount, hotnessScore = s.generateNewsStats(publishedAt, source)
		} else if viewCount < 10 && time.Since(publishedAt).Hours() > 24 {
			// 如果发布超过24小时但浏览量还很低，适当补充一些数据
			log.Printf("[RSS DEBUG] Boosting low stats for older news: %s", title)
			additionalViews, additionalLikes, additionalComments, additionalShares, newHotness := s.generateNewsStats(publishedAt, source)

			// 增加一些数据，但不要完全替换
			viewCount += additionalViews / 3
			likeCount += additionalLikes / 3
			commentCount += additionalComments / 3
			shareCount += additionalShares / 3

			// 重新计算热度
			hotnessScore = newHotness
		}
	}

	// 更新或创建新闻条目
	newsItem := models.News{
		RSSSourceID:  &source.ID,
		SourceType:   models.NewsTypeRSS,
		Title:        title,
		Link:         link,
		Description:  description,
		Content:      content,
		Summary:      summary,
		Author:       author,
		Category:     categoryStr,
		Tags:         utils.SliceToJSON(categories),
		PublishedAt:  publishedAt,
		GUID:         guid,
		ImageURL:     imageURL,
		Status:       "published",
		IsProcessed:  false,
		Source:       source.Name,
		Language:     source.Language,
		IsActive:     true,
		ViewCount:    viewCount,
		LikeCount:    likeCount,
		CommentCount: commentCount,
		ShareCount:   shareCount,
		HotnessScore: hotnessScore,
	}

	// 保存到数据库
	if isNew {
		log.Printf("[RSS DEBUG] Creating new news item: %s", title)
		if err := s.db.Create(&newsItem).Error; err != nil {
			log.Printf("[RSS ERROR] Failed to create news item: %v", err)
			return nil, false, err
		}
	} else {
		log.Printf("[RSS DEBUG] Updating existing news item: %s", title)
		// 更新现有条目
		existingItem.Title = title
		existingItem.Description = description
		existingItem.Content = content
		existingItem.Summary = summary
		existingItem.Author = author
		existingItem.Category = categoryStr
		existingItem.Tags = utils.SliceToJSON(categories)
		existingItem.PublishedAt = publishedAt
		existingItem.ImageURL = imageURL
		existingItem.Source = source.Name
		existingItem.Language = source.Language
		existingItem.IsActive = true

		// 更新统计数据（如果生成了新的数据）
		existingItem.ViewCount = viewCount
		existingItem.LikeCount = likeCount
		existingItem.CommentCount = commentCount
		existingItem.ShareCount = shareCount
		existingItem.HotnessScore = hotnessScore

		if err := s.db.Save(&existingItem).Error; err != nil {
			log.Printf("[RSS ERROR] Failed to update news item: %v", err)
			return nil, false, err
		}
		newsItem = existingItem
	}

	// 新闻数据保存完成
	log.Printf("[RSS DEBUG] 新闻已保存到数据库，ID: %d", newsItem.ID)

	return &newsItem, isNew, nil
}

// cleanUTF8 清理字符串中的无效UTF-8字符
func (s *RSSService) cleanUTF8(text string) string {
	if text == "" {
		return text
	}

	// 如果已经是有效的UTF-8，直接返回
	if utf8.ValidString(text) {
		return text
	}

	// 清理无效的UTF-8字符
	cleaned := make([]rune, 0, len(text))
	for _, r := range text {
		if r != utf8.RuneError {
			cleaned = append(cleaned, r)
		}
	}

	result := string(cleaned)

	// 如果清理后仍然无效，则进行更严格的清理
	if !utf8.ValidString(result) {
		// 使用更严格的方法：逐字节检查
		var buffer strings.Builder
		for i, width := 0, 0; i < len(text); i += width {
			r, w := utf8.DecodeRuneInString(text[i:])
			if r == utf8.RuneError && w == 1 {
				// 跳过无效字符
				width = 1
				continue
			}
			buffer.WriteRune(r)
			width = w
		}
		result = buffer.String()
	}

	return result
}

// extractSummary 从HTML内容中提取纯文本摘要（改进版）
func (s *RSSService) extractSummary(htmlContent string, maxLength int) string {
	if htmlContent == "" {
		return ""
	}

	// 先清理UTF-8编码
	content := s.cleanUTF8(htmlContent)

	// 解码HTML实体
	content = html.UnescapeString(content)

	// 移除HTML标签
	re := regexp.MustCompile(`<[^>]*>`)
	text := re.ReplaceAllString(content, "")

	// 移除多余的空白字符
	text = regexp.MustCompile(`\s+`).ReplaceAllString(text, " ")
	text = strings.TrimSpace(text)

	// 再次清理UTF-8编码（以防HTML解码引入了无效字符）
	text = s.cleanUTF8(text)

	// 截取指定长度
	if len(text) > maxLength {
		// 按字符截取，而不是按字节
		runes := []rune(text)
		if len(runes) > maxLength {
			text = string(runes[:maxLength])

			// 尝试在单词或句子边界截断
			if lastSpace := strings.LastIndex(text, " "); lastSpace > maxLength/2 {
				text = text[:lastSpace]
			}

			text = strings.TrimSpace(text) + "..."
		}
	}

	return text
}

// GetNews 获取新闻列表
func (s *RSSService) GetNews(query *NewsQueryRequest) (*NewsListResponse, error) {
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
	var newsResponses []models.NewsResponse
	for _, item := range news {
		newsResponses = append(newsResponses, item.ToResponse())
	}

	return &NewsListResponse{
		Total: total,
		News:  newsResponses,
	}, nil
}

// GetNewsItem 获取单个新闻详情
func (s *RSSService) GetNewsItem(id uint) (*models.NewsResponse, error) {
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

	// 直接使用 ToResponse 方法
	response := newsItem.ToResponse()
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

// GetRSSCategories 获取所有RSS源分类
func (s *RSSService) GetRSSCategories() ([]string, error) {
	var categories []string

	err := s.db.Model(&models.RSSSource{}).
		Distinct("category").
		Where("category != ''").
		Pluck("category", &categories).Error

	if err != nil {
		return nil, err
	}

	return categories, nil
}

// RSSStats RSS统计信息
type RSSStats struct {
	TotalSources    int64 `json:"total_sources"`
	ActiveSources   int64 `json:"active_sources"`
	InactiveSources int64 `json:"inactive_sources"`
	TotalNews       int64 `json:"total_news"`
	TodayNews       int64 `json:"today_news"`
	Categories      int64 `json:"categories"`
}

// GetRSSStats 获取RSS统计信息
func (s *RSSService) GetRSSStats() (*RSSStats, error) {
	stats := &RSSStats{}

	// 总RSS源数
	if err := s.db.Model(&models.RSSSource{}).Count(&stats.TotalSources).Error; err != nil {
		return nil, err
	}

	// 活跃RSS源数
	if err := s.db.Model(&models.RSSSource{}).Where("is_active = ?", true).Count(&stats.ActiveSources).Error; err != nil {
		return nil, err
	}

	// 非活跃RSS源数
	stats.InactiveSources = stats.TotalSources - stats.ActiveSources

	// 总新闻数（来自RSS）
	if err := s.db.Model(&models.News{}).Where("source_type = ?", models.NewsTypeRSS).Count(&stats.TotalNews).Error; err != nil {
		return nil, err
	}

	// 今日新闻数
	today := time.Now().Truncate(24 * time.Hour)
	if err := s.db.Model(&models.News{}).
		Where("source_type = ? AND created_at >= ?", models.NewsTypeRSS, today).
		Count(&stats.TodayNews).Error; err != nil {
		return nil, err
	}

	// 分类数
	if err := s.db.Model(&models.RSSSource{}).
		Distinct("category").
		Where("category != ''").
		Count(&stats.Categories).Error; err != nil {
		return nil, err
	}

	return stats, nil
}

