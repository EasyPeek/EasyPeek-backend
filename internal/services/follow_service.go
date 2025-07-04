package services

import (
	"errors"
	"fmt"

	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"gorm.io/gorm"
)

type FollowService struct {
	db *gorm.DB
}

// NewFollowService 创建关注服务实例
func NewFollowService() *FollowService {
	return &FollowService{
		db: database.GetDB(),
	}
}

// AddFollow 添加关注事件
func (s *FollowService) AddFollow(userID uint, eventID uint) error {
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	// 验证用户是否存在
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return err
	}

	// 验证事件是否存在
	var event models.Event
	if err := s.db.First(&event, eventID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("event not found")
		}
		return err
	}

	// 检查是否已经关注
	var existingFollow models.Follow
	err := s.db.Where("user_id = ? AND event_id = ?", userID, eventID).First(&existingFollow).Error
	if err == nil {
		return errors.New("already following this event")
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	// 创建关注记录
	follow := &models.Follow{
		UserID:       userID,
		EventID:      eventID,
		FollowType:   "event", // 设置关注类型为事件
		FollowTarget: "event", // 设置关注目标为事件
	}

	err = s.db.Create(follow).Error
	if err != nil {
		return err
	}

	// 创建关注成功的消息通知
	messageService := NewMessageService()
	title := "关注事件成功"
	content := fmt.Sprintf("您已成功关注事件「%s」，我们将为您推送相关的最新动态。", event.Title)

	// 创建关注消息（忽略错误，不影响关注操作）
	_ = messageService.CreateMessage(
		userID,
		"follow",
		title,
		content,
		"event",
		eventID,
		nil, // 系统消息
	)

	return nil
}

// RemoveFollow 取消关注事件
func (s *FollowService) RemoveFollow(userID uint, eventID uint) error {
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	result := s.db.Where("user_id = ? AND event_id = ?", userID, eventID).Delete(&models.Follow{})
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return errors.New("follow relationship not found")
	}
	return nil
}

// GetUserFollows 获取用户关注的事件列表
func (s *FollowService) GetUserFollows(userID uint) ([]models.FollowResponse, error) {
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	var follows []models.Follow

	// 构建查询，预加载事件信息
	if err := s.db.Preload("Event").Where("user_id = ?", userID).Order("created_at DESC").Find(&follows).Error; err != nil {
		return nil, err
	}

	// 转换为响应格式
	var responses []models.FollowResponse
	for _, follow := range follows {
		responses = append(responses, follow.ToResponse())
	}

	return responses, nil
}

// GetUserFollowsWithPagination 获取用户关注的事件列表（支持分页）
func (s *FollowService) GetUserFollowsWithPagination(userID uint, page, pageSize int) ([]models.FollowResponse, int64, error) {
	if s.db == nil {
		return nil, 0, errors.New("database connection not initialized")
	}

	// 获取总数
	var total int64
	if err := s.db.Model(&models.Follow{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 计算偏移量
	offset := (page - 1) * pageSize

	var follows []models.Follow
	// 构建查询，预加载事件信息，支持分页
	if err := s.db.Preload("Event").Where("user_id = ?", userID).Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&follows).Error; err != nil {
		return nil, 0, err
	}

	// 转换为响应格式
	var responses []models.FollowResponse
	for _, follow := range follows {
		responses = append(responses, follow.ToResponse())
	}

	return responses, total, nil
}

// CheckFollow 检查是否已关注事件
func (s *FollowService) CheckFollow(userID uint, eventID uint) (bool, error) {
	if s.db == nil {
		return false, errors.New("database connection not initialized")
	}

	var count int64
	err := s.db.Model(&models.Follow{}).Where("user_id = ? AND event_id = ?", userID, eventID).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, nil
}

// GetFollowStats 获取用户关注统计
func (s *FollowService) GetFollowStats(userID uint) (*models.FollowStats, error) {
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	var follows []models.Follow
	if err := s.db.Preload("Event").Where("user_id = ?", userID).Find(&follows).Error; err != nil {
		return nil, err
	}

	stats := &models.FollowStats{
		FollowedEvents: []models.EventResponse{},
		TotalCount:     len(follows),
	}

	for _, follow := range follows {
		eventResponse := models.EventResponse{
			ID:          follow.Event.ID,
			Title:       follow.Event.Title,
			Description: follow.Event.Description,
			StartTime:   follow.Event.StartTime,
			EndTime:     follow.Event.EndTime,
			Location:    follow.Event.Location,
			Status:      follow.Event.Status,
			Category:    follow.Event.Category,
		}
		stats.FollowedEvents = append(stats.FollowedEvents, eventResponse)
	}

	return stats, nil
}

// GetFollowersByEvent 获取关注某个事件的用户列表（用于消息通知）
func (s *FollowService) GetFollowersByEvent(eventID uint) ([]uint, error) {
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	var follows []models.Follow
	if err := s.db.Where("event_id = ?", eventID).Find(&follows).Error; err != nil {
		return nil, err
	}

	var userIDs []uint
	for _, follow := range follows {
		userIDs = append(userIDs, follow.UserID)
	}

	return userIDs, nil
}

// GetAvailableEvents 获取可关注的事件列表
func (s *FollowService) GetAvailableEvents(page, pageSize int) ([]models.Event, int64, error) {
	if s.db == nil {
		return nil, 0, errors.New("database connection not initialized")
	}

	var events []models.Event
	var total int64

	// 统计总数
	if err := s.db.Model(&models.Event{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	// 分页查询
	offset := (page - 1) * pageSize
	if err := s.db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&events).Error; err != nil {
		return nil, 0, err
	}

	return events, total, nil
}
