package models

import (
	"time"

	"gorm.io/gorm"
)

// NewsType 新闻类型枚举
type NewsType string

const (
	NewsTypeManual NewsType = "manual" // 手动创建的新闻
	NewsTypeRSS    NewsType = "rss"    // RSS抓取的新闻
)

// News 对应数据库中的 'news' 表
type News struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Title       string    `json:"title" gorm:"not null;type:varchar(500)"` // 新闻标题，扩展长度支持RSS
	Content     string    `json:"content" gorm:"type:text"`                // 正文内容，RSS新闻可能为空
	Summary     string    `json:"summary" gorm:"type:text"`                // AI摘要（可选），使用TEXT类型
	Description string    `json:"description" gorm:"type:text"`            // RSS描述字段
	Source      string    `json:"source" gorm:"type:varchar(100)"`         // 新闻来源，限制长度为100
	Category    string    `json:"category" gorm:"type:varchar(100)"`       // 分类，扩展长度
	PublishedAt time.Time `json:"published_at" gorm:"not null"`            // 发布时间
	CreatedBy   *uint     `json:"created_by" gorm:"index"`                 // 创建者 (管理员ID)，RSS新闻可为空
	IsActive    bool      `json:"is_active" gorm:"default:true"`           // 是否展示，默认true

	// 事件关联字段
	BelongedEventID *uint `json:"belonged_event_id" gorm:"index"` // 关联的事件ID

	// RSS相关字段
	SourceType  NewsType `json:"source_type" gorm:"type:varchar(20);default:'manual';index"` // 新闻类型
	RSSSourceID *uint    `json:"rss_source_id" gorm:"index"`                                 // RSS源ID
	Link        string   `json:"link" gorm:"type:varchar(1000);index"`                       // 原文链接
	GUID        string   `json:"guid" gorm:"type:varchar(500);index"`                        // RSS GUID
	Author      string   `json:"author" gorm:"type:varchar(100)"`                            // 作者
	ImageURL    string   `json:"image_url" gorm:"type:varchar(1000)"`                        // 图片URL
	Tags        string   `json:"tags" gorm:"type:text"`                                      // 标签（JSON字符串）
	Language    string   `json:"language" gorm:"type:varchar(10);default:'zh'"`              // 语言

	// 统计字段
	ViewCount    int64   `json:"view_count" gorm:"default:0"`          // 浏览次数
	LikeCount    int64   `json:"like_count" gorm:"default:0"`          // 点赞数
	CommentCount int64   `json:"comment_count" gorm:"default:0"`       // 评论数
	ShareCount   int64   `json:"share_count" gorm:"default:0"`         // 分享数
	HotnessScore float64 `json:"hotness_score" gorm:"default:0;index"` // 热度分值

	// 状态字段
	Status      string `json:"status" gorm:"type:varchar(20);default:'published';index"` // 状态
	IsProcessed bool   `json:"is_processed" gorm:"default:false"`                        // 是否已处理

	// GORM 自动维护的时间戳
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"` // 软删除，JSON输出时忽略，数据库中创建索引

	// GORM 关系定义
	Creator       *User      `json:"creator,omitempty" gorm:"foreignKey:CreatedBy;references:ID"`
	RSSSource     *RSSSource `json:"rss_source,omitempty" gorm:"foreignKey:RSSSourceID;references:ID"`
	BelongedEvent *Event     `json:"belonged_event,omitempty" gorm:"foreignKey:BelongedEventID;references:ID"` // 关联的事件
}

// TableName 为 News 模型指定数据库表名为 'news'

// NewsResponse 用于向前端返回新闻信息时，过滤掉敏感或不需要的字段
type NewsResponse struct {
	ID              uint               `json:"id"`
	Title           string             `json:"title"`
	Content         string             `json:"content"`
	Summary         string             `json:"summary"`
	Description     string             `json:"description"`
	Source          string             `json:"source"`
	Category        string             `json:"category"`
	PublishedAt     time.Time          `json:"published_at"`
	CreatedBy       *uint              `json:"created_by"`
	IsActive        bool               `json:"is_active"`
	BelongedEventID *uint              `json:"belonged_event_id,omitempty"` // 关联的事件ID
	SourceType      NewsType           `json:"source_type"`
	RSSSourceID     *uint              `json:"rss_source_id,omitempty"`
	Link            string             `json:"link"`
	GUID            string             `json:"guid"`
	Author          string             `json:"author"`
	ImageURL        string             `json:"image_url"`
	Tags            string             `json:"tags"`
	Language        string             `json:"language"`
	ViewCount       int64              `json:"view_count"`
	LikeCount       int64              `json:"like_count"`
	CommentCount    int64              `json:"comment_count"`
	ShareCount      int64              `json:"share_count"`
	HotnessScore    float64            `json:"hotness_score"`
	Status          string             `json:"status"`
	IsProcessed     bool               `json:"is_processed"`
	CreatedAt       time.Time          `json:"created_at"`
	UpdatedAt       time.Time          `json:"updated_at"`
	RSSSource       *RSSSourceResponse `json:"rss_source,omitempty"`
}

// NewsCreateRequest 用于创建新闻时的请求体
type NewsCreateRequest struct {
	Title           string     `json:"title" binding:"required,min=5,max=255"` // 标题必填，限制长度
	Content         string     `json:"content" binding:"required"`             // 内容必填
	Summary         string     `json:"summary"`                                // 摘要可选
	Source          string     `json:"source" binding:"omitempty,max=100"`     // 来源可选，限制长度
	Category        string     `json:"category" binding:"omitempty,max=50"`    // 分类可选，限制长度
	PublishedAt     *time.Time `json:"published_at"`                           // 发布时间可选，使用指针允许为null
	IsActive        *bool      `json:"is_active"`                              // 是否展示可选，使用指针允许为null
	BelongedEventID *uint      `json:"belonged_event_id"`                      // 关联的事件ID，可选
}

// NewsUpdateRequest 用于更新新闻时的请求体
type NewsUpdateRequest struct {
	Title           string     `json:"title" binding:"omitempty,min=5,max=255"`
	Content         string     `json:"content" binding:"omitempty"`
	Summary         string     `json:"summary"`
	Source          string     `json:"source" binding:"omitempty,max=100"`
	Category        string     `json:"category" binding:"omitempty,max=50"`
	PublishedAt     *time.Time `json:"published_at"`
	IsActive        *bool      `json:"is_active"`
	BelongedEventID *uint      `json:"belonged_event_id"` // 关联的事件ID，可选
}

func (n *News) ToResponse() NewsResponse {
	response := NewsResponse{
		ID:              n.ID,
		Title:           n.Title,
		Content:         n.Content,
		Summary:         n.Summary,
		Description:     n.Description,
		Source:          n.Source,
		Category:        n.Category,
		PublishedAt:     n.PublishedAt,
		CreatedBy:       n.CreatedBy,
		IsActive:        n.IsActive,
		BelongedEventID: n.BelongedEventID, // 添加事件关联ID
		SourceType:      n.SourceType,
		RSSSourceID:     n.RSSSourceID,
		Link:            n.Link,
		GUID:            n.GUID,
		Author:          n.Author,
		ImageURL:        n.ImageURL,
		Tags:            n.Tags,
		Language:        n.Language,
		ViewCount:       n.ViewCount,
		LikeCount:       n.LikeCount,
		CommentCount:    n.CommentCount,
		ShareCount:      n.ShareCount,
		HotnessScore:    n.HotnessScore,
		Status:          n.Status,
		IsProcessed:     n.IsProcessed,
		CreatedAt:       n.CreatedAt,
		UpdatedAt:       n.UpdatedAt,
	}

	// 如果有RSS源关联，添加RSS源信息
	if n.RSSSource != nil {
		response.RSSSource = &RSSSourceResponse{
			ID:          n.RSSSource.ID,
			Name:        n.RSSSource.Name,
			URL:         n.RSSSource.URL,
			Category:    n.RSSSource.Category,
			Language:    n.RSSSource.Language,
			IsActive:    n.RSSSource.IsActive,
			LastFetched: n.RSSSource.LastFetched,
			FetchCount:  n.RSSSource.FetchCount,
			ErrorCount:  n.RSSSource.ErrorCount,
			Description: n.RSSSource.Description,
			Tags:        n.RSSSource.Tags,
			Priority:    n.RSSSource.Priority,
			UpdateFreq:  n.RSSSource.UpdateFreq,
			CreatedAt:   n.RSSSource.CreatedAt,
			UpdatedAt:   n.RSSSource.UpdatedAt,
		}
	}

	return response
}
func (News) TableName() string {
	return "news"
}
