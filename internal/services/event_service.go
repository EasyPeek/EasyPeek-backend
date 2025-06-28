package services

import (
	"encoding/json"
	"errors"
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
