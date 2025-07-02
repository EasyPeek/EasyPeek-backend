package models

import (
	"time"

	"gorm.io/gorm"
)

// Comment 对应数据库中的 'comments' 表
type Comment struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	NewsID    uint           `json:"news_id" gorm:"not null;index"`     // 新闻ID
	UserID    uint           `json:"user_id" gorm:"not null;index"`     // 发表评论的用户ID
	Content   string         `json:"content" gorm:"type:text;not null"` // 评论的内容
	CreatedAt time.Time      `json:"created_at"`                        // 评论的时间
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`                    // 软删除

	// GORM 关系定义
	News *News `json:"news,omitempty" gorm:"foreignKey:NewsID;references:ID"`
	User *User `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID"`
}

// CommentResponse 用于向前端返回评论信息时，过滤掉敏感或不需要的字段
type CommentResponse struct {
	ID        uint      `json:"id"`
	NewsID    uint      `json:"news_id"`
	UserID    uint      `json:"user_id"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"created_at"`
}

// CommentCreateRequest 用于创建评论时的请求体
type CommentCreateRequest struct {
	NewsID  uint   `json:"news_id" binding:"required"`                // 新闻ID必填
	Content string `json:"content" binding:"required,min=1,max=1000"` // 评论内容必填，限制长度
}

// CommentDeleteRequest 用于删除评论时的请求体
type CommentDeleteRequest struct {
	CommentID uint `json:"comment_id" binding:"required"` // 评论ID必填
}

func (c *Comment) ToResponse() CommentResponse {
	response := CommentResponse{
		ID:        c.ID,
		NewsID:    c.NewsID,
		UserID:    c.UserID,
		Content:   c.Content,
		CreatedAt: c.CreatedAt,
	}

	return response
}

func (Comment) TableName() string {
	return "comments"
}
