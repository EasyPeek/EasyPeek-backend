package models

import (
	"time"

	"gorm.io/gorm"
)

// Event 事件模型
type Event struct {
	gorm.Model
	Title       string    `json:"title" gorm:"type:varchar(100);not null"`
	Description string    `json:"description" gorm:"type:text"`
	StartTime   time.Time `json:"start_time" gorm:"not null"`
	EndTime     time.Time `json:"end_time" gorm:"not null"`
	Location    string    `json:"location" gorm:"type:varchar(255)"`
	Status      string    `json:"status" gorm:"type:varchar(20);default:'进行中'"` // 进行中、已结束
	CreatedBy   uint      `json:"created_by" gorm:"not null"`
	Image       string    `json:"image" gorm:"type:varchar(255)"`
}

// EventResponse 事件响应结构
type EventResponse struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
	Location    string    `json:"location"`
	Status      string    `json:"status"`
	CreatedBy   uint      `json:"created_by"`
	Image       string    `json:"image,omitempty"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// EventListResponse 事件列表响应结构
type EventListResponse struct {
	Total  int64           `json:"total"`
	Events []EventResponse `json:"events"`
}

// EventQueryRequest 事件查询请求
type EventQueryRequest struct {
	Status string `form:"status"`
	Page   int    `form:"page,default=1"`
	Limit  int    `form:"limit,default=10"`
	Search string `form:"search"`
}

// CreateEventRequest 创建事件请求
type CreateEventRequest struct {
	Title       string    `json:"title" binding:"required,min=1,max=100"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time" binding:"required"`
	EndTime     time.Time `json:"end_time" binding:"required,gtfield=StartTime"`
	Location    string    `json:"location" binding:"required"`
	Image       string    `json:"image"`
}

// UpdateEventRequest 更新事件请求
type UpdateEventRequest struct {
	Title       string    `json:"title" binding:"omitempty,min=1,max=100"`
	Description string    `json:"description"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time" binding:"omitempty,gtfield=StartTime"`
	Location    string    `json:"location"`
	Status      string    `json:"status" binding:"omitempty,oneof=进行中 已结束"`
	Image       string    `json:"image"`
}
