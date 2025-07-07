package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"sort"
	"strings"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"gorm.io/gorm"
)

// 辅助函数：将字符串切片转换为JSON字符串
func sliceToJSON(slice []string) string {
	if len(slice) == 0 {
		return "[]"
	}
	jsonBytes, _ := json.Marshal(slice)
	return string(jsonBytes)
}

// 辅助函数：将JSON字符串转换为字符串切片
func jsonToSlice(jsonStr string) []string {
	if jsonStr == "" {
		return []string{}
	}
	var slice []string
	json.Unmarshal([]byte(jsonStr), &slice)
	return slice
}

type EventService struct {
	db *gorm.DB
}

func NewEventService() *EventService {
	return &EventService{
		db: database.GetDB(),
	}
}

// GetEvents 获取事件列表
func (s *EventService) GetEvents(query *models.EventQueryRequest) (*models.EventListResponse, error) {
	var events []models.Event
	var total int64

	db := s.db.Model(&models.Event{})

	// 添加状态筛选
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	// 添加分类筛选
	if query.Category != "" {
		db = db.Where("category = ?", query.Category)
	}

	// 添加搜索条件
	if query.Search != "" {
		searchTerm := "%" + query.Search + "%"
		db = db.Where("title LIKE ? OR description LIKE ? OR content LIKE ? OR location LIKE ?", searchTerm, searchTerm, searchTerm, searchTerm)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// 排序
	orderBy := "created_at desc"
	switch query.SortBy {
	case "hotness":
		orderBy = "hotness_score desc, created_at desc"
	case "views":
		orderBy = "view_count desc, created_at desc"
	case "time":
		orderBy = "created_at desc"
	default:
		orderBy = "created_at desc"
	}

	// 分页
	offset := (query.Page - 1) * query.Limit
	err := db.Order(orderBy).Offset(offset).Limit(query.Limit).Find(&events).Error
	if err != nil {
		return nil, err
	}

	// 构建响应
	var eventResponses []models.EventResponse
	for _, event := range events {
		eventResponses = append(eventResponses, convertToEventResponse(&event))
	}

	return &models.EventListResponse{
		Total:  total,
		Events: eventResponses,
	}, nil
}

// GetEventByID 根据ID获取事件
func (s *EventService) GetEventByID(id uint) (*models.EventResponse, error) {
	var event models.Event
	if err := s.db.First(&event, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("event not found")
		}
		return nil, err
	}

	response := convertToEventResponse(&event)
	return &response, nil
}

// ViewEvent 浏览事件（增加浏览量并重新计算热度）
func (s *EventService) ViewEvent(id uint) (*models.EventResponse, error) {
	var event models.Event
	if err := s.db.First(&event, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("event not found")
		}
		return nil, err
	}

	// 增加浏览量
	err := s.db.Model(&models.Event{}).
		Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error

	if err != nil {
		return nil, err
	}

	// 自动重新计算热度值
	_, err = s.CalculateHotness(id, nil)
	if err != nil {
		return nil, err
	}

	// 重新获取更新后的事件数据
	if err := s.db.First(&event, id).Error; err != nil {
		return nil, err
	}

	response := convertToEventResponse(&event)
	return &response, nil
}

// CreateEvent 创建事件
func (s *EventService) CreateEvent(req *models.CreateEventRequest) (*models.EventResponse, error) {
	// 检查时间
	if req.EndTime.Before(req.StartTime) {
		return nil, errors.New("end time must be after start time")
	}

	event := models.Event{
		Title:        req.Title,
		Description:  req.Description,
		Content:      req.Content,
		StartTime:    req.StartTime,
		EndTime:      req.EndTime,
		Location:     req.Location,
		Status:       "进行中",
		CreatedBy:    1, // 这里应该从当前用户上下文中获取，示例代码使用固定值
		Image:        req.Image,
		Category:     req.Category,
		Tags:         sliceToJSON(req.Tags),
		Source:       req.Source,
		Author:       req.Author,
		RelatedLinks: sliceToJSON(req.RelatedLinks),
		ViewCount:    0,
		HotnessScore: 0.0,
	}

	// 保存到数据库
	if err := s.db.Create(&event).Error; err != nil {
		return nil, err
	}

	response := convertToEventResponse(&event)
	return &response, nil
}

// UpdateEvent 更新事件
func (s *EventService) UpdateEvent(id uint, req *models.UpdateEventRequest) (*models.EventResponse, error) {
	var event models.Event
	if err := s.db.First(&event, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("event not found")
		}
		return nil, err
	}

	// 更新字段
	if req.Title != "" {
		event.Title = req.Title
	}
	if req.Description != "" {
		event.Description = req.Description
	}
	if req.Content != "" {
		event.Content = req.Content
	}
	if !req.StartTime.IsZero() {
		event.StartTime = req.StartTime
	}
	if !req.EndTime.IsZero() {
		event.EndTime = req.EndTime
	}
	if req.Location != "" {
		event.Location = req.Location
	}
	if req.Status != "" {
		event.Status = req.Status
	}
	if req.Category != "" {
		event.Category = req.Category
	}
	if req.Tags != nil {
		event.Tags = sliceToJSON(req.Tags)
	}
	if req.Source != "" {
		event.Source = req.Source
	}
	if req.Author != "" {
		event.Author = req.Author
	}
	if req.RelatedLinks != nil {
		event.RelatedLinks = sliceToJSON(req.RelatedLinks)
	}
	if req.Image != "" {
		event.Image = req.Image
	}

	// 验证时间
	if event.EndTime.Before(event.StartTime) {
		return nil, errors.New("end time must be after start time")
	}

	// 更新数据库
	if err := s.db.Save(&event).Error; err != nil {
		return nil, err
	}

	response := convertToEventResponse(&event)
	return &response, nil
}

// DeleteEvent 删除事件
func (s *EventService) DeleteEvent(id uint) error {
	var event models.Event
	if err := s.db.First(&event, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("event not found")
		}
		return err
	}

	// 使用软删除
	if err := s.db.Delete(&event).Error; err != nil {
		return err
	}

	return nil
}

// GetEventsByStatus 根据状态获取事件
func (s *EventService) GetEventsByStatus(status string) ([]models.EventResponse, error) {
	var events []models.Event
	if err := s.db.Where("status = ?", status).Find(&events).Error; err != nil {
		return nil, err
	}

	// 构建响应
	var eventResponses []models.EventResponse
	for _, event := range events {
		eventResponses = append(eventResponses, convertToEventResponse(&event))
	}

	return eventResponses, nil
}

// 检查并更新事件状态
func (s *EventService) UpdateEventStatus() error {
	now := time.Now()

	// 找出所有已结束但状态仍为"进行中"的事件
	if err := s.db.Model(&models.Event{}).
		Where("end_time < ? AND status = ?", now, "进行中").
		Update("status", "已结束").Error; err != nil {
		return err
	}

	return nil
}

// 将Event转换为EventResponse
func convertToEventResponse(event *models.Event) models.EventResponse {
	return models.EventResponse{
		ID:           event.ID,
		Title:        event.Title,
		Description:  event.Description,
		Content:      event.Content,
		StartTime:    event.StartTime,
		EndTime:      event.EndTime,
		Location:     event.Location,
		Status:       event.Status,
		CreatedBy:    event.CreatedBy,
		Image:        event.Image,
		Category:     event.Category,
		Tags:         event.Tags,
		Source:       event.Source,
		Author:       event.Author,
		RelatedLinks: event.RelatedLinks,
		ViewCount:    event.ViewCount,
		LikeCount:    event.LikeCount,
		CommentCount: event.CommentCount,
		ShareCount:   event.ShareCount,
		HotnessScore: event.HotnessScore,
		CreatedAt:    event.CreatedAt,
		UpdatedAt:    event.UpdatedAt,
	}
}

// GetHotEvents 获取热点事件
func (s *EventService) GetHotEvents(limit int) ([]models.EventResponse, error) {
	if limit <= 0 {
		limit = 10
	}

	var events []models.Event
	err := s.db.Where("status = ?", "进行中").
		Order("hotness_score desc, view_count desc, created_at desc").
		Limit(limit).Find(&events).Error
	if err != nil {
		return nil, err
	}

	var eventResponses []models.EventResponse
	for _, event := range events {
		eventResponses = append(eventResponses, convertToEventResponse(&event))
	}

	return eventResponses, nil
}

// GetEventCategories 获取所有事件分类
func (s *EventService) GetEventCategories() ([]string, error) {
	var categories []string
	err := s.db.Model(&models.Event{}).
		Select("DISTINCT category").
		Where("category != ''").
		Order("category").
		Pluck("category", &categories).Error
	return categories, err
}

// IncrementViewCount 增加事件浏览次数
func (s *EventService) IncrementViewCount(id uint) error {
	return s.db.Model(&models.Event{}).
		Where("id = ?", id).
		UpdateColumn("view_count", gorm.Expr("view_count + 1")).Error
}

// UpdateHotnessScore 更新事件热度分值
func (s *EventService) UpdateHotnessScore(id uint, score float64) error {
	return s.db.Model(&models.Event{}).
		Where("id = ?", id).
		UpdateColumn("hotness_score", score).Error
}

// GetEventsByCategory 按分类获取事件列表
func (s *EventService) GetEventsByCategory(category string, query *models.EventQueryRequest) (*models.EventListResponse, error) {
	var events []models.Event
	var total int64

	db := s.db.Model(&models.Event{}).Where("category = ?", category)

	// 添加状态筛选
	if query.Status != "" {
		db = db.Where("status = ?", query.Status)
	}

	// 计算总数
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// 排序
	switch query.SortBy {
	case "hotness":
		db = db.Order("hotness_score DESC, created_at DESC")
	case "views":
		db = db.Order("view_count DESC, created_at DESC")
	default:
		db = db.Order("created_at DESC")
	}

	// 分页
	offset := (query.Page - 1) * query.Limit
	if err := db.Offset(offset).Limit(query.Limit).Find(&events).Error; err != nil {
		return nil, err
	}

	var eventResponses []models.EventResponse
	for _, event := range events {
		eventResponses = append(eventResponses, convertToEventResponse(&event))
	}

	return &models.EventListResponse{
		Total:  total,
		Events: eventResponses,
	}, nil
}

// GetPopularTags 获取热门标签
func (s *EventService) GetPopularTags(limit int, minCount int) ([]models.TagResponse, error) {
	type TagCount struct {
		Tag      string `json:"tag"`
		Count    int    `json:"count"`
		Category string `json:"category"`
	}

	var results []TagCount
	// 这里使用原生SQL查询，因为标签存储为JSON字符串
	query := `
		SELECT 
			TRIM(BOTH '"' FROM JSON_EXTRACT(tags, CONCAT('$[', numbers.n, ']'))) as tag,
			COUNT(*) as count,
			category
		FROM events
		CROSS JOIN (
			SELECT 0 as n UNION SELECT 1 UNION SELECT 2 UNION SELECT 3 UNION SELECT 4 
			UNION SELECT 5 UNION SELECT 6 UNION SELECT 7 UNION SELECT 8 UNION SELECT 9
		) numbers
		WHERE JSON_LENGTH(tags) > numbers.n
		AND tags != '[]' AND tags != ''
		GROUP BY tag, category
		HAVING count >= ?
		ORDER BY count DESC, tag
		LIMIT ?
	`

	err := s.db.Raw(query, minCount, limit).Scan(&results).Error
	if err != nil {
		return nil, err
	}

	var tagResponses []models.TagResponse
	for _, result := range results {
		tagResponses = append(tagResponses, models.TagResponse{
			Tag:      result.Tag,
			Count:    result.Count,
			Category: result.Category,
		})
	}

	return tagResponses, nil
}

// GetTrendingEvents 获取趋势事件
func (s *EventService) GetTrendingEvents(limit int, timeRange string) ([]models.TrendingEventResponse, error) {
	var events []models.Event

	// 根据时间范围确定查询条件
	var timeCondition time.Time
	switch timeRange {
	case "1h":
		timeCondition = time.Now().Add(-1 * time.Hour)
	case "6h":
		timeCondition = time.Now().Add(-6 * time.Hour)
	case "24h":
		timeCondition = time.Now().Add(-24 * time.Hour)
	case "7d":
		timeCondition = time.Now().Add(-7 * 24 * time.Hour)
	default:
		timeCondition = time.Now().Add(-24 * time.Hour)
	}

	// 查询在指定时间范围内创建或更新的事件
	// 按热度分值和浏览量的组合进行排序，模拟趋势计算
	err := s.db.Where("created_at >= ? OR updated_at >= ?", timeCondition, timeCondition).
		Order("(hotness_score * 0.6 + LEAST(view_count / 100.0, 10) * 0.4) DESC").
		Limit(limit).
		Find(&events).Error

	if err != nil {
		return nil, err
	}

	var trendingEvents []models.TrendingEventResponse
	for _, event := range events {
		// 计算趋势分值（这里简化计算，实际应用中可能需要更复杂的算法）
		trendScore := event.HotnessScore*0.6 + float64(event.ViewCount)/100.0*0.4
		if trendScore > 10 {
			trendScore = 10
		}

		// 计算浏览量增长率（这里使用模拟值，实际需要历史数据对比）
		viewGrowthRate := float64(event.ViewCount) * 0.1 // 简化计算

		trendingEvents = append(trendingEvents, models.TrendingEventResponse{
			ID:             event.ID,
			Title:          event.Title,
			Category:       event.Category,
			HotnessScore:   event.HotnessScore,
			TrendScore:     trendScore,
			ViewGrowthRate: viewGrowthRate,
			Status:         event.Status,
			CreatedAt:      event.CreatedAt,
		})
	}

	return trendingEvents, nil
}

// UpdateEventTags 更新事件标签
func (s *EventService) UpdateEventTags(id uint, tags []string, operation string) (*models.Event, error) {
	var event models.Event
	if err := s.db.First(&event, id).Error; err != nil {
		return nil, err
	}

	var newTags []string
	currentTags := jsonToSlice(event.Tags)

	switch operation {
	case "replace":
		newTags = tags
	case "add":
		// 添加新标签，去重
		tagMap := make(map[string]bool)
		for _, tag := range currentTags {
			tagMap[tag] = true
		}
		for _, tag := range tags {
			tagMap[tag] = true
		}
		for tag := range tagMap {
			newTags = append(newTags, tag)
		}
	case "remove":
		// 移除指定标签
		removeMap := make(map[string]bool)
		for _, tag := range tags {
			removeMap[tag] = true
		}
		for _, tag := range currentTags {
			if !removeMap[tag] {
				newTags = append(newTags, tag)
			}
		}
	default:
		newTags = tags
	}

	event.Tags = sliceToJSON(newTags)
	event.UpdatedAt = time.Now()

	if err := s.db.Save(&event).Error; err != nil {
		return nil, err
	}

	return &event, nil
}

// CalculateHotness 计算事件热度（自动计算）
func (s *EventService) CalculateHotness(id uint, factors *models.HotnessFactors) (*models.HotnessCalculationResult, error) {
	var event models.Event
	if err := s.db.First(&event, id).Error; err != nil {
		return nil, err
	}

	// 设置默认权重
	if factors == nil {
		factors = &models.HotnessFactors{
			ViewWeight:    0.2,
			LikeWeight:    0.3,
			CommentWeight: 0.25,
			ShareWeight:   0.15,
			TimeWeight:    0.1,
		}
	}

	// 计算各项分值
	viewScore := calculateViewScore(event.ViewCount)
	likeScore := calculateLikeScore(event.LikeCount)
	commentScore := calculateCommentScore(event.CommentCount)
	shareScore := calculateShareScore(event.ShareCount)
	timeScore := calculateTimeScore(event.CreatedAt)

	// 计算最终分值
	finalScore := viewScore*factors.ViewWeight +
		likeScore*factors.LikeWeight +
		commentScore*factors.CommentWeight +
		shareScore*factors.ShareWeight +
		timeScore*factors.TimeWeight

	// 限制在0-10范围内
	if finalScore > 10 {
		finalScore = 10
	}
	if finalScore < 0 {
		finalScore = 0
	}

	previousScore := event.HotnessScore
	event.HotnessScore = finalScore
	event.UpdatedAt = time.Now()

	if err := s.db.Save(&event).Error; err != nil {
		return nil, err
	}

	return &models.HotnessCalculationResult{
		ID:            event.ID,
		HotnessScore:  finalScore,
		PreviousScore: previousScore,
		CalculationDetails: models.CalculationDetails{
			ViewScore:    viewScore,
			LikeScore:    likeScore,
			CommentScore: commentScore,
			ShareScore:   shareScore,
			TimeScore:    timeScore,
			FinalScore:   finalScore,
		},
		UpdatedAt: event.UpdatedAt,
	}, nil
}

// 辅助函数：计算浏览量分值
func calculateViewScore(viewCount int64) float64 {
	// 浏览量分值计算：对数增长，最高10分
	if viewCount == 0 {
		return 0
	}
	score := float64(viewCount) / 1000.0 * 8.0
	if score > 10 {
		score = 10
	}
	return score
}

// 辅助函数：计算时间分值
func calculateTimeScore(createdAt time.Time) float64 {
	// 时间分值计算：越新的事件分值越高
	hours := time.Since(createdAt).Hours()
	if hours <= 1 {
		return 10
	} else if hours <= 6 {
		return 9
	} else if hours <= 24 {
		return 8
	} else if hours <= 72 {
		return 6
	} else if hours <= 168 { // 7天
		return 4
	} else {
		return 2
	}
}

// 辅助函数：计算点赞分值
func calculateLikeScore(likeCount int64) float64 {
	// 点赞分值计算：对数增长，最高10分
	if likeCount == 0 {
		return 0
	}
	// 每100个点赞算1分，最高10分
	score := float64(likeCount) / 100.0
	if score > 10 {
		score = 10
	}
	return score
}

// 辅助函数：计算评论分值
func calculateCommentScore(commentCount int64) float64 {
	// 评论分值计算：评论比点赞更有价值
	if commentCount == 0 {
		return 0
	}
	// 每10个评论算1分，最高10分
	score := float64(commentCount) / 10.0
	if score > 10 {
		score = 10
	}
	return score
}

// 辅助函数：计算分享分值
func calculateShareScore(shareCount int64) float64 {
	// 分享分值计算：分享是最有价值的互动
	if shareCount == 0 {
		return 0
	}
	// 每5个分享算1分，最高10分
	score := float64(shareCount) / 5.0
	if score > 10 {
		score = 10
	}
	return score
}

// LikeEvent 点赞事件
func (s *EventService) LikeEvent(eventID uint, userID uint) error {
	// 这里可以实现点赞逻辑，比如检查用户是否已经点赞过
	// 为简化，这里直接增加点赞数
	err := s.db.Model(&models.Event{}).
		Where("id = ?", eventID).
		UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error

	if err != nil {
		return err
	}

	// 自动重新计算热度值
	_, err = s.CalculateHotness(eventID, nil)
	return err
}

// UnlikeEvent 取消点赞事件
func (s *EventService) UnlikeEvent(eventID uint, userID uint) error {
	// 这里可以实现取消点赞逻辑
	// 为简化，这里直接减少点赞数（确保不会小于0）
	err := s.db.Model(&models.Event{}).
		Where("id = ? AND like_count > 0", eventID).
		UpdateColumn("like_count", gorm.Expr("like_count - 1")).Error

	if err != nil {
		return err
	}

	// 自动重新计算热度值
	_, err = s.CalculateHotness(eventID, nil)
	return err
}

// IncrementCommentCount 增加评论数
func (s *EventService) IncrementCommentCount(eventID uint) error {
	err := s.db.Model(&models.Event{}).
		Where("id = ?", eventID).
		UpdateColumn("comment_count", gorm.Expr("comment_count + 1")).Error

	if err != nil {
		return err
	}

	// 自动重新计算热度值
	_, err = s.CalculateHotness(eventID, nil)
	return err
}

// IncrementShareCount 增加分享数
func (s *EventService) IncrementShareCount(eventID uint) error {
	err := s.db.Model(&models.Event{}).
		Where("id = ?", eventID).
		UpdateColumn("share_count", gorm.Expr("share_count + 1")).Error

	if err != nil {
		return err
	}

	// 自动重新计算热度值
	_, err = s.CalculateHotness(eventID, nil)
	return err
}

// GetEventStats 获取事件统计信息
func (s *EventService) GetEventStats(eventID uint) (*models.InteractionStatsResponse, error) {
	var event models.Event
	if err := s.db.First(&event, eventID).Error; err != nil {
		return nil, err
	}

	return &models.InteractionStatsResponse{
		EventID:      event.ID,
		ViewCount:    event.ViewCount,
		LikeCount:    event.LikeCount,
		CommentCount: event.CommentCount,
		ShareCount:   event.ShareCount,
		HotnessScore: event.HotnessScore,
	}, nil
}

// EventCluster 表示一个事件聚类
type EventCluster struct {
	Title        string
	Description  string
	Category     string
	StartTime    time.Time
	EndTime      time.Time
	Location     string
	Status       string
	Tags         []string
	Source       string
	NewsList     []models.News
	HotnessScore float64
}

// EventGenerationResult 事件生成结果
type EventGenerationResult struct {
	GeneratedEvents   []models.EventResponse `json:"generated_events"`
	TotalEvents       int                    `json:"total_events"`
	ProcessedNews     int                    `json:"processed_news"`
	GenerationTime    time.Time              `json:"generation_time"`
	ElapsedTime       string                 `json:"elapsed_time"`
	CategoryBreakdown map[string]int         `json:"category_breakdown"`
}

// GenerateEventsFromNews 从新闻自动生成事件
func (s *EventService) GenerateEventsFromNews() (*EventGenerationResult, error) {
	startTime := time.Now()

	// 1. 获取所有新闻
	var allNews []models.News
	if err := s.db.Find(&allNews).Error; err != nil {
		return nil, fmt.Errorf("获取新闻列表失败: %w", err)
	}

	if len(allNews) == 0 {
		return nil, errors.New("数据库中没有新闻，无法生成事件")
	}

	// 2. 根据分类将新闻分组
	categoryNews := make(map[string][]models.News)
	for _, news := range allNews {
		if news.Category == "" {
			news.Category = "未分类"
		}
		categoryNews[news.Category] = append(categoryNews[news.Category], news)
	}

	// 3. 为每个分类生成事件聚类
	allEventClusters := make([]*EventCluster, 0)
	categoryBreakdown := make(map[string]int)

	for category, newsList := range categoryNews {
		// 对每个分类内的新闻进行简单聚类
		clusters := s.clusterNewsByTitle(newsList)
		categoryBreakdown[category] = len(clusters)
		allEventClusters = append(allEventClusters, clusters...)
	}

	// 4. 按热度对事件聚类排序
	sort.Slice(allEventClusters, func(i, j int) bool {
		return allEventClusters[i].HotnessScore > allEventClusters[j].HotnessScore
	})

	// 5. 转换聚类为事件并保存到数据库
	generatedEvents := make([]models.EventResponse, 0)
	newsService := NewNewsService() // 创建新闻服务实例

	for _, cluster := range allEventClusters {
		event := s.convertClusterToEvent(cluster)

		// 保存事件到数据库
		savedEvent, err := s.CreateEvent(&event)
		if err != nil {
			return nil, fmt.Errorf("保存事件失败: %w", err)
		}

		// 更新关联新闻的事件ID
		newsIDs := make([]uint, 0)
		for _, news := range cluster.NewsList {
			newsIDs = append(newsIDs, news.ID)
		}

		if err := newsService.UpdateNewsEventAssociation(newsIDs, savedEvent.ID); err != nil {
			return nil, fmt.Errorf("更新新闻关联失败: %w", err)
		}

		generatedEvents = append(generatedEvents, *savedEvent)
	}

	elapsed := time.Since(startTime)
	return &EventGenerationResult{
		GeneratedEvents:   generatedEvents,
		TotalEvents:       len(generatedEvents),
		ProcessedNews:     len(allNews),
		GenerationTime:    time.Now(),
		ElapsedTime:       elapsed.String(),
		CategoryBreakdown: categoryBreakdown,
	}, nil
}

// clusterNewsByTitle 根据标题相似度简单聚类新闻
func (s *EventService) clusterNewsByTitle(newsList []models.News) []*EventCluster {
	// 按发布时间排序
	sort.Slice(newsList, func(i, j int) bool {
		return newsList[i].PublishedAt.Before(newsList[j].PublishedAt)
	})

	clusters := make([]*EventCluster, 0)
	processed := make(map[uint]bool)

	// 对每个未处理的新闻，尝试创建或加入聚类
	for i, news := range newsList {
		if processed[news.ID] {
			continue
		}

		// 创建新聚类，使用分类作为标题
		eventTitle := news.Category
		if eventTitle == "" || eventTitle == "未分类" {
			eventTitle = "综合新闻"
		}

		cluster := &EventCluster{
			Title:        eventTitle,
			Description:  news.Summary,
			Category:     news.Category,
			StartTime:    news.PublishedAt,
			EndTime:      news.PublishedAt.Add(24 * time.Hour), // 默认事件持续一天
			Location:     "全国",                                 // 默认位置
			Status:       "进行中",
			Tags:         s.extractTags(news),
			Source:       news.Source,
			NewsList:     []models.News{news},
			HotnessScore: float64(news.ViewCount + news.LikeCount*2 + news.CommentCount*3 + news.ShareCount*5),
		}

		// 查找相似的新闻加入此聚类
		for j := i + 1; j < len(newsList); j++ {
			if processed[newsList[j].ID] {
				continue
			}

			// 如果标题相似度高，加入聚类
			if s.areTitlesSimilar(news.Title, newsList[j].Title) {
				cluster.NewsList = append(cluster.NewsList, newsList[j])
				processed[newsList[j].ID] = true

				// 更新聚类信息
				if newsList[j].PublishedAt.Before(cluster.StartTime) {
					cluster.StartTime = newsList[j].PublishedAt
				}
				if newsList[j].PublishedAt.After(cluster.EndTime) {
					cluster.EndTime = newsList[j].PublishedAt
				}

				// 合并标签
				newsTags := s.extractTags(newsList[j])
				for _, tag := range newsTags {
					if !s.contains(cluster.Tags, tag) {
						cluster.Tags = append(cluster.Tags, tag)
					}
				}

				// 累加热度分数
				cluster.HotnessScore += float64(newsList[j].ViewCount + newsList[j].LikeCount*2 +
					newsList[j].CommentCount*3 + newsList[j].ShareCount*5)
			}
		}

		// 完善聚类信息
		if len(cluster.Description) == 0 && len(cluster.NewsList) > 0 {
			// 如果没有描述，使用第一条新闻的内容开头作为描述
			for _, n := range cluster.NewsList {
				if len(n.Content) > 10 {
					// 使用内容的前100个字符作为描述
					endPos := int(math.Min(100, float64(len(n.Content))))
					cluster.Description = n.Content[:endPos] + "..."
					break
				}
			}
		}

		// 更新状态
		now := time.Now()
		if cluster.EndTime.Before(now) {
			cluster.Status = "已结束"
		} else if cluster.StartTime.After(now) {
			cluster.Status = "未开始"
		}

		// 将此聚类加入结果
		clusters = append(clusters, cluster)
		processed[news.ID] = true
	}

	return clusters
}

// areTitlesSimilar 检查两个标题是否相似（放宽标准，提高聚合程度）
func (s *EventService) areTitlesSimilar(title1, title2 string) bool {
	title1 = strings.ToLower(title1)
	title2 = strings.ToLower(title2)

	// 1. 检查地域关键词匹配
	if s.hasRegionalMatch(title1, title2) {
		return true
	}

	// 2. 检查主题关键词匹配
	if s.hasTopicMatch(title1, title2) {
		return true
	}

	// 3. 通用关键词相似度检查（降低阈值）
	keywords1 := s.extractKeywords(title1)
	keywords2 := s.extractKeywords(title2)

	var matches int
	var totalKeywords int

	// 计算关键词匹配数
	for _, kw1 := range keywords1 {
		if len(kw1) < 2 {
			continue
		}
		totalKeywords++
		for _, kw2 := range keywords2 {
			if len(kw2) < 2 {
				continue
			}
			// 包含关系或编辑距离相似
			if strings.Contains(kw1, kw2) || strings.Contains(kw2, kw1) || s.isSimilarKeyword(kw1, kw2) {
				matches++
				break // 避免重复计算
			}
		}
	}

	// 降低相似度阈值：只需要20%的关键词匹配或至少1个关键词匹配
	minKeywords := int(math.Max(1, float64(totalKeywords)*0.2))
	return matches >= minKeywords
}

// hasRegionalMatch 检查是否有地域关键词匹配
func (s *EventService) hasRegionalMatch(title1, title2 string) bool {
	// 定义地域关键词组
	regionalGroups := map[string][]string{
		"中东":   {"中东", "以色列", "巴勒斯坦", "伊朗", "叙利亚", "伊拉克", "沙特", "阿联酋", "黎巴嫩", "约旦", "也门", "卡塔尔", "科威特", "巴林", "阿曼"},
		"俄乌":   {"俄罗斯", "乌克兰", "俄乌", "普京", "泽连斯基", "基辅", "莫斯科", "顿巴斯", "克里米亚", "哈尔科夫", "马里乌波尔"},
		"朝鲜半岛": {"朝鲜", "韩国", "金正恩", "文在寅", "尹锡悦", "平壤", "首尔", "三八线", "板门店"},
		"南海":   {"南海", "台海", "台湾", "南沙", "西沙", "钓鱼岛", "尖阁诸岛"},
		"欧洲":   {"欧盟", "英国", "法国", "德国", "意大利", "西班牙", "荷兰", "比利时", "瑞士", "奥地利", "波兰", "捷克"},
		"美洲":   {"美国", "加拿大", "墨西哥", "巴西", "阿根廷", "智利", "哥伦比亚", "委内瑞拉"},
		"非洲":   {"埃及", "南非", "尼日利亚", "肯尼亚", "摩洛哥", "阿尔及利亚", "利比亚", "苏丹", "埃塞俄比亚"},
		"东南亚":  {"越南", "泰国", "新加坡", "马来西亚", "印尼", "菲律宾", "缅甸", "柬埔寨", "老挝"},
		"南亚":   {"印度", "巴基斯坦", "孟加拉国", "斯里兰卡", "尼泊尔", "不丹", "马尔代夫"},
	}

	// 检查两个标题是否属于同一地域
	for _, keywords := range regionalGroups {
		found1, found2 := false, false
		for _, keyword := range keywords {
			if strings.Contains(title1, keyword) {
				found1 = true
			}
			if strings.Contains(title2, keyword) {
				found2 = true
			}
		}
		if found1 && found2 {
			return true
		}
	}

	return false
}

// hasTopicMatch 检查是否有主题关键词匹配
func (s *EventService) hasTopicMatch(title1, title2 string) bool {
	// 定义主题关键词组
	topicGroups := map[string][]string{
		"经济": {"经济", "GDP", "通胀", "利率", "股市", "汇率", "贸易", "投资", "金融", "央行", "货币", "市场", "企业", "公司"},
		"科技": {"科技", "AI", "人工智能", "5G", "芯片", "半导体", "互联网", "数字", "智能", "技术", "创新", "研发"},
		"政治": {"政治", "选举", "总统", "首相", "政府", "议会", "国会", "外交", "会谈", "峰会", "访问", "制裁"},
		"军事": {"军事", "军队", "武器", "导弹", "战机", "军演", "防务", "安全", "冲突", "战争", "和平", "停火"},
		"能源": {"能源", "石油", "天然气", "电力", "核能", "煤炭", "新能源", "太阳能", "风能", "电池"},
		"环境": {"环境", "气候", "碳排放", "全球变暖", "污染", "环保", "绿色", "可持续", "减排"},
		"健康": {"健康", "医疗", "疫苗", "病毒", "疫情", "医院", "药物", "治疗", "医生", "患者"},
		"教育": {"教育", "学校", "大学", "学生", "老师", "考试", "学习", "培训", "课程"},
		"体育": {"体育", "奥运", "世界杯", "足球", "篮球", "网球", "游泳", "田径", "运动员", "比赛"},
		"文化": {"文化", "艺术", "电影", "音乐", "文学", "博物馆", "遗产", "传统", "节庆"},
	}

	// 检查两个标题是否属于同一主题
	for _, keywords := range topicGroups {
		found1, found2 := false, false
		for _, keyword := range keywords {
			if strings.Contains(title1, keyword) {
				found1 = true
			}
			if strings.Contains(title2, keyword) {
				found2 = true
			}
		}
		if found1 && found2 {
			return true
		}
	}

	return false
}

// isSimilarKeyword 检查两个关键词是否相似（编辑距离等）
func (s *EventService) isSimilarKeyword(kw1, kw2 string) bool {
	// 简单的相似度检查：长度相近且有公共子串
	if math.Abs(float64(len(kw1)-len(kw2))) > 2 {
		return false
	}

	// 检查是否有公共子串（长度>=2）
	for i := 0; i <= len(kw1)-2; i++ {
		substr := kw1[i : i+2]
		if strings.Contains(kw2, substr) {
			return true
		}
	}

	return false
}

// extractKeywords 从标题中提取关键词（增强版）
func (s *EventService) extractKeywords(title string) []string {
	title = strings.ToLower(title)

	// 替换标点符号为空格
	title = strings.ReplaceAll(title, "，", " ")
	title = strings.ReplaceAll(title, "。", " ")
	title = strings.ReplaceAll(title, "？", " ")
	title = strings.ReplaceAll(title, "！", " ")
	title = strings.ReplaceAll(title, "、", " ")
	title = strings.ReplaceAll(title, "：", " ")
	title = strings.ReplaceAll(title, "；", " ")
	title = strings.ReplaceAll(title, ",", " ")
	title = strings.ReplaceAll(title, ".", " ")
	title = strings.ReplaceAll(title, "?", " ")
	title = strings.ReplaceAll(title, "!", " ")
	title = strings.ReplaceAll(title, ":", " ")
	title = strings.ReplaceAll(title, ";", " ")
	title = strings.ReplaceAll(title, "'", " ")
	title = strings.ReplaceAll(title, "\"", " ")
	title = strings.ReplaceAll(title, "(", " ")
	title = strings.ReplaceAll(title, ")", " ")
	title = strings.ReplaceAll(title, "（", " ")
	title = strings.ReplaceAll(title, "）", " ")
	title = strings.ReplaceAll(title, "[", " ")
	title = strings.ReplaceAll(title, "]", " ")
	title = strings.ReplaceAll(title, "【", " ")
	title = strings.ReplaceAll(title, "】", " ")
	title = strings.ReplaceAll(title, "《", " ")
	title = strings.ReplaceAll(title, "》", " ")

	words := strings.Split(title, " ")
	result := make([]string, 0)

	// 扩展停用词列表
	stopWords := map[string]bool{
		"的": true, "了": true, "是": true, "在": true, "有": true, "和": true, "与": true, "为": true, "将": true, "被": true, "把": true, "对": true, "向": true, "从": true, "到": true, "于": true, "以": true, "及": true, "或": true, "而": true, "且": true, "但": true, "不": true, "没": true, "无": true, "非": true,
		"a": true, "an": true, "the": true, "to": true, "of": true, "for": true, "and": true, "or": true, "in": true, "on": true, "at": true, "by": true, "with": true, "from": true, "up": true, "about": true, "into": true, "through": true, "during": true, "before": true, "after": true, "above": true, "below": true, "between": true, "among": true, "this": true, "that": true, "these": true, "those": true,
		"新闻": true, "报道": true, "消息": true, "最新": true, "今日": true, "昨日": true, "今天": true, "昨天": true, "明天": true, "本周": true, "上周": true, "下周": true, "本月": true, "上月": true, "下月": true, "今年": true, "去年": true, "明年": true,
	}

	for _, word := range words {
		word = strings.TrimSpace(word)
		// 保留长度>=2且不是停用词的词语
		if len(word) >= 2 && !stopWords[word] {
			result = append(result, word)
		}
	}

	return result
}

// extractTags 从新闻中提取标签
func (s *EventService) extractTags(news models.News) []string {
	tags := make([]string, 0)

	// 1. 从新闻的tags字段提取
	if news.Tags != "" {
		// 如果是JSON格式，解析
		if strings.HasPrefix(news.Tags, "[") && strings.HasSuffix(news.Tags, "]") {
			var parsedTags []string
			if err := json.Unmarshal([]byte(news.Tags), &parsedTags); err == nil {
				tags = append(tags, parsedTags...)
			}
		} else {
			// 否则按逗号分割
			for _, tag := range strings.Split(news.Tags, ",") {
				tag = strings.TrimSpace(tag)
				if tag != "" {
					tags = append(tags, tag)
				}
			}
		}
	}

	// 2. 添加分类作为标签
	if news.Category != "" && !s.contains(tags, news.Category) {
		tags = append(tags, news.Category)
	}

	// 3. 添加来源作为标签
	if news.Source != "" && !s.contains(tags, news.Source) {
		tags = append(tags, news.Source)
	}

	return tags
}

// contains 检查切片是否包含指定元素
func (s *EventService) contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

// convertClusterToEvent 将聚类转换为创建事件的请求
func (s *EventService) convertClusterToEvent(cluster *EventCluster) models.CreateEventRequest {
	// 生成详细内容
	content := fmt.Sprintf("# %s\n\n## 事件概述\n\n%s\n\n## 相关新闻\n\n",
		cluster.Title, cluster.Description)

	links := make([]string, 0)
	for _, news := range cluster.NewsList {
		// 添加链接
		if news.Link != "" {
			links = append(links, news.Link)
		}

		// 添加新闻到内容
		content += fmt.Sprintf("### %s\n\n", news.Title)
		if news.Source != "" {
			content += fmt.Sprintf("来源: %s   ", news.Source)
		}
		if !news.PublishedAt.IsZero() {
			content += fmt.Sprintf("发布时间: %s\n\n", news.PublishedAt.Format("2006-01-02 15:04:05"))
		} else {
			content += "\n\n"
		}

		// 添加摘要或内容片段
		if news.Summary != "" {
			content += news.Summary + "\n\n"
		} else if news.Content != "" {
			// 使用内容的前200个字符作为摘要
			endPos := int(math.Min(200, float64(len(news.Content))))
			content += news.Content[:endPos]
			if len(news.Content) > 200 {
				content += "..."
			}
			content += "\n\n"
		}
	}

	return models.CreateEventRequest{
		Title:        cluster.Title,
		Description:  cluster.Description,
		Content:      content,
		StartTime:    cluster.StartTime,
		EndTime:      cluster.EndTime,
		Location:     cluster.Location,
		Category:     cluster.Category,
		Tags:         cluster.Tags,
		Source:       cluster.Source,
		Author:       "系统自动生成",
		RelatedLinks: links,
	}
}

// UpdateEventStats 更新事件统计信息
func (s *EventService) UpdateEventStats(eventID uint) error {
	// 1. 获取事件基本信息
	var event models.Event
	if err := s.db.First(&event, eventID).Error; err != nil {
		return fmt.Errorf("获取事件信息失败: %w", err)
	}

	// 2. 统计关联的新闻数量
	var newsCount int64
	if err := s.db.Model(&models.News{}).
		Where("belonged_event_id = ?", eventID).
		Count(&newsCount).Error; err != nil {
		return fmt.Errorf("统计新闻数量失败: %w", err)
	}

	// 3. 获取关联的所有新闻来重新计算统计信息
	var relatedNews []models.News
	if err := s.db.Where("belonged_event_id = ?", eventID).
		Find(&relatedNews).Error; err != nil {
		return fmt.Errorf("获取关联新闻失败: %w", err)
	}

	// 4. 重新计算聚合统计信息
	var totalViews, totalLikes, totalComments, totalShares int64
	var totalHotness float64

	// 生成新的标签
	var allTags []string
	tagCountMap := make(map[string]int)

	for _, news := range relatedNews {
		totalViews += news.ViewCount
		totalLikes += news.LikeCount
		totalComments += news.CommentCount
		totalShares += news.ShareCount
		totalHotness += news.HotnessScore

		// 从新闻中提取标签
		newsTags := s.extractTags(news)
		for _, tag := range newsTags {
			tagCountMap[tag]++
		}
	}

	// 5. 选择出现频次最高的标签作为事件标签
	for tag, count := range tagCountMap {
		if count >= 2 || len(tagCountMap) <= 5 { // 如果标签总数不多，降低阈值
			allTags = append(allTags, tag)
		}
	}

	// 6. 按标签出现次数排序，取前10个
	sort.Slice(allTags, func(i, j int) bool {
		return tagCountMap[allTags[i]] > tagCountMap[allTags[j]]
	})

	maxTags := 10
	if len(allTags) > maxTags {
		allTags = allTags[:maxTags]
	}

	// 7. 更新事件的统计信息
	updates := map[string]interface{}{
		"view_count":    totalViews,
		"like_count":    totalLikes,
		"comment_count": totalComments,
		"share_count":   totalShares,
		"hotness_score": totalHotness,
		"tags":          sliceToJSON(allTags),
		"updated_at":    time.Now(),
	}

	if err := s.db.Model(&models.Event{}).
		Where("id = ?", eventID).
		Updates(updates).Error; err != nil {
		return fmt.Errorf("更新事件统计信息失败: %w", err)
	}

	return nil
}

// UpdateAllEventStats 更新所有事件的统计信息
func (s *EventService) UpdateAllEventStats() error {
	// 获取所有事件ID
	var eventIDs []uint
	if err := s.db.Model(&models.Event{}).
		Pluck("id", &eventIDs).Error; err != nil {
		return fmt.Errorf("获取事件ID列表失败: %w", err)
	}

	// 逐个更新事件统计信息
	for _, eventID := range eventIDs {
		if err := s.UpdateEventStats(eventID); err != nil {
			// 记录错误但继续处理其他事件
			fmt.Printf("更新事件 %d 统计信息失败: %v\n", eventID, err)
		}
	}

	return nil
}

// RefreshEventHotness 刷新事件热度评分
func (s *EventService) RefreshEventHotness(eventID uint) error {
	// 先更新统计信息
	if err := s.UpdateEventStats(eventID); err != nil {
		return err
	}

	// 重新计算热度
	_, err := s.CalculateHotness(eventID, nil)
	return err
}

// BatchUpdateEventStats 批量更新指定事件的统计信息
func (s *EventService) BatchUpdateEventStats(eventIDs []uint) error {
	for _, eventID := range eventIDs {
		if err := s.UpdateEventStats(eventID); err != nil {
			return fmt.Errorf("批量更新事件 %d 统计信息失败: %w", eventID, err)
		}
	}
	return nil
}
