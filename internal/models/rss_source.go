package models

import (
	"time"

	"gorm.io/gorm"
)

// RSSSource
type RSSSource struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	Name        string         `json:"name" gorm:"type:varchar(100);not null"`
	URL         string         `json:"url" gorm:"type:varchar(500);not null;uniqueIndex"`
	Description string         `json:"description" gorm:"type:text"`
	Category    string         `json:"category" gorm:"type:varchar(50);not null;index"`
	Language    string         `json:"language" gorm:"type:varchar(10);default:'zh'"`
	IsActive    bool           `json:"is_active" gorm:"default:true"`
	FetchCount  int64          `json:"fetch_count" gorm:"default:0"`
	ErrorCount  int64          `json:"error_count" gorm:"default:0"`
	Tags        string         `json:"tags" gorm:"type:text"`
	Priority    int            `json:"priority" gorm:"default:1"`
	UpdateFreq  int            `json:"update_freq" gorm:"default:60"`
	LastFetched time.Time      `json:"last_fetched" gorm:"default:null"`
	CreatedAt   time.Time      `gorm:"autoCreateTime"`
	UpdatedAt   time.Time      `gorm:"autoUpdateTime"`
	DeletedAt   gorm.DeletedAt `gorm:"index"`
}

// RSS源响应结构
type RSSSourceResponse struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	URL         string    `json:"url"`
	Description string    `json:"description"`
	Category    string    `json:"category"`
	Language    string    `json:"language"`
	IsActive    bool      `json:"is_active"`
	FetchCount  int64     `json:"fetch_count"`
	ErrorCount  int64     `json:"error_count"`
	Tags        string    `json:"tags"`
	Priority    int       `json:"priority"`
	UpdateFreq  int       `json:"update_freq"`
	LastFetched time.Time `json:"last_fetched"`
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
}

// 更新RSS源请求
type UpdateRSSSourceRequest struct {
	Name        string   `json:"name" binding:"omitempty,min=1,max=100"`
	URL         string   `json:"url" binding:"omitempty,url,max=500"`
	Category    string   `json:"category" binding:"omitempty,max=50"`
	Description string   `json:"description" binding:"omitempty,max=1000"`
	Language    string   `json:"language" binding:"omitempty,max=10"`
	IsActive    *bool    `json:"is_active"`
	Tags        []string `json:"tags"`
	Priority    int      `json:"priority" binding:"omitempty,min=1,max=10"`
	UpdateFreq  int      `json:"update_freq" binding:"omitempty,min=5,max=1440"`
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

func (r *RSSSource) ToResponse() *RSSSourceResponse {
	return &RSSSourceResponse{
		ID:          r.ID,
		Name:        r.Name,
		URL:         r.URL,
		Category:    r.Category,
		Language:    r.Language,
		IsActive:    r.IsActive,
		LastFetched: r.LastFetched,
		FetchCount:  r.FetchCount,
		ErrorCount:  r.ErrorCount,
		Description: r.Description,
		Tags:        r.Tags,
		Priority:    r.Priority,
		UpdateFreq:  r.UpdateFreq,
		CreatedAt:   r.CreatedAt,
		UpdatedAt:   r.UpdatedAt,
	}
}
