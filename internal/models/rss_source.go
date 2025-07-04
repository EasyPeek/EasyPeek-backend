package models

import (
	"time"

	"gorm.io/gorm"
)

// RSSSource RSS源配置
type RSSSource struct {
	gorm.Model
	Name        string    `json:"name" gorm:"type:varchar(100);not null"`            // RSS源名称
	URL         string    `json:"url" gorm:"type:varchar(500);not null;uniqueIndex"` // RSS URL
	Category    string    `json:"category" gorm:"type:varchar(50);not null;index"`   // 分类
	Language    string    `json:"language" gorm:"type:varchar(10);default:'zh'"`     // 语言
	IsActive    bool      `json:"is_active" gorm:"default:true"`                     // 是否启用
	LastFetched time.Time `json:"last_fetched" gorm:"default:null"`                  // 最后抓取时间
	FetchCount  int64     `json:"fetch_count" gorm:"default:0"`                      // 抓取次数
	ErrorCount  int64     `json:"error_count" gorm:"default:0"`                      // 错误次数
	Description string    `json:"description" gorm:"type:text"`                      // 描述
	Tags        string    `json:"tags" gorm:"type:text"`                             // 标签（JSON字符串）
	Priority    int       `json:"priority" gorm:"default:1"`                         // 优先级（1-10）
	UpdateFreq  int       `json:"update_freq" gorm:"default:60"`                     // 更新频率（分钟）
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
	// 兼容前端字段名
	FetchInterval int `json:"fetch_interval" binding:"omitempty,min=5,max=1440"`
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
	// 兼容前端字段名
	FetchInterval int `json:"fetch_interval" binding:"omitempty,min=5,max=1440"`
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
