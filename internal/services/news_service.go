package services

import (
	"encoding/json"
	"errors"
	"fmt" // 导入 fmt 包用于错误信息拼接
	"strings"
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

	// 如果pageSize为-1，返回所有数据，否则进行分页
	if pageSize == -1 {
		// 返回所有数据，不进行分页
		if err := s.db.Find(&newsList).Error; err != nil {
			return nil, 0, fmt.Errorf("failed to get all news: %w", err)
		}
	} else {
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

// SearchNewsWithMode 支持多种搜索模式的新闻搜索
func (s *NewsService) SearchNewsWithMode(query, mode string, page, pageSize int) ([]models.News, int64, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, 0, errors.New("database connection not initialized")
	}

	var newsList []models.News
	var total int64
	var dbQuery *gorm.DB

	// 根据搜索模式构建不同的查询
	switch mode {
	case "keywords":
		// 关键词搜索：精确匹配，空格分隔的关键词都必须包含
		keywords := strings.Fields(query) // 分割关键词
		if len(keywords) == 0 {
			return []models.News{}, 0, nil
		}

		dbQuery = s.db.Model(&models.News{}).Where("is_active = ?", true)

		// 每个关键词都必须在标题、内容或摘要中出现
		for _, keyword := range keywords {
			searchTerm := "%" + keyword + "%"
			dbQuery = dbQuery.Where("(title ILIKE ? OR content ILIKE ? OR summary ILIKE ?)",
				searchTerm, searchTerm, searchTerm)
		}

		// 按创建时间排序（关键词搜索已经通过多重WHERE条件保证了相关性）
		dbQuery = dbQuery.Order("created_at DESC")

	case "semantic":
		// 语义搜索：扩展查询，包含同义词和相关概念
		expandedQuery := expandSemanticQuery(query)
		searchQuery := "%" + expandedQuery + "%"

		dbQuery = s.db.Model(&models.News{}).
			Where("is_active = ? AND (title ILIKE ? OR content ILIKE ? OR summary ILIKE ? OR tags ILIKE ?)",
				true, searchQuery, searchQuery, searchQuery, searchQuery).
			Order("created_at DESC")

	default: // normal 模式
		// 普通搜索：模糊匹配
		searchQuery := "%" + query + "%"
		dbQuery = s.db.Model(&models.News{}).
			Where("is_active = ? AND (title ILIKE ? OR content ILIKE ? OR summary ILIKE ?)",
				true, searchQuery, searchQuery, searchQuery).
			Order("created_at DESC")
	}

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

// expandSemanticQuery 扩展语义查询，添加相关关键词
func expandSemanticQuery(query string) string {
	// 简单的语义扩展映射
	semanticMap := map[string][]string{
		"人工智能": {"AI", "机器学习", "深度学习", "神经网络", "算法"},
		"AI":   {"人工智能", "机器学习", "深度学习", "智能"},
		"科技":   {"技术", "创新", "发明", "研发", "数字化"},
		"经济":   {"财经", "金融", "投资", "市场", "贸易"},
		"环保":   {"环境", "绿色", "生态", "可持续", "节能"},
		"医疗":   {"健康", "医院", "药物", "治疗", "疾病"},
		"教育":   {"学习", "学校", "培训", "知识", "学术"},
		"新能源":  {"清洁能源", "太阳能", "风能", "电动", "绿色能源"},
	}

	expandedTerms := []string{query}
	queryLower := strings.ToLower(query)

	// 查找相关词汇
	for key, synonyms := range semanticMap {
		if strings.Contains(queryLower, strings.ToLower(key)) {
			expandedTerms = append(expandedTerms, synonyms...)
		}
	}

	// 返回扩展后的查询字符串
	return strings.Join(expandedTerms, " ")
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

// GetHotNews 获取热门新闻，按照浏览量、点赞数等热度指标排序
func (s *NewsService) GetHotNews(limit int) ([]models.News, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	var newsList []models.News

	// 设置默认限制
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	// 查询热门新闻，按创建时间倒序排列（可以后续根据实际的热度字段调整）
	// 这里暂时按照创建时间排序，实际项目中可以根据浏览量、点赞数等字段排序
	if err := s.db.Where("is_active = ?", true).
		Order("created_at desc").
		Limit(limit).
		Find(&newsList).Error; err != nil {
		return nil, fmt.Errorf("failed to get hot news: %w", err)
	}

	return newsList, nil
}

// GetLatestNews 获取最新新闻，按发布时间倒序排列
func (s *NewsService) GetLatestNews(limit int) ([]models.News, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	var newsList []models.News

	// 设置默认限制
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	// 查询最新新闻，按发布时间倒序排列
	if err := s.db.Where("is_active = ?", true).
		Order("published_at desc").
		Limit(limit).
		Find(&newsList).Error; err != nil {
		return nil, fmt.Errorf("failed to get latest news: %w", err)
	}

	return newsList, nil
}

// GetNewsByCategoryHot 按分类获取热门新闻
func (s *NewsService) GetNewsByCategoryHot(category string, limit int) ([]models.News, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	var newsList []models.News

	// 设置默认限制
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	// 查询指定分类的热门新闻，按热度相关字段排序
	// 这里暂时按照创建时间排序，实际项目中可以根据浏览量、点赞数等字段排序
	if err := s.db.Where("is_active = ? AND category = ?", true, category).
		Order("created_at desc").
		Limit(limit).
		Find(&newsList).Error; err != nil {
		return nil, fmt.Errorf("failed to get hot news by category: %w", err)
	}

	return newsList, nil
}

// GetNewsByCategoryLatest 按分类获取最新新闻
func (s *NewsService) GetNewsByCategoryLatest(category string, limit int) ([]models.News, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	var newsList []models.News

	// 设置默认限制
	if limit <= 0 || limit > 100 {
		limit = 10
	}

	// 查询指定分类的最新新闻，按发布时间倒序排列
	if err := s.db.Where("is_active = ? AND category = ?", true, category).
		Order("published_at desc").
		Limit(limit).
		Find(&newsList).Error; err != nil {
		return nil, fmt.Errorf("failed to get latest news by category: %w", err)
	}

	return newsList, nil
}

// LikeNews 点赞新闻
func (s *NewsService) LikeNews(newsID uint, userID uint) error {
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	// 检查新闻是否存在
	var news models.News
	if err := s.db.First(&news, newsID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return errors.New("news not found")
		}
		return fmt.Errorf("failed to find news: %w", err)
	}

	// 检查用户是否已经点赞过该新闻
	var existingLike models.NewsLike
	err := s.db.Where("news_id = ? AND user_id = ?", newsID, userID).First(&existingLike).Error

	if err == nil {
		// 已经点赞过，取消点赞
		if err := s.db.Delete(&existingLike).Error; err != nil {
			return fmt.Errorf("failed to unlike news: %w", err)
		}

		// 减少点赞数
		if err := s.db.Model(&news).UpdateColumn("like_count", gorm.Expr("like_count - 1")).Error; err != nil {
			return fmt.Errorf("failed to decrease like count: %w", err)
		}

		return nil
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return fmt.Errorf("failed to check existing like: %w", err)
	}

	// 没有点赞过，添加点赞记录
	newLike := models.NewsLike{
		NewsID: newsID,
		UserID: userID,
		LikeAt: time.Now(),
	}

	if err := s.db.Create(&newLike).Error; err != nil {
		return fmt.Errorf("failed to create like record: %w", err)
	}

	// 增加点赞数
	if err := s.db.Model(&news).UpdateColumn("like_count", gorm.Expr("like_count + 1")).Error; err != nil {
		return fmt.Errorf("failed to increase like count: %w", err)
	}

	return nil
}

// CheckUserLikedNews 检查用户是否已点赞该新闻
func (s *NewsService) CheckUserLikedNews(newsID uint, userID uint) (bool, error) {
	if s.db == nil {
		return false, errors.New("database connection not initialized")
	}

	var like models.NewsLike
	err := s.db.Where("news_id = ? AND user_id = ?", newsID, userID).First(&like).Error

	if err == nil {
		return true, nil
	} else if errors.Is(err, gorm.ErrRecordNotFound) {
		return false, nil
	}

	return false, fmt.Errorf("failed to check user like status: %w", err)
}

// IncrementViewCount 增加新闻浏览量
func (s *NewsService) IncrementViewCount(newsID uint) error {
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	// 检查新闻是否存在并增加浏览量
	result := s.db.Model(&models.News{}).Where("id = ?", newsID).UpdateColumn("view_count", gorm.Expr("view_count + 1"))

	if result.Error != nil {
		return fmt.Errorf("failed to increment view count: %w", result.Error)
	}

	if result.RowsAffected == 0 {
		return errors.New("news not found")
	}

	return nil
}

// UserPreferences 用户偏好结构
type UserPreferences struct {
	Categories           []string `json:"categories"`
	Keywords             []string `json:"keywords"`
	Sources              []string `json:"sources"`
	ReadingMode          string   `json:"readingMode"`        // fast, balanced, deep
	DiversityLevel       int      `json:"diversityLevel"`     // 1-10
	PersonalizedWeight   int      `json:"personalizedWeight"` // 1-10
	ExcludeKeywords      []string `json:"excludeKeywords"`
	EnableSmartRecommend bool     `json:"enableSmartRecommend"`
	EnableTrendingBoost  bool     `json:"enableTrendingBoost"`
	EnableSemanticMatch  bool     `json:"enableSemanticMatch"`
}

// GetPersonalizedNews 获取个性化推荐新闻
func (s *NewsService) GetPersonalizedNews(userID uint, page, pageSize int) ([]models.News, int64, error) {
	if s.db == nil {
		return nil, 0, errors.New("database connection not initialized")
	}

	// 获取用户偏好
	userPrefs, err := s.getUserPreferences(userID)
	if err != nil {
		// 如果获取不到用户偏好，返回热门新闻
		news, err := s.GetHotNews(pageSize)
		return news, int64(len(news)), err
	}

	// 构建基础查询
	query := s.db.Model(&models.News{}).Where("is_active = ?", true)

	// 根据用户偏好过滤
	query = s.applyPreferenceFilters(query, userPrefs)

	// 计算总数
	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count personalized news: %w", err)
	}

	// 获取新闻列表
	var newsList []models.News
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}

	// 根据偏好排序
	orderClause := s.buildPersonalizedOrder(userPrefs)
	if err := query.Order(orderClause).Offset(offset).Limit(pageSize).Find(&newsList).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to get personalized news: %w", err)
	}

	// 后处理：应用用户偏好评分和重新排序
	newsList = s.scoreAndSortByPreferences(newsList, userPrefs)

	return newsList, total, nil
}

// getUserPreferences 获取用户偏好
func (s *NewsService) getUserPreferences(userID uint) (*UserPreferences, error) {
	var user models.User
	if err := s.db.First(&user, userID).Error; err != nil {
		return nil, fmt.Errorf("failed to get user: %w", err)
	}

	if user.Interests == "" {
		return nil, errors.New("user has no preferences set")
	}

	var prefs UserPreferences
	if err := json.Unmarshal([]byte(user.Interests), &prefs); err != nil {
		return nil, fmt.Errorf("failed to parse user preferences: %w", err)
	}

	return &prefs, nil
}

// applyPreferenceFilters 应用偏好过滤器
func (s *NewsService) applyPreferenceFilters(query *gorm.DB, prefs *UserPreferences) *gorm.DB {
	// 分类过滤
	if len(prefs.Categories) > 0 {
		query = query.Where("category IN ?", prefs.Categories)
	}

	// 来源过滤
	if len(prefs.Sources) > 0 {
		query = query.Where("source IN ?", prefs.Sources)
	}

	// 排除关键词
	for _, keyword := range prefs.ExcludeKeywords {
		excludePattern := "%" + keyword + "%"
		query = query.Where("title NOT ILIKE ? AND content NOT ILIKE ? AND summary NOT ILIKE ?",
			excludePattern, excludePattern, excludePattern)
	}

	// 根据阅读模式过滤内容长度
	switch prefs.ReadingMode {
	case "fast":
		// 快速阅读：偏好短文章
		query = query.Where("LENGTH(content) < ?", 1000)
	case "deep":
		// 深度阅读：偏好长文章
		query = query.Where("LENGTH(content) > ?", 2000)
		// "balanced" 不做特殊过滤
	}

	return query
}

// buildPersonalizedOrder 构建个性化排序
func (s *NewsService) buildPersonalizedOrder(prefs *UserPreferences) string {
	if prefs.EnableTrendingBoost {
		// 结合热度和时间排序
		return "view_count DESC, like_count DESC, created_at DESC"
	}

	// 默认按时间排序
	return "created_at DESC"
}

// scoreAndSortByPreferences 基于偏好评分和排序
func (s *NewsService) scoreAndSortByPreferences(newsList []models.News, prefs *UserPreferences) []models.News {
	// 为每条新闻计算偏好匹配分数
	type NewsWithScore struct {
		News  models.News
		Score float64
	}

	var newsWithScores []NewsWithScore

	for _, news := range newsList {
		score := s.calculatePreferenceScore(news, prefs)
		newsWithScores = append(newsWithScores, NewsWithScore{
			News:  news,
			Score: score,
		})
	}

	// 根据分数排序
	for i := 0; i < len(newsWithScores)-1; i++ {
		for j := i + 1; j < len(newsWithScores); j++ {
			if newsWithScores[i].Score < newsWithScores[j].Score {
				newsWithScores[i], newsWithScores[j] = newsWithScores[j], newsWithScores[i]
			}
		}
	}

	// 提取排序后的新闻列表
	var sortedNews []models.News
	for _, nws := range newsWithScores {
		sortedNews = append(sortedNews, nws.News)
	}

	return sortedNews
}

// calculatePreferenceScore 计算偏好匹配分数
func (s *NewsService) calculatePreferenceScore(news models.News, prefs *UserPreferences) float64 {
	score := 0.0

	// 分类匹配权重
	for _, category := range prefs.Categories {
		if news.Category == category {
			score += 10.0 * (float64(prefs.PersonalizedWeight) / 10.0)
			break
		}
	}

	// 关键词匹配权重
	for _, keyword := range prefs.Keywords {
		keywordLower := strings.ToLower(keyword)
		titleLower := strings.ToLower(news.Title)
		summaryLower := strings.ToLower(news.Summary)
		contentLower := strings.ToLower(news.Content)

		titleMatch := strings.Contains(titleLower, keywordLower)
		summaryMatch := strings.Contains(summaryLower, keywordLower)
		contentMatch := strings.Contains(contentLower, keywordLower)

		if titleMatch {
			score += 8.0 * (float64(prefs.PersonalizedWeight) / 10.0)
		} else if summaryMatch {
			score += 5.0 * (float64(prefs.PersonalizedWeight) / 10.0)
		} else if contentMatch {
			score += 3.0 * (float64(prefs.PersonalizedWeight) / 10.0)
		}
	}

	// 来源匹配权重
	for _, source := range prefs.Sources {
		if news.Source == source {
			score += 6.0 * (float64(prefs.PersonalizedWeight) / 10.0)
			break
		}
	}

	// 热度加成
	if prefs.EnableTrendingBoost {
		// 基于浏览量和点赞数的热度分数
		hotnessScore := float64(news.ViewCount)/1000.0 + float64(news.LikeCount)/100.0
		score += hotnessScore * 0.3
	}

	// 新鲜度加成
	timeDiff := time.Since(news.CreatedAt).Hours()
	if timeDiff < 24 { // 24小时内的新闻
		score += 2.0
	} else if timeDiff < 72 { // 3天内的新闻
		score += 1.0
	}

	// 多样性调整
	diversityFactor := float64(prefs.DiversityLevel) / 10.0
	if diversityFactor < 0.5 {
		// 低多样性：更专注于偏好内容
		score *= (1.0 + (0.5 - diversityFactor))
	} else if diversityFactor > 0.7 {
		// 高多样性：适当降低匹配权重
		score *= (1.0 - (diversityFactor-0.7)*0.3)
	}

	return score
}

// GetNewsWithPreferences 根据偏好获取新闻（支持多种模式）
func (s *NewsService) GetNewsWithPreferences(userID uint, mode string, page, pageSize int, filters map[string]interface{}) ([]models.News, int64, error) {
	switch mode {
	case "personalized":
		if userID > 0 {
			return s.GetPersonalizedNews(userID, page, pageSize)
		}
		// 未登录用户返回热门新闻
		news, err := s.GetHotNews(pageSize)
		return news, int64(len(news)), err
	case "hot":
		news, err := s.GetHotNews(pageSize)
		return news, int64(len(news)), err
	case "category":
		if category, ok := filters["category"].(string); ok && category != "" && category != "all" {
			news, err := s.GetNewsByCategoryHot(category, pageSize)
			return news, int64(len(news)), err
		}
		return s.GetAllNews(page, pageSize)
	default:
		return s.GetAllNews(page, pageSize)
	}
}
