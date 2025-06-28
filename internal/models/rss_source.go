package models

import (
	"time"

	"gorm.io/gorm"
)

// RSSSource RSS源配置
type RSSSource struct {
	gorm.Model
	Name        string    `json:"name" gorm:"type:varchar(100);not null"`                    // RSS源名称
	URL         string    `json:"url" gorm:"type:varchar(500);not null;uniqueIndex"`         // RSS URL
	Category    string    `json:"category" gorm:"type:varchar(50);not null;index"`           // 分类
	Language    string    `json:"language" gorm:"type:varchar(10);default:'zh'"`             // 语言
	IsActive    bool      `json:"is_active" gorm:"default:true"`                             // 是否启用
	LastFetched time.Time `json:"last_fetched" gorm:"default:null"`                          // 最后抓取时间
	FetchCount  int64     `json:"fetch_count" gorm:"default:0"`                              // 抓取次数
	ErrorCount  int64     `json:"error_count" gorm:"default:0"`                              // 错误次数
	Description string    `json:"description" gorm:"type:text"`                              // 描述
	Tags        string    `json:"tags" gorm:"type:text"`                                     // 标签（JSON字符串）
	Priority    int       `json:"priority" gorm:"default:1"`                                 // 优先级（1-10）
	UpdateFreq  int       `json:"update_freq" gorm:"default:60"`                             // 更新频率（分钟）
}

// NewsItem 新闻条目
type NewsItem struct {
	gorm.Model
	RSSSourceID uint      `json:"rss_source_id" gorm:"not null;index"`                       // RSS源ID
	Title       string    `json:"title" gorm:"type:varchar(500);not null;index"`             // 标题
	Link        string    `json:"link" gorm:"type:varchar(1000);not null;uniqueIndex"`       // 链接
	Description string    `json:"description" gorm:"type:text"`                              // 描述
	Content     string    `json:"content" gorm:"type:text"`                                  // 全文内容
	Author      string    `json:"author" gorm:"type:varchar(100)"`                           // 作者
	Category    string    `json:"category" gorm:"type:varchar(100);index"`                   // 分类
	Tags        string    `json:"tags" gorm:"type:text"`                                     // 标签（JSON字符串）
	PublishedAt time.Time `json:"published_at" gorm:"index"`                                 // 发布时间
	GUID        string    `json:"guid" gorm:"type:varchar(500);index"`                       // GUID
	ImageURL    string    `json:"image_url" gorm:"type:varchar(1000)"`                       // 图片URL
	
	// 统计字段
	ViewCount    int64   `json:"view_count" gorm:"default:0"`                               // 浏览次数
	LikeCount    int64   `json:"like_count" gorm:"default:0"`                               // 点赞数
	CommentCount int64   `json:"comment_count" gorm:"default:0"`                            // 评论数
	ShareCount   int64   `json:"share_count" gorm:"default:0"`                              // 分享数
	HotnessScore float64 `json:"hotness_score" gorm:"default:0;index"`                      // 热度分值
	
	// 状态字段
	Status       string `json:"status" gorm:"type:varchar(20);default:'published';index"`  // 状态：published, draft, archived
	IsProcessed  bool   `json:"is_processed" gorm:"default:false"`                         // 是否已处理
	
	// 关联
	RSSSource RSSSource `json:"rss_source,omitempty" gorm:"foreignKey:RSSSourceID"`
}

// RSS源响应结构
type RSSSourceResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Category    string    `json:"category"`
	Language    string    `json:"language"`
	IsActive    bool      `json:"is_active"`
	LastFetched time.Time `json:"last_fetched"`
	FetchCount  int64     `json:"fetch_count"`
	ErrorCount  int64     `json:"error_count"`
	Description string    `json:"description"`
	Tags        string    `json:"tags"`
	Priority    int       `json:"priority"`
	UpdateFreq  int       `json:"update_freq"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// 新闻条目响应结构
type NewsItemResponse struct {
	ID           uint      `json:"id"`
	RSSSourceID  uint      `json:"rss_source_id"`
	Title        string    `json:"title"`
	Link         string    `json:"link"`
	Description  string    `json:"description"`
	Content      string    `json:"content,omitempty"`
	Author       string    `json:"author"`
	Category     string    `json:"category"`
	Tags         string    `json:"tags"`
	PublishedAt  time.Time `json:"published_at"`
	GUID         string    `json:"guid"`
	ImageURL     string    `json:"image_url"`
	ViewCount    int64     `json:"view_count"`
	LikeCount    int64     `json:"like_count"`
	CommentCount int64     `json:"comment_count"`
	ShareCount   int64     `json:"share_count"`
	HotnessScore float64   `json:"hotness_score"`
	Status       string    `json:"status"`
	IsProcessed  bool      `json:"is_processed"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
	RSSSource    *RSSSourceResponse `json:"rss_source,omitempty"`
}

// 创建RSS源请求
type CreateRSSSourceRequest struct {
	Name        string   `json:"name" binding:"required,min=1,max=100"`
	URL         string   `json:"url" binding:"required,url,max=500"`
	Category    string   `json:"category" binding:"required,max=50"`
	Language    string   `json:"language" binding:"omitempty,max=10"`
	Description string   `json:"description" binding:"omitempty,max=1000"`
	Tags        []string `json:"tags"`
	Priority    int      `json:"priority" binding:"omitempty,min=1,max=10"`
	UpdateFreq  int      `json:"update_freq" binding:"omitempty,min=5,max=1440"`
}

// 更新RSS源请求
type UpdateRSSSourceRequest struct {
	Name        string   `json:"name" binding:"omitempty,min=1,max=100"`
	URL         string   `json:"url" binding:"omitempty,url,max=500"`
	Category    string   `json:"category" binding:"omitempty,max=50"`
	Language    string   `json:"language" binding:"omitempty,max=10"`
	IsActive    *bool    `json:"is_active"`
	Description string   `json:"description" binding:"omitempty,max=1000"`
	Tags        []string `json:"tags"`
	Priority    int      `json:"priority" binding:"omitempty,min=1,max=10"`
	UpdateFreq  int      `json:"update_freq" binding:"omitempty,min=5,max=1440"`
}

// 新闻查询请求
type NewsQueryRequest struct {
	RSSSourceID uint   `form:"rss_source_id"`
	Category    string `form:"category"`
	Status      string `form:"status"`
	Search      string `form:"search"`
	SortBy      string `form:"sort_by"` // published_at, hotness, views
	Page        int    `form:"page,default=1"`
	Limit       int    `form:"limit,default=10"`
	StartDate   string `form:"start_date"` // YYYY-MM-DD
	EndDate     string `form:"end_date"`   // YYYY-MM-DD
}

// 新闻列表响应
type NewsListResponse struct {
	Total int64              `json:"total"`
	News  []NewsItemResponse `json:"news"`
}

// RSS抓取统计
type RSSFetchStats struct {
	SourceID     uint      `json:"source_id"`
	SourceName   string    `json:"source_name"`
	TotalItems   int       `json:"total_items"`
	NewItems     int       `json:"new_items"`
	UpdatedItems int       `json:"updated_items"`
	ErrorItems   int       `json:"error_items"`
	FetchTime    time.Time `json:"fetch_time"`
	Duration     string    `json:"duration"`
}

// RSS抓取结果
type RSSFetchResult struct {
	Success bool            `json:"success"`
	Message string          `json:"message"`
	Stats   []RSSFetchStats `json:"stats"`
}