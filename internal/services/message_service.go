package services

import (
	"errors"
	"fmt"
	"log"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"gorm.io/gorm"
)

type MessageService struct {
	db *gorm.DB
}

// NewMessageService 创建消息服务实例
func NewMessageService() *MessageService {
	return &MessageService{
		db: database.GetDB(),
	}
}

// CreateMessage 创建消息
func (s *MessageService) CreateMessage(userID uint, msgType, title, content string, relatedType string, relatedID uint, senderID *uint) error {
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	// 验证消息类型
	validTypes := []string{"system", "like", "comment", "follow", "news_update", "event_update"}
	isValidType := false
	for _, validType := range validTypes {
		if validType == msgType {
			isValidType = true
			break
		}
	}
	if !isValidType {
		return errors.New("invalid message type")
	}

	// 验证用户是否存在
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	message := &models.Message{
		UserID:      userID,
		Type:        msgType,
		Title:       title,
		Content:     content,
		RelatedType: relatedType,
		RelatedID:   relatedID,
		SenderID:    senderID,
		IsRead:      false,
	}

	return s.db.Create(message).Error
}

// GetUserMessages 获取用户消息列表
func (s *MessageService) GetUserMessages(userID uint, page, pageSize int, msgType string) ([]models.MessageResponse, int64, error) {
	if s.db == nil {
		return nil, 0, errors.New("database connection not initialized")
	}

	var messages []models.Message
	var total int64

	// 构建查询
	query := s.db.Model(&models.Message{}).Where("user_id = ?", userID)
	if msgType != "" {
		query = query.Where("type = ?", msgType)
	}

	// 统计总数
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := query.Preload("Sender").Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&messages).Error; err != nil {
		return nil, 0, err
	}

	// 转换为响应格式
	var responses []models.MessageResponse
	for _, message := range messages {
		responses = append(responses, message.ToResponse())
	}

	return responses, total, nil
}

// MarkAsRead 标记单条消息为已读
func (s *MessageService) MarkAsRead(userID, messageID uint) error {
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	result := s.db.Model(&models.Message{}).Where("id = ? AND user_id = ?", messageID, userID).Update("is_read", true)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("message not found or access denied")
	}
	return nil
}

// MarkAllAsRead 标记用户所有消息为已读
func (s *MessageService) MarkAllAsRead(userID uint) error {
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	return s.db.Model(&models.Message{}).Where("user_id = ? AND is_read = ?", userID, false).Update("is_read", true).Error
}

// GetUnreadCount 获取用户未读消息数量
func (s *MessageService) GetUnreadCount(userID uint) (int64, error) {
	if s.db == nil {
		return 0, errors.New("database connection not initialized")
	}

	var count int64
	err := s.db.Model(&models.Message{}).Where("user_id = ? AND is_read = ?", userID, false).Count(&count).Error
	return count, err
}

// DeleteMessage 删除消息
func (s *MessageService) DeleteMessage(userID, messageID uint) error {
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	result := s.db.Where("id = ? AND user_id = ?", messageID, userID).Delete(&models.Message{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("message not found or access denied")
	}
	return nil
}

// CreateBatchMessages 批量创建消息（用于通知关注者）
func (s *MessageService) CreateBatchMessages(userIDs []uint, msgType, title, content string, relatedType string, relatedID uint, senderID *uint) error {
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	if len(userIDs) == 0 {
		return nil
	}

	var messages []models.Message
	for _, userID := range userIDs {
		messages = append(messages, models.Message{
			UserID:      userID,
			Type:        msgType,
			Title:       title,
			Content:     content,
			RelatedType: relatedType,
			RelatedID:   relatedID,
			SenderID:    senderID,
			IsRead:      false,
		})
	}

	return s.db.CreateInBatches(messages, 100).Error
}

// NotifyEventFollowers 通知关注某个事件的用户
func (s *MessageService) NotifyEventFollowers(eventID uint, msgType, title, content string, relatedType string, relatedID uint, senderID *uint) error {
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	// 获取关注该事件的用户ID列表
	var follows []models.Follow
	if err := s.db.Where("event_id = ?", eventID).Find(&follows).Error; err != nil {
		return err
	}

	if len(follows) == 0 {
		return nil // 没有关注者
	}

	// 提取用户ID
	var userIDs []uint
	for _, follow := range follows {
		userIDs = append(userIDs, follow.UserID)
	}

	// 批量创建消息
	return s.CreateBatchMessages(userIDs, msgType, title, content, relatedType, relatedID, senderID)
}

// GetMessagesByType 根据类型获取消息
func (s *MessageService) GetMessagesByType(userID uint, msgType string, page, pageSize int) ([]models.MessageResponse, int64, error) {
	return s.GetUserMessages(userID, page, pageSize, msgType)
}

// CleanupOldMessages 清理旧消息（可选功能，清理超过指定天数的已读消息）
func (s *MessageService) CleanupOldMessages(days int) error {
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	if days <= 0 {
		return errors.New("days must be positive")
	}

	// 删除超过指定天数的已读消息
	result := s.db.Where("is_read = ? AND created_at < NOW() - INTERVAL ? DAY", true, days).Delete(&models.Message{})
	return result.Error
}

// GetFollowedEventsLatestNews 获取用户关注事件的最新新闻
func (s *MessageService) GetFollowedEventsLatestNews(userID uint, limit int) (*models.FollowedEventsNewsResponse, error) {
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	// 设置默认限制
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	// 获取用户关注的事件列表
	var follows []models.Follow
	if err := s.db.Preload("Event").Where("user_id = ?", userID).Find(&follows).Error; err != nil {
		return nil, err
	}

	if len(follows) == 0 {
		return &models.FollowedEventsNewsResponse{
			TotalEvents: 0,
			EventsNews:  []models.EventLatestNewsResponse{},
		}, nil
	}

	// 提取事件ID列表
	var eventIDs []uint
	eventMap := make(map[uint]models.Event)
	for _, follow := range follows {
		eventIDs = append(eventIDs, follow.EventID)
		eventMap[follow.EventID] = follow.Event
	}

	// 为每个事件获取最新的新闻
	var eventsNews []models.EventLatestNewsResponse

	// 使用子查询获取每个事件的最新新闻
	var latestNews []struct {
		EventID     uint
		NewsID      uint
		NewsTitle   string
		NewsSummary string
		NewsSource  string
		NewsAuthor  string
		NewsLink    string
		PublishedAt time.Time
		UpdatedAt   time.Time
	}

	// 构建复杂查询：为每个事件找到最新的新闻
	subQuery := s.db.Table("news").
		Select("belonged_event_id, MAX(published_at) as max_published").
		Where("belonged_event_id IN (?) AND is_active = true", eventIDs).
		Group("belonged_event_id")

	if err := s.db.Table("news").
		Select("news.belonged_event_id as event_id, news.id as news_id, news.title as news_title, news.summary as news_summary, news.source as news_source, news.author as news_author, news.link as news_link, news.published_at, news.updated_at").
		Joins("INNER JOIN (?) latest ON news.belonged_event_id = latest.belonged_event_id AND news.published_at = latest.max_published", subQuery).
		Where("news.is_active = true").
		Order("news.published_at DESC").
		Limit(limit).
		Scan(&latestNews).Error; err != nil {
		return nil, err
	}

	// 构建响应数据
	for _, news := range latestNews {
		if event, exists := eventMap[news.EventID]; exists {
			eventsNews = append(eventsNews, models.EventLatestNewsResponse{
				EventID:     news.EventID,
				EventTitle:  event.Title,
				EventStatus: event.Status,
				NewsID:      news.NewsID,
				NewsTitle:   news.NewsTitle,
				NewsSummary: news.NewsSummary,
				NewsSource:  news.NewsSource,
				NewsAuthor:  news.NewsAuthor,
				NewsLink:    news.NewsLink,
				PublishedAt: news.PublishedAt,
				UpdatedAt:   news.UpdatedAt,
			})
		}
	}

	return &models.FollowedEventsNewsResponse{
		TotalEvents: len(follows),
		EventsNews:  eventsNews,
	}, nil
}

// GetEventLatestNewsByEventIDs 根据事件ID列表获取每个事件的最新新闻（辅助方法）
func (s *MessageService) GetEventLatestNewsByEventIDs(eventIDs []uint, limit int) ([]models.EventLatestNewsResponse, error) {
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	if len(eventIDs) == 0 {
		return []models.EventLatestNewsResponse{}, nil
	}

	var results []struct {
		EventID     uint      `json:"event_id"`
		EventTitle  string    `json:"event_title"`
		EventStatus string    `json:"event_status"`
		NewsID      uint      `json:"news_id"`
		NewsTitle   string    `json:"news_title"`
		NewsSummary string    `json:"news_summary"`
		NewsSource  string    `json:"news_source"`
		NewsAuthor  string    `json:"news_author"`
		NewsLink    string    `json:"news_link"`
		PublishedAt time.Time `json:"published_at"`
		UpdatedAt   time.Time `json:"updated_at"`
	}

	// 使用窗口函数获取每个事件的最新新闻
	query := `
		SELECT 
			e.id as event_id,
			e.title as event_title,
			e.status as event_status,
			n.id as news_id,
			n.title as news_title,
			n.summary as news_summary,
			n.source as news_source,
			n.author as news_author,
			n.link as news_link,
			n.published_at,
			n.updated_at
		FROM events e
		LEFT JOIN (
			SELECT 
				*,
				ROW_NUMBER() OVER (PARTITION BY belonged_event_id ORDER BY published_at DESC) as rn
			FROM news 
			WHERE belonged_event_id IN (?) AND is_active = true
		) n ON e.id = n.belonged_event_id AND n.rn = 1
		WHERE e.id IN (?)
		ORDER BY n.published_at DESC
		LIMIT ?
	`

	if err := s.db.Raw(query, eventIDs, eventIDs, limit).Scan(&results).Error; err != nil {
		return nil, err
	}

	var eventsNews []models.EventLatestNewsResponse
	for _, result := range results {
		eventsNews = append(eventsNews, models.EventLatestNewsResponse{
			EventID:     result.EventID,
			EventTitle:  result.EventTitle,
			EventStatus: result.EventStatus,
			NewsID:      result.NewsID,
			NewsTitle:   result.NewsTitle,
			NewsSummary: result.NewsSummary,
			NewsSource:  result.NewsSource,
			NewsAuthor:  result.NewsAuthor,
			NewsLink:    result.NewsLink,
			PublishedAt: result.PublishedAt,
			UpdatedAt:   result.UpdatedAt,
		})
	}

	return eventsNews, nil
}

// GetFollowedEventsRecentNews 获取用户关注事件的最近新闻（用于通知）
func (s *MessageService) GetFollowedEventsRecentNews(userID uint, hours int) ([]models.EventLatestNewsResponse, error) {
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	// 设置默认时间范围为24小时
	if hours <= 0 {
		hours = 24
	}

	// 获取用户关注的事件列表
	var follows []models.Follow
	if err := s.db.Preload("Event").Where("user_id = ?", userID).Find(&follows).Error; err != nil {
		return nil, err
	}

	if len(follows) == 0 {
		return []models.EventLatestNewsResponse{}, nil
	}

	// 提取事件ID列表
	var eventIDs []uint
	eventMap := make(map[uint]models.Event)
	for _, follow := range follows {
		eventIDs = append(eventIDs, follow.EventID)
		eventMap[follow.EventID] = follow.Event
	}

	// 获取指定时间范围内的新闻
	cutoffTime := time.Now().Add(-time.Duration(hours) * time.Hour)

	var recentNews []struct {
		EventID     uint      `json:"event_id"`
		NewsID      uint      `json:"news_id"`
		NewsTitle   string    `json:"news_title"`
		NewsSummary string    `json:"news_summary"`
		NewsSource  string    `json:"news_source"`
		NewsAuthor  string    `json:"news_author"`
		NewsLink    string    `json:"news_link"`
		PublishedAt time.Time `json:"published_at"`
		UpdatedAt   time.Time `json:"updated_at"`
	}

	// 查询最近的新闻
	if err := s.db.Table("news").
		Select("belonged_event_id as event_id, id as news_id, title as news_title, summary as news_summary, source as news_source, author as news_author, link as news_link, published_at, updated_at").
		Where("belonged_event_id IN (?) AND is_active = true AND published_at >= ?", eventIDs, cutoffTime).
		Order("published_at DESC").
		Scan(&recentNews).Error; err != nil {
		return nil, err
	}

	// 构建响应数据
	var eventsNews []models.EventLatestNewsResponse
	for _, news := range recentNews {
		if event, exists := eventMap[news.EventID]; exists {
			eventsNews = append(eventsNews, models.EventLatestNewsResponse{
				EventID:     news.EventID,
				EventTitle:  event.Title,
				EventStatus: event.Status,
				NewsID:      news.NewsID,
				NewsTitle:   news.NewsTitle,
				NewsSummary: news.NewsSummary,
				NewsSource:  news.NewsSource,
				NewsAuthor:  news.NewsAuthor,
				NewsLink:    news.NewsLink,
				PublishedAt: news.PublishedAt,
				UpdatedAt:   news.UpdatedAt,
			})
		}
	}

	return eventsNews, nil
}

// CreateNewsUpdateNotifications 为用户关注的事件创建新闻更新通知
func (s *MessageService) CreateNewsUpdateNotifications(userID uint, hours int) error {
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	// 获取最近的新闻
	recentNews, err := s.GetFollowedEventsRecentNews(userID, hours)
	if err != nil {
		return err
	}

	if len(recentNews) == 0 {
		return nil // 没有新的新闻
	}

	// 为每条新闻创建通知消息
	for _, news := range recentNews {
		// 检查是否已经为此新闻创建过通知
		var existingMessage models.Message
		err := s.db.Where("user_id = ? AND type = ? AND related_type = ? AND related_id = ?",
			userID, "news_update", "news", news.NewsID).First(&existingMessage).Error

		if err == nil {
			// 已存在通知，跳过
			continue
		}

		// 创建新的通知消息
		title := "关注事件有新动态"
		content := fmt.Sprintf("您关注的事件「%s」有新的相关新闻：%s", news.EventTitle, news.NewsTitle)

		err = s.CreateMessage(
			userID,
			"news_update",
			title,
			content,
			"news",
			news.NewsID,
			nil, // 系统消息
		)

		if err != nil {
			log.Printf("Failed to create news update notification for user %d, news %d: %v", userID, news.NewsID, err)
			continue
		}
	}

	return nil
}
