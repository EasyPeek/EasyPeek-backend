package models

import (
	"time"

	"gorm.io/gorm"
)

// News 对应数据库中的 'news' 表
type News struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Title       string    `json:"title" gorm:"not null;type:varchar(255)"` // 新闻标题，限制长度为255
	Content     string    `json:"content" gorm:"not null;type:text"`       // 正文内容，使用TEXT类型存储长文本
	Summary     string    `json:"summary" gorm:"type:text"`                // AI摘要（可选），使用TEXT类型
	Source      string    `json:"source" gorm:"type:varchar(100)"`         // 新闻来源，限制长度为100
	Category    string    `json:"category" gorm:"type:varchar(50)"`        // 分类，限制长度为50
	PublishedAt time.Time `json:"published_at" gorm:"not null"`            // 发布时间
	CreatedBy   uint      `json:"created_by" gorm:"not null"`              // 创建者 (管理员ID), 对应 users 表的 ID
	IsActive    bool      `json:"is_active" gorm:"default:true"`           // 是否展示，默认true

	// GORM 自动维护的时间戳
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"` // 软删除，JSON输出时忽略，数据库中创建索引

	// GORM 关系定义 (可选但推荐): 新闻的创建者 (假设 User 模型存在)
	Creator User `gorm:"foreignKey:CreatedBy;references:ID"` // CreatedBy 是 News 的外键，引用 User 的 ID
}

// TableName 为 News 模型指定数据库表名为 'news'

// NewsResponse 用于向前端返回新闻信息时，过滤掉敏感或不需要的字段
type NewsResponse struct {
	ID          uint      `json:"id"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	Summary     string    `json:"summary"`
	Source      string    `json:"source"`
	Category    string    `json:"category"`
	PublishedAt time.Time `json:"published_at"`
	CreatedBy   uint      `json:"created_by"`
	IsActive    bool      `json:"is_active"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	// 注意: CreatedBy 字段可能需要额外处理，比如返回创建者的用户名而非ID
	// 如果需要，可以在这里嵌套 UserResponse 或自定义
}

// NewsCreateRequest 用于创建新闻时的请求体
type NewsCreateRequest struct {
	Title       string     `json:"title" binding:"required,min=5,max=255"` // 标题必填，限制长度
	Content     string     `json:"content" binding:"required"`             // 内容必填
	Summary     string     `json:"summary"`                                // 摘要可选
	Source      string     `json:"source" binding:"omitempty,max=100"`     // 来源可选，限制长度
	Category    string     `json:"category" binding:"omitempty,max=50"`    // 分类可选，限制长度
	PublishedAt *time.Time `json:"published_at"`                           // 发布时间可选，使用指针允许为null
	IsActive    *bool      `json:"is_active"`                              // 是否展示可选，使用指针允许为null
}

// NewsUpdateRequest 用于更新新闻时的请求体
type NewsUpdateRequest struct {
	Title       string     `json:"title" binding:"omitempty,min=5,max=255"`
	Content     string     `json:"content" binding:"omitempty"`
	Summary     string     `json:"summary"`
	Source      string     `json:"source" binding:"omitempty,max=100"`
	Category    string     `json:"category" binding:"omitempty,max=50"`
	PublishedAt *time.Time `json:"published_at"`
	IsActive    *bool      `json:"is_active"`
}

func (n *News) ToResponse() NewsResponse {
	return NewsResponse{
		ID:          n.ID,
		Title:       n.Title,
		Content:     n.Content,
		Summary:     n.Summary,
		Source:      n.Source,
		Category:    n.Category,
		PublishedAt: n.PublishedAt,
		CreatedBy:   n.CreatedBy,
		IsActive:    n.IsActive,
		CreatedAt:   n.CreatedAt,
		UpdatedAt:   n.UpdatedAt,
	}
}
func (News) TableName() string {
	return "news"
}
