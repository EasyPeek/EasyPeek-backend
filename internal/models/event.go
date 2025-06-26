package models

import (
	"time"

	"gorm.io/gorm"
)

// Event 事件模型
type Event struct {
	gorm.Model
	Title       string    `json:"title" gorm:"type:varchar(200);not null;index"`
	Description string    `json:"description" gorm:"type:text"`
	Content     string    `json:"content" gorm:"type:text"` // 新增：事件全文内容
	StartTime   time.Time `json:"start_time" gorm:"not null"`
	EndTime     time.Time `json:"end_time" gorm:"not null"`
	Location    string    `json:"location" gorm:"type:varchar(255)"`
	Status      string    `json:"status" gorm:"type:varchar(20);default:'进行中';index"` // 进行中、已结束
	CreatedBy   uint      `json:"created_by" gorm:"not null"`
	Image       string    `json:"image" gorm:"type:varchar(255)"`

	// 基础分类字段
	Category     string `json:"category" gorm:"type:varchar(50);index"` // 事件分类
	Tags         string `json:"tags" gorm:"type:text"`                  // 事件标签（JSON字符串）
	Source       string `json:"source" gorm:"type:varchar(100)"`        // 新闻来源
	Author       string `json:"author" gorm:"type:varchar(100)"`        // 作者信息
	RelatedLinks string `json:"related_links" gorm:"type:text"`         // 相关链接（JSON字符串）

	// 统计字段
	ViewCount    int64   `json:"view_count" gorm:"default:0"`          // 浏览次数
	HotnessScore float64 `json:"hotness_score" gorm:"default:0;index"` // 事件热度分值
}

// EventResponse 事件响应结构
type EventResponse struct {
	ID           uint      `json:"id"`
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Content      string    `json:"content,omitempty"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Location     string    `json:"location"`
	Status       string    `json:"status"`
	CreatedBy    uint      `json:"created_by"`
	Image        string    `json:"image,omitempty"`
	Category     string    `json:"category"`
	Tags         string    `json:"tags,omitempty"`
	Source       string    `json:"source,omitempty"`
	Author       string    `json:"author,omitempty"`
	RelatedLinks string    `json:"related_links,omitempty"`
	ViewCount    int64     `json:"view_count"`
	HotnessScore float64   `json:"hotness_score"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// EventListResponse 事件列表响应结构
type EventListResponse struct {
	Total  int64           `json:"total"`
	Events []EventResponse `json:"events"`
}

// EventQueryRequest 事件查询请求
type EventQueryRequest struct {
	Status   string `form:"status"`   // 状态筛选
	Category string `form:"category"` // 分类筛选
	Search   string `form:"search"`   // 搜索关键词
	SortBy   string `form:"sort_by"`  // 排序方式: time, hotness, views
	Page     int    `form:"page,default=1"`
	Limit    int    `form:"limit,default=10"`
}

// CreateEventRequest 创建事件请求
type CreateEventRequest struct {
	Title        string    `json:"title" binding:"required,min=1,max=200"`
	Description  string    `json:"description" binding:"max=1000"`
	Content      string    `json:"content" binding:"required"`
	StartTime    time.Time `json:"start_time" binding:"required"`
	EndTime      time.Time `json:"end_time" binding:"required,gtfield=StartTime"`
	Location     string    `json:"location" binding:"required,max=255"`
	Category     string    `json:"category" binding:"required,max=50"`
	Tags         []string  `json:"tags"`
	Source       string    `json:"source" binding:"max=100"`
	Author       string    `json:"author" binding:"max=100"`
	RelatedLinks []string  `json:"related_links"`
	Image        string    `json:"image"`
}

// UpdateEventRequest 更新事件请求
type UpdateEventRequest struct {
	Title        string    `json:"title" binding:"omitempty,min=1,max=200"`
	Description  string    `json:"description" binding:"omitempty,max=1000"`
	Content      string    `json:"content"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time" binding:"omitempty,gtfield=StartTime"`
	Location     string    `json:"location" binding:"omitempty,max=255"`
	Status       string    `json:"status" binding:"omitempty,oneof=进行中 已结束"`
	Category     string    `json:"category" binding:"omitempty,max=50"`
	Tags         []string  `json:"tags"`
	Source       string    `json:"source" binding:"omitempty,max=100"`
	Author       string    `json:"author" binding:"omitempty,max=100"`
	RelatedLinks []string  `json:"related_links"`
	Image        string    `json:"image"`
}
