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
		UserID:    &userID, // 使用指针，因为UserID现在是指针类型
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

// ReplyToComment 回复评论
func (s *CommentService) ReplyToComment(req *models.CommentReplyRequest, userID uint) (*models.Comment, error) {
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

	// 验证父评论是否存在
	var parentComment models.Comment
	if err := s.db.First(&parentComment, req.ParentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("parent comment not found")
		}
		return nil, fmt.Errorf("failed to check parent comment existence: %w", err)
	}

	// 验证父评论是否属于同一个新闻
	if parentComment.NewsID != req.NewsID {
		return nil, errors.New("parent comment does not belong to the same news")
	}

	// 构造回复评论实例
	reply := &models.Comment{
		NewsID:    req.NewsID,
		UserID:    &userID,
		ParentID:  &req.ParentID,
		Content:   req.Content,
		CreatedAt: time.Now(),
	}

	// 将回复保存到数据库
	if err := s.db.Create(reply).Error; err != nil {
		return nil, fmt.Errorf("failed to create reply: %w", err)
	}

	// 更新新闻的评论数
	if err := s.db.Model(&news).Update("comment_count", gorm.Expr("comment_count + ?", 1)).Error; err != nil {
		// 这里只是记录错误，不影响回复创建的成功
		fmt.Printf("failed to update news comment count: %v\n", err)
	}

	// 创建回复消息通知（仅当父评论有作者且不是自己回复自己时）
	if parentComment.UserID != nil && *parentComment.UserID != userID {
		messageService := NewMessageService()
		title := "您的评论收到了回复"
		content := fmt.Sprintf("用户「%s」回复了您的评论「%s」", user.Username, parentComment.Content)

		// 创建回复消息（忽略错误，不影响回复操作）
		_ = messageService.CreateMessage(
			*parentComment.UserID, // 父评论作者ID
			"comment",
			title,
			content,
			"comment",
			reply.ID,
			&userID, // 回复者ID
		)
	}

	return reply, nil
}

// CreateAnonymousComment 创建匿名评论
func (s *CommentService) CreateAnonymousComment(req *models.CommentAnonymousCreateRequest) (*models.Comment, error) {
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

	// 构造匿名评论实例，UserID为nil
	comment := &models.Comment{
		NewsID:    req.NewsID,
		UserID:    nil, // 匿名评论，UserID为空
		Content:   req.Content,
		CreatedAt: time.Now(),
	}

	// 将评论保存到数据库
	if err := s.db.Create(comment).Error; err != nil {
		return nil, fmt.Errorf("failed to create anonymous comment: %w", err)
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
	// 使用 First 方法根据主键ID查找评论，并预加载用户信息和点赞记录
	if err := s.db.Preload("User").Preload("Likes").First(&comment, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("comment not found") // 如果记录未找到
		}
		return nil, fmt.Errorf("failed to get comment by ID: %w", err) // 其他数据库错误
	}
	return &comment, nil
}

// GetCommentsByNewsID 根据新闻ID获取评论列表，支持分页
func (s *CommentService) GetCommentsByNewsID(newsID uint, page, pageSize int, currentUserID *uint) ([]models.Comment, int64, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, 0, errors.New("database connection not initialized")
	}

	var comments []models.Comment
	var total int64

	// 计算总记录数（只统计顶级评论，不包括回复）
	if err := s.db.Model(&models.Comment{}).Where("news_id = ? AND parent_id IS NULL", newsID).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count total comments: %w", err)
	}

	// 计算分页偏移量
	offset := (page - 1) * pageSize
	if offset < 0 { // 确保 offset 不为负
		offset = 0
	}
	if pageSize <= 0 { // 确保 pageSize 大于0
		pageSize = 10 // 默认值
	}

	// 查询带分页的评论数据，按创建时间倒序排列，只获取顶级评论（没有父评论的）
	query := s.db.Where("news_id = ? AND parent_id IS NULL", newsID).
		Preload("Replies", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at ASC") // 回复按时间正序排列
		}).Preload("Replies.User").Preload("Replies.Likes").Preload("User").Preload("Likes").
		Order("created_at desc").
		Offset(offset).Limit(pageSize)

	if err := query.Find(&comments).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get comments with pagination: %w", err)
	}

	// 添加调试日志
	fmt.Printf("GetCommentsByNewsID: newsID=%d, page=%d, pageSize=%d, total=%d, found=%d\n",
		newsID, page, pageSize, total, len(comments))

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
	if offset < 0 { // 确保 offset 不为负
		offset = 0
	}
	if pageSize <= 0 { // 确保 pageSize 大于0
		pageSize = 10 // 默认值
	}

	// 查询带分页的评论数据，按创建时间倒序排列
	if err := s.db.Where("user_id = ?", userID).
		Preload("User").
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
	// 匿名评论不能被删除（因为没有用户ID）
	if comment.UserID == nil {
		return errors.New("permission denied: anonymous comments cannot be deleted")
	}
	if *comment.UserID != userID {
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

	// 更新新闻的评论数（包括回复）
	if err := s.db.Model(&models.News{}).Where("id = ?", comment.NewsID).
		Update("comment_count", gorm.Expr("comment_count - ?", 1)).Error; err != nil {
		// 这里只是记录错误，不影响评论删除的成功
		fmt.Printf("failed to update news comment count: %v\n", err)
	}

	// 如果是顶级评论，还需要删除其下的所有回复
	if comment.ParentID == nil {
		// 删除所有回复
		if err := s.db.Where("parent_id = ?", comment.ID).Delete(&models.Comment{}).Error; err != nil {
			fmt.Printf("failed to delete replies: %v\n", err)
		}
	}

	return nil
}

// AdminDeleteComment 管理员删除评论（硬删除）
func (s *CommentService) AdminDeleteComment(commentID uint) error {
	// TODO: 管理员功能暂未实现，等待前端需求
	return errors.New("admin functionality not implemented yet")
}

// GetAllComments 获取所有评论，支持分页
func (s *CommentService) GetAllComments(page, pageSize int) ([]models.Comment, int64, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, 0, errors.New("database connection not initialized")
	}

	var comments []models.Comment
	var total int64

	// 计算总记录数
	if err := s.db.Model(&models.Comment{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count total comments: %w", err)
	}

	// 计算分页偏移量
	offset := (page - 1) * pageSize
	if offset < 0 { // 确保 offset 不为负
		offset = 0
	}
	if pageSize <= 0 { // 确保 pageSize 大于0
		pageSize = 10 // 默认值
	}

	// 查询带分页的评论数据
	if err := s.db.Preload("User").
		Offset(offset).Limit(pageSize).Find(&comments).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get all comments with pagination: %w", err)
	}

	return comments, total, nil
}

// LikeComment 点赞评论
func (s *CommentService) LikeComment(commentID uint, userID uint) error {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	// 验证评论是否存在
	var comment models.Comment
	if err := s.db.First(&comment, commentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("comment not found")
		}
		return fmt.Errorf("failed to check comment existence: %w", err)
	}

	// 验证用户是否存在
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	// 检查用户是否已经点赞过该评论
	var existingLike models.CommentLike
	if err := s.db.Where("comment_id = ? AND user_id = ?", commentID, userID).First(&existingLike).Error; err == nil {
		// 用户已经点赞过，返回错误
		return errors.New("user has already liked this comment")
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		// 其他数据库错误
		return fmt.Errorf("failed to check existing like: %w", err)
	}

	// 创建点赞记录
	like := models.CommentLike{
		CommentID: commentID,
		UserID:    userID,
		CreatedAt: time.Now(),
	}

	if err := s.db.Create(&like).Error; err != nil {
		return fmt.Errorf("failed to create like record: %w", err)
	}

	// 增加评论的点赞数
	if err := s.db.Model(&comment).Update("like_count", gorm.Expr("like_count + ?", 1)).Error; err != nil {
		return fmt.Errorf("failed to update comment like count: %w", err)
	}

	// 创建点赞消息通知（仅当评论有作者且不是自己给自己点赞时）
	if comment.UserID != nil && *comment.UserID != userID {
		messageService := NewMessageService()
		title := "您的评论收到了点赞"
		content := fmt.Sprintf("用户「%s」点赞了您的评论「%s」", user.Username, comment.Content)

		// 创建点赞消息（忽略错误，不影响点赞操作）
		_ = messageService.CreateMessage(
			*comment.UserID, // 评论作者ID
			"like",
			title,
			content,
			"comment",
			commentID,
			&userID, // 点赞者ID
		)
	}

	return nil
}

// UnlikeComment 取消点赞评论
func (s *CommentService) UnlikeComment(commentID uint, userID uint) error {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	// 验证评论是否存在
	var comment models.Comment
	if err := s.db.First(&comment, commentID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("comment not found")
		}
		return fmt.Errorf("failed to check comment existence: %w", err)
	}

	// 验证用户是否存在
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("user not found")
		}
		return fmt.Errorf("failed to check user existence: %w", err)
	}

	// 查找并删除点赞记录
	var like models.CommentLike
	if err := s.db.Where("comment_id = ? AND user_id = ?", commentID, userID).First(&like).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("like record not found")
		}
		return fmt.Errorf("failed to find like record: %w", err)
	}

	// 删除点赞记录
	if err := s.db.Delete(&like).Error; err != nil {
		return fmt.Errorf("failed to delete like record: %w", err)
	}

	// 减少评论的点赞数
	if err := s.db.Model(&comment).Update("like_count", gorm.Expr("like_count - ?", 1)).Error; err != nil {
		return fmt.Errorf("failed to update comment like count: %w", err)
	}

	return nil
}

// GetUserLikedComments 获取用户点赞过的评论ID列表
func (s *CommentService) GetUserLikedComments(userID uint) ([]uint, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	var likedCommentIDs []uint
	if err := s.db.Model(&models.CommentLike{}).
		Where("user_id = ?", userID).
		Pluck("comment_id", &likedCommentIDs).Error; err != nil {
		return nil, fmt.Errorf("failed to get user liked comments: %w", err)
	}

	return likedCommentIDs, nil
}
