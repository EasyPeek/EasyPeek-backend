package services

import (
	"errors"
	"fmt" // 导入 fmt 包用于错误信息拼接
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/database" // 假设你的数据库连接在此处提供
	"github.com/EasyPeek/EasyPeek-backend/internal/models"   // 导入 News 和相关请求/响应模型
	"gorm.io/gorm"
)

// NewsService 结构体，用于封装与新闻相关的数据库操作和业务逻辑
type NewsService struct {
	db *gorm.DB
}

// NewNewsService 创建并返回一个新的 NewsService 实例
func NewNewsService() *NewsService {
	return &NewsService{
		db: database.GetDB(), // 从 internal/database 包获取 GORM 数据库实例
	}
}

// CreateNews 处理新闻的创建逻辑
func (s *NewsService) CreateNews(req *models.NewsCreateRequest, createdByUserID uint) (*models.News, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	// 构造 News 模型实例
	news := &models.News{
		Title:       req.Title,
		Content:     req.Content,
		Summary:     req.Summary,
		Source:      req.Source,
		Category:    req.Category,
		PublishedAt: time.Now(),            // 默认设置为当前时间，如果请求中没有提供
		CreatedBy:   &createdByUserID,      // 设置创建者ID指针
		IsActive:    true,                  // 默认新闻是活跃的/可见的
		SourceType:  models.NewsTypeManual, // 手动创建的新闻
	}

	// 如果请求中提供了 PublishedAt，则使用请求的值
	if req.PublishedAt != nil {
		news.PublishedAt = *req.PublishedAt
	}
	// 如果请求中提供了 IsActive，则使用请求的值
	if req.IsActive != nil {
		news.IsActive = *req.IsActive
	}

	// 将新闻保存到数据库
	if err := s.db.Create(news).Error; err != nil {
		return nil, fmt.Errorf("failed to create news: %w", err)
	}

	return news, nil
}

// GetNewsByID 根据ID获取单条新闻
func (s *NewsService) GetNewsByID(id uint) (*models.News, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	var news models.News
	// 使用 First 方法根据主键ID查找新闻
	if err := s.db.First(&news, id).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("news not found") // 如果记录未找到
		}
		return nil, fmt.Errorf("failed to get news by ID: %w", err) // 其他数据库错误
	}
	return &news, nil
}

// GetNewsByTitle 根据标题获取新闻列表（标题可能不唯一，所以返回切片）
func (s *NewsService) GetNewsByTitle(title string) ([]models.News, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	var newsList []models.News
	// 使用 Where 方法根据标题查找新闻
	if err := s.db.Where("title = ?", title).Find(&newsList).Error; err != nil {
		return nil, fmt.Errorf("failed to get news by title: %w", err)
	}
	return newsList, nil
}

// UpdateNews 更新现有新闻
func (s *NewsService) UpdateNews(news *models.News, req *models.NewsUpdateRequest) error {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	// 根据请求更新模型字段
	if req.Title != "" {
		news.Title = req.Title
	}
	if req.Content != "" {
		news.Content = req.Content
	}
	if req.Summary != "" {
		news.Summary = req.Summary
	}
	if req.Source != "" {
		news.Source = req.Source
	}
	if req.Category != "" {
		news.Category = req.Category
	}
	if req.PublishedAt != nil {
		news.PublishedAt = *req.PublishedAt
	}
	if req.IsActive != nil {
		news.IsActive = *req.IsActive
	}

	// 使用 Save 方法保存更新，GORM 会根据主键自动判断是插入还是更新
	if err := s.db.Save(news).Error; err != nil {
		return fmt.Errorf("failed to update news: %w", err)
	}
	return nil
}

// DeleteNews 根据ID软删除新闻
func (s *NewsService) DeleteNews(id uint) error {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	// 使用 GORM 的 Delete 方法进行软删除
	result := s.db.Delete(&models.News{}, id)
	if result.Error != nil {
		return fmt.Errorf("failed to delete news: %w", result.Error)
	}
	if result.RowsAffected == 0 {
		return errors.New("news not found or already deleted") // 如果没有行受影响，说明新闻不存在或已被软删除
	}
	return nil
}

// GetAllNews 获取所有新闻，支持分页
func (s *NewsService) GetAllNews(page, pageSize int) ([]models.News, int64, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, 0, errors.New("database connection not initialized")
	}

	var newsList []models.News
	var total int64

	// 计算总记录数
	if err := s.db.Model(&models.News{}).Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count total news: %w", err)
	}

	// 计算分页偏移量
	offset := (page - 1) * pageSize
	if offset < 0 { // 确保 offset 不为负
		offset = 0
	}
	if pageSize <= 0 { // 确保 pageSize 大于0
		pageSize = 10 // 默认值
	}

	// 查询带分页的新闻数据
	if err := s.db.Offset(offset).Limit(pageSize).Find(&newsList).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get all news with pagination: %w", err)
	}

	return newsList, total, nil
}

// SearchNews 根据查询字符串在标题或内容中搜索新闻，支持分页
func (s *NewsService) SearchNews(query string, page, pageSize int) ([]models.News, int64, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, 0, errors.New("database connection not initialized")
	}

	var newsList []models.News
	var total int64

	// 构建搜索查询
	// % 是 SQL 中的通配符，用于模糊匹配
	searchQuery := "%" + query + "%"
	dbQuery := s.db.Model(&models.News{}).
		Where("title ILIKE ? OR content ILIKE ? OR summary ILIKE ?", searchQuery, searchQuery, searchQuery) // ILIKE 用于不区分大小写的模糊匹配，如果是 MySQL 请用 LIKE

	// 计算符合条件的记录总数
	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count search results: %w", err)
	}

	// 计算分页偏移量
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	// 执行带分页的搜索查询
	if err := dbQuery.Offset(offset).Limit(pageSize).Find(&newsList).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to search news with pagination: %w", err)
	}

	return newsList, total, nil
}

// UpdateNewsEventAssociation 批量更新新闻的关联事件ID
func (s *NewsService) UpdateNewsEventAssociation(newsIDs []uint, eventID uint) error {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	if len(newsIDs) == 0 {
		return errors.New("新闻ID列表不能为空")
	}

	// 批量更新新闻的关联事件ID
	result := s.db.Model(&models.News{}).
		Where("id IN ?", newsIDs).
		Update("belonged_event_id", eventID)

	if result.Error != nil {
		return fmt.Errorf("批量更新新闻关联事件失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("没有新闻被更新，请检查新闻ID是否正确")
	}

	return nil
}

// GetNewsByEventID 根据事件ID获取关联的新闻列表
func (s *NewsService) GetNewsByEventID(eventID uint) ([]models.News, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	var newsList []models.News
	if err := s.db.Where("belonged_event_id = ?", eventID).Find(&newsList).Error; err != nil {
		return nil, fmt.Errorf("获取事件关联新闻失败: %w", err)
	}

	return newsList, nil
}

// UpdateNewsEventAssociationByIDs 根据新闻ID列表更新关联事件（支持取消关联）
func (s *NewsService) UpdateNewsEventAssociationByIDs(newsIDs []uint, eventID *uint) error {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	if len(newsIDs) == 0 {
		return errors.New("新闻ID列表不能为空")
	}

	// 批量更新新闻的关联事件ID（如果eventID为nil，则取消关联）
	result := s.db.Model(&models.News{}).
		Where("id IN ?", newsIDs).
		Update("belonged_event_id", eventID)

	if result.Error != nil {
		return fmt.Errorf("批量更新新闻关联事件失败: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("没有新闻被更新，请检查新闻ID是否正确")
	}

	return nil
}

// GetUnlinkedNews 获取未关联任何事件的新闻
func (s *NewsService) GetUnlinkedNews(page, pageSize int) ([]models.News, int64, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, 0, errors.New("database connection not initialized")
	}

	var newsList []models.News
	var total int64

	// 计算未关联事件的新闻总数
	if err := s.db.Model(&models.News{}).Where("belonged_event_id IS NULL").Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count unlinked news: %w", err)
	}

	// 计算分页偏移量
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	// 查询未关联事件的新闻
	if err := s.db.Where("belonged_event_id IS NULL").
		Order("created_at desc").
		Offset(offset).Limit(pageSize).
		Find(&newsList).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get unlinked news: %w", err)
	}

	return newsList, total, nil
}

// GetNewsByCategory 根据分类获取新闻，支持分页
func (s *NewsService) GetNewsByCategory(category string, page, pageSize int) ([]models.News, int64, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, 0, errors.New("database connection not initialized")
	}

	var newsList []models.News
	var total int64

	// 构建分类查询
	dbQuery := s.db.Model(&models.News{}).Where("category = ?", category)

	// 计算符合条件的记录总数
	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count news by category: %w", err)
	}

	// 计算分页偏移量
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	// 执行带分页的分类查询
	if err := dbQuery.Order("created_at desc").
		Offset(offset).Limit(pageSize).
		Find(&newsList).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get news by category: %w", err)
	}

	return newsList, total, nil
}

// GetHotNews 获取热门新闻，按热度分数排序
func (s *NewsService) GetHotNews(limit int) ([]models.News, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	// 设置默认限制
	if limit <= 0 {
		limit = 10
	}

	var newsList []models.News

	// 按热度分数降序排列获取热门新闻
	if err := s.db.Where("is_active = ?", true).
		Order("hotness_score desc, view_count desc, like_count desc, created_at desc").
		Limit(limit).
		Find(&newsList).Error; err != nil {
		return nil, fmt.Errorf("failed to get hot news: %w", err)
	}

	return newsList, nil
}
