package services

import (
	"errors"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"gorm.io/gorm"
)

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

	// 添加搜索条件
	if query.Search != "" {
		searchTerm := "%" + query.Search + "%"
		db = db.Where("title LIKE ? OR description LIKE ? OR location LIKE ?", searchTerm, searchTerm, searchTerm)
	}

	// 获取总数
	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	// 分页
	offset := (query.Page - 1) * query.Limit
	err := db.Order("created_at desc").Offset(offset).Limit(query.Limit).Find(&events).Error
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

// CreateEvent 创建事件
func (s *EventService) CreateEvent(req *models.CreateEventRequest) (*models.EventResponse, error) {
	// 检查时间
	if req.EndTime.Before(req.StartTime) {
		return nil, errors.New("end time must be after start time")
	}

	event := models.Event{
		Title:       req.Title,
		Description: req.Description,
		StartTime:   req.StartTime,
		EndTime:     req.EndTime,
		Location:    req.Location,
		Status:      "进行中",
		CreatedBy:   1, // 这里应该从当前用户上下文中获取，示例代码使用固定值
		Image:       req.Image,
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
		ID:          event.ID,
		Title:       event.Title,
		Description: event.Description,
		StartTime:   event.StartTime,
		EndTime:     event.EndTime,
		Location:    event.Location,
		Status:      event.Status,
		CreatedBy:   event.CreatedBy,
		Image:       event.Image,
		CreatedAt:   event.CreatedAt,
		UpdatedAt:   event.UpdatedAt,
	}
}
