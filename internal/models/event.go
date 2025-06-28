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
	LikeCount    int64   `json:"like_count" gorm:"default:0"`          // 点赞数
	CommentCount int64   `json:"comment_count" gorm:"default:0"`       // 评论数
	ShareCount   int64   `json:"share_count" gorm:"default:0"`         // 分享数
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
	LikeCount    int64     `json:"like_count"`
	CommentCount int64     `json:"comment_count"`
	ShareCount   int64     `json:"share_count"`
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

// TagResponse 标签响应结构
type TagResponse struct {
	Tag      string `json:"tag"`
	Count    int    `json:"count"`
	Category string `json:"category"`
}

// TrendingEventResponse 趋势事件响应结构
type TrendingEventResponse struct {
	ID             uint      `json:"id"`
	Title          string    `json:"title"`
	Category       string    `json:"category"`
	HotnessScore   float64   `json:"hotness_score"`
	TrendScore     float64   `json:"trend_score"`
	ViewGrowthRate float64   `json:"view_growth_rate"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
}

// HotnessFactors 热度计算因子
type HotnessFactors struct {
	ViewWeight    float64 `json:"view_weight"`    // 浏览量权重
	LikeWeight    float64 `json:"like_weight"`    // 点赞权重
	CommentWeight float64 `json:"comment_weight"` // 评论权重
	ShareWeight   float64 `json:"share_weight"`   // 分享权重
	TimeWeight    float64 `json:"time_weight"`    // 时间因素权重
}

// CalculationDetails 热度计算详情
type CalculationDetails struct {
	ViewScore    float64 `json:"view_score"`
	LikeScore    float64 `json:"like_score"`
	CommentScore float64 `json:"comment_score"`
	ShareScore   float64 `json:"share_score"`
	TimeScore    float64 `json:"time_score"`
	FinalScore   float64 `json:"final_score"`
}

// HotnessCalculationResult 热度计算结果
type HotnessCalculationResult struct {
	ID                 uint               `json:"id"`
	HotnessScore       float64            `json:"hotness_score"`
	PreviousScore      float64            `json:"previous_score"`
	CalculationDetails CalculationDetails `json:"calculation_details"`
	UpdatedAt          time.Time          `json:"updated_at"`
}

// UpdateTagsRequest 更新标签请求
type UpdateTagsRequest struct {
	Tags      []string `json:"tags" binding:"required"`
	Operation string   `json:"operation" binding:"omitempty,oneof=replace add remove"`
}

// UpdateHotnessRequest 更新热度请求
type UpdateHotnessRequest struct {
	HotnessScore  *float64        `json:"hotness_score" binding:"omitempty,min=0,max=10"`
	AutoCalculate *bool           `json:"auto_calculate"`
	Factors       *HotnessFactors `json:"factors"`
}

// LikeActionRequest 点赞操作请求
type LikeActionRequest struct {
	Action string `json:"action" binding:"required,oneof=like unlike"` // like or unlike
}

// InteractionStatsResponse 交互统计响应
type InteractionStatsResponse struct {
	EventID      uint    `json:"event_id"`
	ViewCount    int64   `json:"view_count"`
	LikeCount    int64   `json:"like_count"`
	CommentCount int64   `json:"comment_count"`
	ShareCount   int64   `json:"share_count"`
	HotnessScore float64 `json:"hotness_score"`
}
