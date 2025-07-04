package models

import (
	"time"

	"gorm.io/gorm"
)

// Comment 对应数据库中的 'comments' 表
type Comment struct {
	ID        uint           `json:"id" gorm:"primaryKey"`
	NewsID    uint           `json:"news_id" gorm:"not null;index"`     // 新闻ID
	UserID    *uint          `json:"user_id" gorm:"index"`              // 发表评论的用户ID，可为空（匿名评论）
	ParentID  *uint          `json:"parent_id" gorm:"index"`            // 父评论ID，用于回复功能
	Content   string         `json:"content" gorm:"type:text;not null"` // 评论的内容
	LikeCount int            `json:"like_count" gorm:"default:0"`       // 点赞数
	CreatedAt time.Time      `json:"created_at"`                        // 评论的时间
	DeletedAt gorm.DeletedAt `json:"-" gorm:"index"`                    // 软删除

	// GORM 关系定义
	News    *News     `json:"news,omitempty" gorm:"foreignKey:NewsID;references:ID"`
	User    *User     `json:"user,omitempty" gorm:"foreignKey:UserID;references:ID"`
	Parent  *Comment  `json:"parent,omitempty" gorm:"foreignKey:ParentID;references:ID"`
	Replies []Comment `json:"replies,omitempty" gorm:"foreignKey:ParentID"`
}

// CommentResponse 用于向前端返回评论信息时，过滤掉敏感或不需要的字段
type CommentResponse struct {
	ID          uint              `json:"id"`
	NewsID      uint              `json:"news_id"`
	UserID      *uint             `json:"user_id"`   // 可为空，表示匿名用户
	Username    *string           `json:"username"`  // 用户名，可为空（匿名评论）
	ParentID    *uint             `json:"parent_id"` // 父评论ID
	Content     string            `json:"content"`
	LikeCount   int               `json:"like_count"`
	CreatedAt   time.Time         `json:"created_at"`
	IsAnonymous bool              `json:"is_anonymous"`      // 是否为匿名评论
	IsReply     bool              `json:"is_reply"`          // 是否为回复
	Replies     []CommentResponse `json:"replies,omitempty"` // 回复列表
}

// CommentCreateRequest 用于创建评论时的请求体
type CommentCreateRequest struct {
	NewsID  uint   `json:"news_id" binding:"required"`                // 新闻ID必填
	Content string `json:"content" binding:"required,min=1,max=1000"` // 评论内容必填，限制长度
}

// CommentAnonymousCreateRequest 用于匿名用户创建评论时的请求体
type CommentAnonymousCreateRequest struct {
	NewsID  uint   `json:"news_id" binding:"required"`                // 新闻ID必填
	Content string `json:"content" binding:"required,min=1,max=1000"` // 评论内容必填，限制长度
}

// CommentDeleteRequest 用于删除评论时的请求体
type CommentDeleteRequest struct {
	CommentID uint `json:"comment_id" binding:"required"` // 评论ID必填
}

// CommentLikeRequest 用于点赞评论时的请求体
type CommentLikeRequest struct {
	CommentID uint `json:"comment_id" binding:"required"` // 评论ID必填
}

// CommentReplyRequest 用于回复评论时的请求体
type CommentReplyRequest struct {
	NewsID   uint   `json:"news_id" binding:"required"`                // 新闻ID必填
	ParentID uint   `json:"parent_id" binding:"required"`              // 父评论ID必填
	Content  string `json:"content" binding:"required,min=1,max=1000"` // 回复内容必填，限制长度
}

func (c *Comment) ToResponse() CommentResponse {
	response := CommentResponse{
		ID:          c.ID,
		NewsID:      c.NewsID,
		UserID:      c.UserID,
		ParentID:    c.ParentID,
		Content:     c.Content,
		LikeCount:   c.LikeCount,
		CreatedAt:   c.CreatedAt,
		IsAnonymous: c.UserID == nil,     // 如果UserID为空，则为匿名评论
		IsReply:     c.ParentID != nil,   // 如果有父评论ID，则为回复
		Replies:     []CommentResponse{}, // 初始化回复列表
	}

	// 如果有用户信息，添加用户名
	if c.User != nil {
		response.Username = &c.User.Username
	}

	// 如果有回复，转换为响应格式
	if len(c.Replies) > 0 {
		for _, reply := range c.Replies {
			response.Replies = append(response.Replies, reply.ToResponse())
		}
	}

	return response
}

func (Comment) TableName() string {
	return "comments"
}
