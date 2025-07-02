package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"gorm.io/gorm"
)

// CommentService 结构体，用于封装与评论相关的数据库操作和业务逻辑
type CommentService struct {
	db *gorm.DB
}

// NewCommentService 创建并返回一个新的 CommentService 实例
func NewCommentService() *CommentService {
	return &CommentService{
		db: database.GetDB(),
	}
}

// CreateComment 创建新评论
func (s *CommentService) CreateComment(req *models.CommentCreateRequest, userID uint) (*models.Comment, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	// 验证新闻是否存在
	var news models.News
	if err := s.db.First(&news, req.NewsID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("news not found")
		}
		return nil, fmt.Errorf("failed to check news existence: %w", err)
	}

	// 验证用户是否存在
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("user not found")
		}
		return nil, fmt.Errorf("failed to check user existence: %w", err)
	}

	// 构造 Comment 模型实例
	comment := &models.Comment{
		NewsID:    req.NewsID,
		UserID:    userID,
		Content:   req.Content,
		CreatedAt: time.Now(),
	}

	// 将评论保存到数据库
	if err := s.db.Create(comment).Error; err != nil {
		return nil, fmt.Errorf("failed to create comment: %w", err)
	}

	// 更新新闻的评论数
	if err := s.db.Model(&news).Update("comment_count", gorm.Expr("comment_count + ?", 1)).Error; err != nil {
		// 这里只是记录错误，不影响评论创建的成功
		fmt.Printf("failed to update news comment count: %v\n", err)
	}

	return comment, nil
}

// GetCommentByID 根据ID获取单条评论
func (s *CommentService) GetCommentByID(id uint) (*models.Comment, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	var comment models.Comment
	// 使用 First 方法根据主键ID查找评论，并预加载用户信息
	if err := s.db.Preload("User").First(&comment, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("comment not found")
		}
		return nil, fmt.Errorf("failed to get comment by ID: %w", err)
	}
	return &comment, nil
}

// GetCommentsByNewsID 根据新闻ID获取评论列表，支持分页
func (s *CommentService) GetCommentsByNewsID(newsID uint, page, pageSize int) ([]models.Comment, int64, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, 0, errors.New("database connection not initialized")
	}

	var comments []models.Comment
	var total int64

	// 计算总记录数
	if err := s.db.Model(&models.Comment{}).Where("news_id = ?", newsID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count total comments: %w", err)
	}

	// 计算分页偏移量
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	// 查询带分页的评论数据，按创建时间倒序排列，并预加载用户信息
	if err := s.db.Where("news_id = ?", newsID).
		Preload("User").
		Order("created_at desc").
		Offset(offset).Limit(pageSize).
		Find(&comments).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get comments with pagination: %w", err)
	}

	return comments, total, nil
}

// GetCommentsByUserID 根据用户ID获取评论列表，支持分页
func (s *CommentService) GetCommentsByUserID(userID uint, page, pageSize int) ([]models.Comment, int64, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, 0, errors.New("database connection not initialized")
	}

	var comments []models.Comment
	var total int64

	// 计算总记录数
	if err := s.db.Model(&models.Comment{}).Where("user_id = ?", userID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count total comments: %w", err)
	}

	// 计算分页偏移量
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	// 查询带分页的评论数据，按创建时间倒序排列，并预加载新闻信息
	if err := s.db.Where("user_id = ?", userID).
		Preload("News").
		Order("created_at desc").
		Offset(offset).Limit(pageSize).
		Find(&comments).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get comments with pagination: %w", err)
	}

	return comments, total, nil
}

// DeleteComment 删除评论（软删除）
func (s *CommentService) DeleteComment(commentID uint, userID uint) error {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	// 先获取评论记录
	var comment models.Comment
	if err := s.db.First(&comment, commentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("comment not found")
		}
		return fmt.Errorf("failed to get comment: %w", err)
	}

	// 检查权限：只有评论的作者才能删除评论
	if comment.UserID != userID {
		return errors.New("permission denied: only comment author can delete")
	}

	// 使用 GORM 的 Delete 方法进行软删除
	result := s.db.Delete(&comment)
	if result.Error != nil {
		return fmt.Errorf("failed to delete comment: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("comment not found or already deleted")
	}

	// 更新新闻的评论数
	if err := s.db.Model(&models.News{}).Where("id = ?", comment.NewsID).
		Update("comment_count", gorm.Expr("comment_count - ?", 1)).Error; err != nil {
		// 这里只是记录错误，不影响评论删除的成功
		fmt.Printf("failed to update news comment count: %v\n", err)
	}

	return nil
}

// AdminDeleteComment 管理员删除评论（硬删除）
func (s *CommentService) AdminDeleteComment(commentID uint) error {
	// TODO: 管理员功能暂未实现，等待前端需求
	return errors.New("admin functionality not implemented yet")
}

// GetAllComments 获取所有评论，支持分页（管理员功能）
func (s *CommentService) GetAllComments(page, pageSize int) ([]models.Comment, int64, error) {
	// TODO: 管理员功能暂未实现，等待前端需求
	return nil, 0, errors.New("admin functionality not implemented yet")
}
