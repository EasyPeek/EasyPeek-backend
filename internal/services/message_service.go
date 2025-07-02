package services

import (
	"errors"

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
