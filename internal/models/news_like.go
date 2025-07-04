package models

import (
	"time"

	"gorm.io/gorm"
)

// NewsLike 新闻点赞记录
type NewsLike struct {
	ID     uint      `json:"id" gorm:"primaryKey"`
	NewsID uint      `json:"news_id" gorm:"not null;index"`
	UserID uint      `json:"user_id" gorm:"not null;index"`
	LikeAt time.Time `json:"like_at" gorm:"not null;default:CURRENT_TIMESTAMP"`

	// GORM 自动维护的时间戳
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	News *News `json:"news,omitempty" gorm:"foreignKey:NewsID;references:ID"`
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID"`
}

// TableName 指定表名
func (NewsLike) TableName() string {
	return "news_likes"
}
