package services

import (
	"errors"
	"fmt" // 导入 fmt 包用于错误信息拼接
	"sort"
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
		Title:           req.Title,
		Content:         req.Content,
		Summary:         req.Summary,
		Source:          req.Source,
		Category:        req.Category,
		PublishedAt:     time.Now(),            // 默认设置为当前时间，如果请求中没有提供
		CreatedBy:       &createdByUserID,      // 设置创建者ID指针
		IsActive:        true,                  // 默认新闻是活跃的/可见的
		SourceType:      models.NewsTypeManual, // 手动创建的新闻
		BelongedEventID: req.BelongedEventID,   // 设置关联的事件ID
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
	if req.BelongedEventID != nil {
		news.BelongedEventID = req.BelongedEventID
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

// GetNewsByEventID 根据事件ID获取相关新闻列表（带分页，兼容不分页调用）
func (s *NewsService) GetNewsByEventID(eventID uint, args ...int) ([]models.News, int64, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, 0, errors.New("database connection not initialized")
	}

	var newsList []models.News
	var total int64

	// 构建查询
	dbQuery := s.db.Model(&models.News{}).Where("belonged_event_id = ?", eventID)

	// 计算符合条件的记录总数
	if err := dbQuery.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count news for event %d: %w", eventID, err)
	}

	// 如果提供了分页参数
	if len(args) >= 2 {
		page, pageSize := args[0], args[1]
		// 计算分页偏移量
		offset := (page - 1) * pageSize
		if offset < 0 {
			offset = 0
		}
		if pageSize <= 0 {
			pageSize = 10
		}

		// 执行带分页的查询，按发布时间降序排序（最新的优先）
		if err := dbQuery.Order("published_at DESC").Offset(offset).Limit(pageSize).Find(&newsList).Error; err != nil {
			return nil, 0, fmt.Errorf("failed to get news for event %d: %w", eventID, err)
		}
	} else {
		// 不分页，获取所有相关新闻
		if err := dbQuery.Order("published_at DESC").Find(&newsList).Error; err != nil {
			return nil, 0, fmt.Errorf("failed to get news for event %d: %w", eventID, err)
		}
	}

	return newsList, total, nil
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

// SmartSearchNews 智能搜索新闻，基于AI分析的关键词、标题和内容进行相似度匹配
func (s *NewsService) SmartSearchNews(query string, page, pageSize int) ([]models.News, int64, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, 0, errors.New("database connection not initialized")
	}

	var newsList []models.News
	var total int64

	// 构建智能搜索查询
	searchQuery := "%" + query + "%"

	// 基础查询：在标题、内容、摘要中搜索
	baseQuery := s.db.Model(&models.News{}).
		Where("title ILIKE ? OR content ILIKE ? OR summary ILIKE ?", searchQuery, searchQuery, searchQuery)

	// 获取所有有AI分析的新闻，检查关键词匹配
	var aiAnalysisResults []models.News
	aiQuery := `
		SELECT DISTINCT n.* FROM news n
		INNER JOIN ai_analyses a ON (
			(a.target_type = 'news' AND a.target_id = n.id) OR
			(a.target_type = 'event' AND n.belonged_event_id = a.target_id)
		)
		WHERE a.keywords ILIKE ? OR a.summary ILIKE ?
		ORDER BY n.published_at DESC
	`

	if err := s.db.Raw(aiQuery, searchQuery, searchQuery).Find(&aiAnalysisResults).Error; err != nil {
		// 如果AI查询失败，仅使用基础查询
		if err := baseQuery.Count(&total).Error; err != nil {
			return nil, 0, fmt.Errorf("failed to count search results: %w", err)
		}

		offset := (page - 1) * pageSize
		if offset < 0 {
			offset = 0
		}
		if pageSize <= 0 {
			pageSize = 10
		}

		if err := baseQuery.Offset(offset).Limit(pageSize).Order("published_at DESC").Find(&newsList).Error; err != nil {
			return nil, 0, fmt.Errorf("failed to search news: %w", err)
		}

		return newsList, total, nil
	}

	// 合并基础搜索和AI搜索结果
	var combinedResults []models.News
	newsMap := make(map[uint]models.News)

	// 添加基础搜索结果
	var baseResults []models.News
	if err := baseQuery.Find(&baseResults).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to search news: %w", err)
	}

	for _, news := range baseResults {
		newsMap[news.ID] = news
	}

	// 添加AI搜索结果
	for _, news := range aiAnalysisResults {
		newsMap[news.ID] = news
	}

	// 转换为切片并排序
	for _, news := range newsMap {
		combinedResults = append(combinedResults, news)
	}

	// 按发布时间排序
	sort.Slice(combinedResults, func(i, j int) bool {
		return combinedResults[i].PublishedAt.After(combinedResults[j].PublishedAt)
	})

	total = int64(len(combinedResults))

	// 分页
	offset := (page - 1) * pageSize
	if offset < 0 {
		offset = 0
	}
	if pageSize <= 0 {
		pageSize = 10
	}

	end := offset + pageSize
	if end > int(total) {
		end = int(total)
	}

	if offset >= int(total) {
		return []models.News{}, total, nil
	}

	newsList = combinedResults[offset:end]
	return newsList, total, nil
}

// GetHotKeywords 获取热门关键词，基于AI分析结果动态生成
func (s *NewsService) GetHotKeywords(limit int) ([]string, error) {
	// 检查数据库连接是否已初始化
	if s.db == nil {
		return nil, errors.New("database connection not initialized")
	}

	// 设置默认限制
	if limit <= 0 || limit > 50 {
		limit = 20
	}

	// 查询最近7天的AI分析结果
	var analysisResults []struct {
		Keywords string `json:"keywords"`
	}

	query := `
		SELECT keywords 
		FROM ai_analyses 
		WHERE status = 'completed' 
			AND keywords IS NOT NULL 
			AND keywords != '' 
			AND created_at >= NOW() - INTERVAL '7 days'
		ORDER BY created_at DESC
		LIMIT 1000
	`

	if err := s.db.Raw(query).Scan(&analysisResults).Error; err != nil {
		return nil, fmt.Errorf("failed to get AI analysis results: %w", err)
	}

	// 统计关键词频率
	keywordCount := make(map[string]int)

	for _, result := range analysisResults {
		if result.Keywords == "" {
			continue
		}

		// 解析关键词字符串
		var keywords []string
		if strings.Contains(result.Keywords, ",") {
			// 处理逗号分隔的关键词
			keywords = strings.Split(result.Keywords, ",")
		} else if strings.Contains(result.Keywords, "、") {
			// 处理中文顿号分隔的关键词
			keywords = strings.Split(result.Keywords, "、")
		} else if strings.Contains(result.Keywords, ";") {
			// 处理分号分隔的关键词
			keywords = strings.Split(result.Keywords, ";")
		} else {
			// 单个关键词
			keywords = []string{result.Keywords}
		}

		for _, keyword := range keywords {
			keyword = strings.TrimSpace(keyword)
			if keyword != "" && len(keyword) > 1 {
				keywordCount[keyword]++
			}
		}
	}

	// 排序关键词
	type keywordFreq struct {
		keyword string
		count   int
	}

	var sortedKeywords []keywordFreq
	for keyword, count := range keywordCount {
		sortedKeywords = append(sortedKeywords, keywordFreq{keyword: keyword, count: count})
	}

	sort.Slice(sortedKeywords, func(i, j int) bool {
		return sortedKeywords[i].count > sortedKeywords[j].count
	})

	// 返回前N个热门关键词
	var hotKeywords []string
	for i, kf := range sortedKeywords {
		if i >= limit {
			break
		}
		hotKeywords = append(hotKeywords, kf.keyword)
	}

	return hotKeywords, nil
}
