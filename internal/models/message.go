package models

import (
	"time"

	"gorm.io/gorm"
)

// Message 消息模型
type Message struct {
	ID          uint           `json:"id" gorm:"primaryKey"`
	UserID      uint           `json:"user_id" gorm:"not null;index"` // 接收消息的用户ID
	Type        string         `json:"type" gorm:"not null"`          // 消息类型：system, like, comment, follow
	Title       string         `json:"title" gorm:"not null"`         // 消息标题
	Content     string         `json:"content" gorm:"type:text"`      // 消息内容
	RelatedType string         `json:"related_type"`                  // 关联类型：news, event
	RelatedID   uint           `json:"related_id"`                    // 关联内容ID
	SenderID    *uint          `json:"sender_id"`                     // 发送者ID（可为空，系统消息）
	IsRead      bool           `json:"is_read" gorm:"default:false"`  // 是否已读
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	DeletedAt   gorm.DeletedAt `json:"-" gorm:"index"`

	// 关联关系
	User   User `json:"user" gorm:"foreignKey:UserID"`
	Sender User `json:"sender" gorm:"foreignKey:SenderID"`
}

// MessageResponse 消息响应结构
type MessageResponse struct {
	ID          uint      `json:"id"`
	Type        string    `json:"type"`
	Title       string    `json:"title"`
	Content     string    `json:"content"`
	RelatedType string    `json:"related_type"`
	RelatedID   uint      `json:"related_id"`
	SenderName  string    `json:"sender_name"`
	IsRead      bool      `json:"is_read"`
	CreatedAt   time.Time `json:"created_at"`
}

// CreateMessageRequest 创建消息请求
type CreateMessageRequest struct {
	UserID      uint   `json:"user_id" binding:"required"`
	Type        string `json:"type" binding:"required"`
	Title       string `json:"title" binding:"required"`
	Content     string `json:"content" binding:"required"`
	RelatedType string `json:"related_type"`
	RelatedID   uint   `json:"related_id"`
	SenderID    *uint  `json:"sender_id"`
}

// MarkReadRequest 标记已读请求
type MarkReadRequest struct {
	MessageIDs []uint `json:"message_ids"`
}

// ToResponse 转换为响应格式
func (m *Message) ToResponse() MessageResponse {
	senderName := "系统"
	if m.SenderID != nil && m.Sender.Username != "" {
		senderName = m.Sender.Username
	}

	return MessageResponse{
		ID:          m.ID,
		Type:        m.Type,
		Title:       m.Title,
		Content:     m.Content,
		RelatedType: m.RelatedType,
		RelatedID:   m.RelatedID,
		SenderName:  senderName,
		IsRead:      m.IsRead,
		CreatedAt:   m.CreatedAt,
	}
}
