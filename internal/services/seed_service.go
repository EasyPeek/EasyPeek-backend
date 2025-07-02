package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/utils"
	"gorm.io/gorm"
)

/*
SeedService 种子数据服务

主要功能：
1. 导入新闻数据从JSON文件
2. 智能事件生成和关联
3. 创建默认管理员账户
4. 创建默认RSS源

集成的事件生成功能：
- 在导入新闻数据后自动分析相关性
- 使用AI API生成事件摘要（可配置）
- 智能分组：按类别和时间窗口（24小时）分组
- 自动关联：将相关新闻链接到生成的事件
- 可配置：支持启用/禁用事件生成功能

使用方式：
  seedService := NewSeedService()                    // 默认启用事件生成
  seedService := NewSeedServiceWithConfig(false)     // 禁用事件生成
  seedService.SetEventGeneration(true)               // 动态切换
  seedService.SeedAllData()                          // 导入数据（包含事件生成）
  seedService.SeedNewsFromJSONWithoutEvents("file") // 仅导入新闻
*/

type SeedService struct {
	db                    *gorm.DB
	enableEventGeneration bool
	aiConfig              *AIServiceConfig
}

// NewSeedService 创建新的种子数据服务实例
func NewSeedService() *SeedService {
	return &SeedService{
		db:                    database.GetDB(),
		enableEventGeneration: true, // 默认启用事件生成
		aiConfig:              DefaultAIConfig(),
	}
}

// NewSeedServiceWithConfig 创建带有配置的种子数据服务实例
func NewSeedServiceWithConfig(enableEventGeneration bool) *SeedService {
	return &SeedService{
		db:                    database.GetDB(),
		enableEventGeneration: enableEventGeneration,
		aiConfig:              DefaultAIConfig(),
	}
}

// NewSeedServiceWithAIConfig 创建带有自定义AI配置的种子数据服务实例
func NewSeedServiceWithAIConfig(aiConfig *AIServiceConfig) *SeedService {
	if aiConfig == nil {
		aiConfig = DefaultAIConfig()
	}
	return &SeedService{
		db:                    database.GetDB(),
		enableEventGeneration: aiConfig.EventGeneration.Enabled,
		aiConfig:              aiConfig,
	}
}

// NewsJSONData 定义JSON文件中的新闻数据结构
type NewsJSONData struct {
	Title        string  `json:"title"`
	Content      string  `json:"content"`
	Summary      string  `json:"summary"`
	Description  string  `json:"description"`
	Source       string  `json:"source"`
	Category     string  `json:"category"`
	PublishedAt  string  `json:"published_at"`
	CreatedBy    *uint   `json:"created_by"`
	IsActive     bool    `json:"is_active"`
	SourceType   string  `json:"source_type"`
	RSSSourceID  *uint   `json:"rss_source_id"`
	Link         string  `json:"link"`
	GUID         string  `json:"guid"`
	Author       string  `json:"author"`
	ImageURL     string  `json:"image_url"`
	Tags         string  `json:"tags"`
	Language     string  `json:"language"`
	ViewCount    int64   `json:"view_count"`
	LikeCount    int64   `json:"like_count"`
	CommentCount int64   `json:"comment_count"`
	ShareCount   int64   `json:"share_count"`
	HotnessScore float64 `json:"hotness_score"`
	Status       string  `json:"status"`
	IsProcessed  bool    `json:"is_processed"`
}

// SeedNewsFromJSON 从JSON文件导入新闻数据
func (s *SeedService) SeedNewsFromJSON(jsonFilePath string) error {
	log.Printf("开始从文件 %s 导入新闻数据...", jsonFilePath)

	// 检查数据库连接
	if s.db == nil {
		return fmt.Errorf("database connection not initialized")
	}

	// 检查现有新闻数据数量
	var count int64
	if err := s.db.Model(&models.News{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check existing news count: %w", err)
	}

	log.Printf("数据库中当前有 %d 条新闻记录，准备进行增量导入", count)

	// 读取JSON文件
	jsonData, err := os.ReadFile(jsonFilePath)
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %w", err)
	}

	// 解析JSON数据
	var newsDataList []NewsJSONData
	if err := json.Unmarshal(jsonData, &newsDataList); err != nil {
		return fmt.Errorf("failed to parse JSON data: %w", err)
	}

	log.Printf("成功解析JSON文件，找到 %d 条新闻记录", len(newsDataList))

	// 批量插入数据
	var newsList []models.News
	importedCount := 0
	skippedCount := 0

	for i, newsData := range newsDataList {
		// 解析发布时间
		publishedAt, err := time.Parse("2006-01-02 15:04:05", newsData.PublishedAt)
		if err != nil {
			log.Printf("警告：解析第 %d 条记录的发布时间失败，使用当前时间: %v", i+1, err)
			publishedAt = time.Now()
		}

		// 检查是否已存在相同GUID或链接的记录
		var existingNews models.News
		err = s.db.Where("guid = ? OR link = ?", newsData.GUID, newsData.Link).First(&existingNews).Error
		if err == nil {
			skippedCount++
			log.Printf("跳过重复记录：%s", newsData.Title)
			continue
		} else if err != gorm.ErrRecordNotFound {
			log.Printf("检查重复记录时出错：%v", err)
			continue
		}

		// 转换SourceType
		var sourceType models.NewsType = models.NewsTypeManual
		if newsData.SourceType == "rss" {
			sourceType = models.NewsTypeRSS
		}

		// 创建新闻记录
		news := models.News{
			Title:        newsData.Title,
			Content:      newsData.Content,
			Summary:      newsData.Summary,
			Description:  newsData.Description,
			Source:       newsData.Source,
			Category:     newsData.Category,
			PublishedAt:  publishedAt,
			CreatedBy:    newsData.CreatedBy,
			IsActive:     newsData.IsActive,
			SourceType:   sourceType,
			RSSSourceID:  newsData.RSSSourceID,
			Link:         newsData.Link,
			GUID:         newsData.GUID,
			Author:       newsData.Author,
			ImageURL:     newsData.ImageURL,
			Tags:         newsData.Tags,
			Language:     newsData.Language,
			ViewCount:    newsData.ViewCount,
			LikeCount:    newsData.LikeCount,
			CommentCount: newsData.CommentCount,
			ShareCount:   newsData.ShareCount,
			HotnessScore: newsData.HotnessScore,
			Status:       newsData.Status,
			IsProcessed:  newsData.IsProcessed,
		}

		newsList = append(newsList, news)
		importedCount++

		// 每100条记录批量插入一次，避免单次事务过大
		if len(newsList) >= 100 {
			if err := s.batchInsertNews(newsList); err != nil {
				return fmt.Errorf("failed to batch insert news: %w", err)
			}
			newsList = []models.News{} // 清空切片
		}
	}

	// 插入剩余的记录
	if len(newsList) > 0 {
		if err := s.batchInsertNews(newsList); err != nil {
			return fmt.Errorf("failed to insert remaining news: %w", err)
		}
	}

	log.Printf("新闻数据导入完成！成功导入 %d 条记录，跳过 %d 条重复记录", importedCount, skippedCount)

	// 如果导入了新的新闻数据且启用了事件生成，尝试生成事件
	if importedCount > 0 && s.enableEventGeneration {
		log.Println("开始为导入的新闻生成关联事件...")
		if err := s.GenerateEventsFromNewsWithDefaults(); err != nil {
			log.Printf("事件生成失败（但不影响新闻导入）: %v", err)
		}
	}

	return nil
}

// batchInsertNews 批量插入新闻记录
func (s *SeedService) batchInsertNews(newsList []models.News) error {
	if len(newsList) == 0 {
		return nil
	}

	// 使用事务进行批量插入
	return s.db.Transaction(func(tx *gorm.DB) error {
		// CreateInBatches 可以进行分批插入，避免单次插入过多数据
		if err := tx.CreateInBatches(newsList, 50).Error; err != nil {
			return err
		}
		return nil
	})
}

// SeedAllData 导入所有初始化数据
func (s *SeedService) SeedAllData() error {
	log.Println("开始初始化种子数据...")

	// 导入新闻数据（会自动生成事件如果启用了事件生成）
	if err := s.SeedNewsFromJSON("data/new.json"); err != nil {
		return fmt.Errorf("failed to seed news data: %w", err)
	}

	// 检查是否启用事件生成，如果启用则在启动时生成事件
	if s.enableEventGeneration {
		log.Println("🚀 启动时检查事件生成...")

		// 检查是否有未关联事件的新闻
		var unlinkedNewsCount int64
		if err := s.db.Model(&models.News{}).Where("belonged_event_id IS NULL").Count(&unlinkedNewsCount).Error; err != nil {
			log.Printf("检查未关联新闻失败: %v", err)
		} else if unlinkedNewsCount > 0 {
			log.Printf("发现 %d 条未关联事件的新闻，开始生成事件...", unlinkedNewsCount)
			if err := s.GenerateEventsFromNewsWithDefaults(); err != nil {
				log.Printf("启动时事件生成失败: %v", err)
			}
		} else {
			log.Println("所有新闻都已关联事件，跳过事件生成")
		}
	}

	// 在这里可以添加其他类型的数据导入，例如：
	// - 用户数据
	// - RSS源数据
	// - 事件数据等

	log.Println("所有种子数据初始化完成！")
	return nil
}

// SeedNewsFromJSONWithoutEvents 导入新闻数据但不生成事件
func (s *SeedService) SeedNewsFromJSONWithoutEvents(jsonFilePath string) error {
	originalSetting := s.enableEventGeneration
	s.enableEventGeneration = false
	defer func() {
		s.enableEventGeneration = originalSetting
	}()

	return s.SeedNewsFromJSON(jsonFilePath)
}

// SeedNewsFromJSONWithEvents 导入新闻数据并强制生成事件
func (s *SeedService) SeedNewsFromJSONWithEvents(jsonFilePath string) error {
	originalSetting := s.enableEventGeneration
	s.enableEventGeneration = true
	defer func() {
		s.enableEventGeneration = originalSetting
	}()

	return s.SeedNewsFromJSON(jsonFilePath)
}

// SetEventGeneration 设置是否启用事件生成
func (s *SeedService) SetEventGeneration(enable bool) {
	s.enableEventGeneration = enable
	if s.aiConfig != nil {
		s.aiConfig.EventGeneration.Enabled = enable
	}
	if enable {
		log.Println("事件生成功能已启用")
	} else {
		log.Println("事件生成功能已禁用")
	}
}

// SetAIAPIKey 设置AI API密钥
func (s *SeedService) SetAIAPIKey(apiKey string) {
	if s.aiConfig == nil {
		s.aiConfig = DefaultAIConfig()
	}
	s.aiConfig.APIKey = apiKey
	log.Println("AI API密钥已设置")
}

// SetAIEndpoint 设置AI API端点
func (s *SeedService) SetAIEndpoint(endpoint string) {
	if s.aiConfig == nil {
		s.aiConfig = DefaultAIConfig()
	}
	s.aiConfig.APIEndpoint = endpoint
	log.Printf("AI API端点已设置为: %s", endpoint)
}

// SetAIModel 设置AI模型
func (s *SeedService) SetAIModel(model string) {
	if s.aiConfig == nil {
		s.aiConfig = DefaultAIConfig()
	}
	s.aiConfig.Model = model
	log.Printf("AI模型已设置为: %s", model)
}

// SetAIProvider 设置AI提供商
func (s *SeedService) SetAIProvider(provider string) {
	if s.aiConfig == nil {
		s.aiConfig = DefaultAIConfig()
	}
	s.aiConfig.Provider = provider
	log.Printf("AI提供商已设置为: %s", provider)
}

// GetAIConfig 获取AI配置
func (s *SeedService) GetAIConfig() *AIServiceConfig {
	if s.aiConfig == nil {
		s.aiConfig = DefaultAIConfig()
	}
	return s.aiConfig
}

// UpdateAIConfig 更新AI配置
func (s *SeedService) UpdateAIConfig(config *AIServiceConfig) {
	if config != nil {
		s.aiConfig = config
		s.enableEventGeneration = config.EventGeneration.Enabled
		log.Println("AI配置已更新")
	}
}

// SeedInitialAdmin 创建初始管理员账户
func (s *SeedService) SeedInitialAdmin() error {
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	// 检查是否已经存在管理员账户
	var adminCount int64
	if err := s.db.Model(&models.User{}).Where("role = ?", "admin").Count(&adminCount).Error; err != nil {
		return err
	}

	// 如果已经存在管理员，不需要创建
	if adminCount > 0 {
		log.Println("Admin account already exists, skipping seed")
		return nil
	}

	// 从环境变量或默认值获取管理员信息
	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "admin@easypeek.com"
	}

	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "admin123456"
	}

	adminUsername := os.Getenv("ADMIN_USERNAME")
	if adminUsername == "" {
		adminUsername = "admin"
	}

	// 验证输入
	if !utils.IsValidEmail(adminEmail) {
		return errors.New("invalid admin email format")
	}

	if !utils.IsValidPassword(adminPassword) {
		return errors.New("admin password must contain at least one letter and one number")
	}

	if !utils.IsValidUsername(adminUsername) {
		return errors.New("invalid admin username format")
	}

	// 检查邮箱和用户名是否已存在
	var existingUser models.User
	if err := s.db.Where("email = ? OR username = ?", adminEmail, adminUsername).First(&existingUser).Error; err == nil {
		return errors.New("admin email or username already exists")
	}

	// 创建管理员账户
	adminUser := &models.User{
		Username: adminUsername,
		Email:    adminEmail,
		Password: adminPassword, // 会被 BeforeCreate hook 自动加密
		Role:     "admin",
		Status:   "active",
	}

	if err := s.db.Create(adminUser).Error; err != nil {
		return err
	}

	log.Printf("Initial admin account created successfully:")
	log.Printf("- Username: %s", adminUsername)
	log.Printf("- Email: %s", adminEmail)
	log.Printf("- Password: %s", adminPassword)
	log.Println("Please change the default password after first login!")

	return nil
}

// SeedDefaultData 种子数据初始化
func (s *SeedService) SeedDefaultData() error {
	// 创建初始管理员
	if err := s.SeedInitialAdmin(); err != nil {
		return err
	}

	// 可以在这里添加其他默认数据的初始化
	// 例如：默认分类、默认RSS源等

	return nil
}

// SeedRSSources 创建默认RSS源（可选）
func (s *SeedService) SeedRSSources() error {
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	// 检查是否已经存在RSS源
	var rssCount int64
	if err := s.db.Model(&models.RSSSource{}).Count(&rssCount).Error; err != nil {
		return err
	}

	// 如果已经存在RSS源，不需要创建
	if rssCount > 0 {
		log.Println("RSS sources already exist, skipping seed")
		return nil
	}

	// 创建一些默认的RSS源
	defaultSources := []models.RSSSource{
		{
			Name:        "新浪新闻",
			URL:         "http://rss.sina.com.cn/news/china/focus15.xml",
			Category:    "国内新闻",
			Language:    "zh",
			IsActive:    true,
			Description: "新浪网国内新闻RSS源",
			Priority:    1,
			UpdateFreq:  60,
		},
		{
			Name:        "网易科技",
			URL:         "http://rss.163.com/rss/tech_index.xml",
			Category:    "科技",
			Language:    "zh",
			IsActive:    true,
			Description: "网易科技新闻RSS源",
			Priority:    1,
			UpdateFreq:  60,
		},
	}

	for _, source := range defaultSources {
		if err := s.db.Create(&source).Error; err != nil {
			log.Printf("Failed to create RSS source %s: %v", source.Name, err)
		} else {
			log.Printf("Created default RSS source: %s", source.Name)
		}
	}

	return nil
}

// AIServiceConfig AI服务配置（内置在服务中）
type AIServiceConfig struct {
	// AI服务提供商: openai, baidu, custom
	Provider string `json:"provider"`

	// API密钥
	APIKey string `json:"api_key"`

	// API端点
	APIEndpoint string `json:"api_endpoint"`

	// 使用的模型
	Model string `json:"model"`

	// 最大token数
	MaxTokens int `json:"max_tokens"`

	// 超时时间(秒)
	Timeout int `json:"timeout"`

	// 是否启用AI功能
	Enabled bool `json:"enabled"`

	// 事件生成相关配置
	EventGeneration struct {
		// 是否启用自动事件生成
		Enabled bool `json:"enabled"`

		// 置信度阈值 (0.0-1.0)
		ConfidenceThreshold float64 `json:"confidence_threshold"`

		// 最小新闻数量才生成事件
		MinNewsCount int `json:"min_news_count"`

		// 时间窗口(小时)
		TimeWindowHours int `json:"time_window_hours"`

		// 最大处理新闻数量
		MaxNewsLimit int `json:"max_news_limit"`
	} `json:"event_generation"`
}

// EventGenerationConfig AI事件生成配置（向后兼容）
type EventGenerationConfig struct {
	APIKey      string `json:"api_key"`
	APIEndpoint string `json:"api_endpoint"`
	Model       string `json:"model"`
	MaxTokens   int    `json:"max_tokens"`
}

// DefaultAIConfig 获取默认AI配置（从环境变量加载）
func DefaultAIConfig() *AIServiceConfig {
	// 从环境变量读取API密钥
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Println("警告：未设置 OPENAI_API_KEY 环境变量，AI功能将使用模拟模式")
	}

	// 从环境变量读取其他配置，如果没有设置则使用默认值
	provider := os.Getenv("AI_PROVIDER")
	if provider == "" {
		provider = "openai"
	}

	endpoint := os.Getenv("OPENAI_API_ENDPOINT")
	if endpoint == "" {
		endpoint = "https://api.openai.com/v1/chat/completions"
	}

	model := os.Getenv("OPENAI_MODEL")
	if model == "" {
		model = "gpt-3.5-turbo"
	}

	config := &AIServiceConfig{
		Provider:    provider,
		APIKey:      apiKey,
		APIEndpoint: endpoint,
		Model:       model,
		MaxTokens:   2000,
		Timeout:     30,
		Enabled:     true,
	}

	// 事件生成配置
	config.EventGeneration.Enabled = true
	config.EventGeneration.ConfidenceThreshold = 0.6
	config.EventGeneration.MinNewsCount = 2
	config.EventGeneration.TimeWindowHours = 24
	config.EventGeneration.MaxNewsLimit = 50

	return config
}

// NewsEventMapping 新闻事件映射结构
type NewsEventMapping struct {
	NewsIDs   []uint `json:"news_ids"`
	EventData struct {
		Title        string   `json:"title"`
		Description  string   `json:"description"`
		Content      string   `json:"content"`
		Category     string   `json:"category"`
		Tags         []string `json:"tags"`
		Location     string   `json:"location"`
		Source       string   `json:"source"`
		Author       string   `json:"author"`
		RelatedLinks []string `json:"related_links"`
	} `json:"event_data"`
}

// AIEventSummaryRequest AI事件总结请求
type AIEventSummaryRequest struct {
	NewsArticles []struct {
		Title       string `json:"title"`
		Content     string `json:"content"`
		Summary     string `json:"summary"`
		Description string `json:"description"`
		Source      string `json:"source"`
		Category    string `json:"category"`
		PublishedAt string `json:"published_at"`
	} `json:"news_articles"`
	Prompt string `json:"prompt"`
}

// AIEventSummaryResponse AI事件总结响应
type AIEventSummaryResponse struct {
	Success bool `json:"success"`
	Data    struct {
		Title        string   `json:"title"`
		Description  string   `json:"description"`
		Content      string   `json:"content"`
		Category     string   `json:"category"`
		Tags         []string `json:"tags"`
		Location     string   `json:"location"`
		StartTime    string   `json:"start_time"`
		EndTime      string   `json:"end_time"`
		Source       string   `json:"source"`
		Author       string   `json:"author"`
		RelatedLinks []string `json:"related_links"`
		Confidence   float64  `json:"confidence"`
	} `json:"data"`
	Message string `json:"message"`
}

// GenerateEventsFromNews 从新闻生成事件并关联
func (s *SeedService) GenerateEventsFromNews(config EventGenerationConfig) error {
	log.Println("开始从新闻生成事件...")

	if s.db == nil {
		return fmt.Errorf("database connection not initialized")
	}

	// 获取所有未关联事件的新闻（包含所有来源类型）
	var newsList []models.News
	if err := s.db.Where("belonged_event_id IS NULL").
		Order("published_at DESC").
		Limit(50). // 限制处理数量，避免一次处理太多
		Find(&newsList).Error; err != nil {
		return fmt.Errorf("failed to fetch unassigned news: %w", err)
	}

	if len(newsList) == 0 {
		log.Println("没有找到需要处理的新闻")
		return nil
	}

	log.Printf("找到 %d 条未关联事件的新闻", len(newsList))

	// 按类别和时间分组新闻
	newsGroups := s.groupNewsByCategory(newsList)

	generatedCount := 0
	for category, categoryNews := range newsGroups {
		log.Printf("处理分类: %s, 新闻数量: %d", category, len(categoryNews))

		// 为每个分组生成事件
		eventMapping, err := s.generateEventFromNewsGroup(categoryNews, config)
		if err != nil {
			log.Printf("为分类 %s 生成事件失败: %v", category, err)
			continue
		}

		if eventMapping == nil {
			log.Printf("分类 %s 不需要生成事件", category)
			continue
		}

		// 创建事件并关联新闻
		if err := s.createEventAndLinkNews(eventMapping); err != nil {
			log.Printf("创建事件并关联新闻失败: %v", err)
			continue
		}

		generatedCount++
	}

	log.Printf("事件生成完成！成功生成 %d 个事件", generatedCount)
	return nil
}

// groupNewsByCategory 按类别和时间窗口分组新闻
func (s *SeedService) groupNewsByCategory(newsList []models.News) map[string][]models.News {
	groups := make(map[string][]models.News)

	for _, news := range newsList {
		category := news.Category
		if category == "" {
			category = "其他"
		}

		groups[category] = append(groups[category], news)
	}

	// 进一步按时间窗口分组（同一分类下的新闻如果时间相近，可能属于同一事件）
	refinedGroups := make(map[string][]models.News)
	for category, categoryNews := range groups {
		timeGroups := s.groupNewsByTimeWindow(categoryNews, 24*time.Hour) // 24小时时间窗口

		for i, timeGroup := range timeGroups {
			if len(timeGroup) >= 2 { // 至少2条新闻才考虑生成事件
				key := fmt.Sprintf("%s_%d", category, i)
				refinedGroups[key] = timeGroup
			}
		}
	}

	return refinedGroups
}

// groupNewsByCategoryWithTimeWindow 按类别和时间窗口分组新闻（带配置参数）
func (s *SeedService) groupNewsByCategoryWithTimeWindow(newsList []models.News, timeWindow time.Duration, minNewsCount int) map[string][]models.News {
	groups := make(map[string][]models.News)

	for _, news := range newsList {
		category := news.Category
		if category == "" {
			category = "其他"
		}

		groups[category] = append(groups[category], news)
	}

	// 进一步按时间窗口分组
	refinedGroups := make(map[string][]models.News)
	for category, categoryNews := range groups {
		timeGroups := s.groupNewsByTimeWindow(categoryNews, timeWindow)

		for i, timeGroup := range timeGroups {
			if len(timeGroup) >= minNewsCount { // 使用配置中的最小新闻数量
				key := fmt.Sprintf("%s_%d", category, i)
				refinedGroups[key] = timeGroup
			}
		}
	}

	return refinedGroups
}

// groupNewsByTimeWindow 按时间窗口分组新闻
func (s *SeedService) groupNewsByTimeWindow(newsList []models.News, window time.Duration) [][]models.News {
	if len(newsList) == 0 {
		return nil
	}

	// 按发布时间排序
	for i := 0; i < len(newsList)-1; i++ {
		for j := i + 1; j < len(newsList); j++ {
			if newsList[i].PublishedAt.After(newsList[j].PublishedAt) {
				newsList[i], newsList[j] = newsList[j], newsList[i]
			}
		}
	}

	var groups [][]models.News
	var currentGroup []models.News
	var windowStart time.Time

	for _, news := range newsList {
		if len(currentGroup) == 0 {
			// 开始新组
			currentGroup = []models.News{news}
			windowStart = news.PublishedAt
		} else if news.PublishedAt.Sub(windowStart) <= window {
			// 在时间窗口内，加入当前组
			currentGroup = append(currentGroup, news)
		} else {
			// 超出时间窗口，保存当前组并开始新组
			if len(currentGroup) > 0 {
				groups = append(groups, currentGroup)
			}
			currentGroup = []models.News{news}
			windowStart = news.PublishedAt
		}
	}

	// 保存最后一组
	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	return groups
}

// generateEventFromNewsGroup 从新闻组生成事件
func (s *SeedService) generateEventFromNewsGroup(newsList []models.News, config EventGenerationConfig) (*NewsEventMapping, error) {
	if len(newsList) < 2 {
		return nil, nil // 新闻太少，不生成事件
	}

	// 准备AI请求数据
	var newsArticles []struct {
		Title       string `json:"title"`
		Content     string `json:"content"`
		Summary     string `json:"summary"`
		Description string `json:"description"`
		Source      string `json:"source"`
		Category    string `json:"category"`
		PublishedAt string `json:"published_at"`
	}

	for _, news := range newsList {
		newsArticles = append(newsArticles, struct {
			Title       string `json:"title"`
			Content     string `json:"content"`
			Summary     string `json:"summary"`
			Description string `json:"description"`
			Source      string `json:"source"`
			Category    string `json:"category"`
			PublishedAt string `json:"published_at"`
		}{
			Title:       news.Title,
			Content:     news.Content,
			Summary:     news.Summary,
			Description: news.Description,
			Source:      news.Source,
			Category:    news.Category,
			PublishedAt: news.PublishedAt.Format("2006-01-02 15:04:05"),
		})
	}

	// 构建AI提示词
	prompt := `请分析以下新闻文章，判断是否可以归纳为一个事件。如果可以，请提取事件信息：

分析要求：
1. 判断这些新闻是否描述同一个事件或相关事件
2. 如果是，请提取事件的核心信息
3. 生成合适的事件标题、描述和详细内容
4. 推断事件的开始和结束时间
5. 提取事件地点、相关标签等信息

返回格式为JSON：
{
  "title": "事件标题",
  "description": "事件简要描述",
  "content": "事件详细内容",
  "category": "事件分类",
  "tags": ["标签1", "标签2"],
  "location": "事件地点",
  "start_time": "2024-01-01 00:00:00",
  "end_time": "2024-01-02 00:00:00",
  "source": "主要信息源",
  "author": "主要作者",
  "related_links": ["相关链接"],
  "confidence": 0.8
}

如果这些新闻不构成一个明确的事件，请返回：{"confidence": 0}

请分析以下新闻：`

	// 调用AI API
	aiResponse, err := s.callAIAPI(AIEventSummaryRequest{
		NewsArticles: newsArticles,
		Prompt:       prompt,
	}, config)

	if err != nil {
		return nil, fmt.Errorf("AI API调用失败: %w", err)
	}

	// 检查置信度
	if aiResponse.Data.Confidence < 0.6 {
		log.Printf("AI置信度不足 (%.2f)，跳过事件生成", aiResponse.Data.Confidence)
		return nil, nil
	}

	// 构建事件映射
	var newsIDs []uint
	for _, news := range newsList {
		newsIDs = append(newsIDs, news.ID)
	}

	mapping := &NewsEventMapping{
		NewsIDs: newsIDs,
	}

	mapping.EventData.Title = aiResponse.Data.Title
	mapping.EventData.Description = aiResponse.Data.Description
	mapping.EventData.Content = aiResponse.Data.Content
	mapping.EventData.Category = aiResponse.Data.Category
	mapping.EventData.Tags = aiResponse.Data.Tags
	mapping.EventData.Location = aiResponse.Data.Location
	mapping.EventData.Source = aiResponse.Data.Source
	mapping.EventData.Author = aiResponse.Data.Author
	mapping.EventData.RelatedLinks = aiResponse.Data.RelatedLinks

	return mapping, nil
}

// generateEventFromNewsGroupWithAI 使用AI配置从新闻组生成事件
func (s *SeedService) generateEventFromNewsGroupWithAI(newsList []models.News, aiConfig *AIServiceConfig) (*NewsEventMapping, error) {
	if len(newsList) < aiConfig.EventGeneration.MinNewsCount {
		return nil, nil // 新闻数量不足
	}

	// 准备AI请求数据
	var newsArticles []struct {
		Title       string `json:"title"`
		Content     string `json:"content"`
		Summary     string `json:"summary"`
		Description string `json:"description"`
		Source      string `json:"source"`
		Category    string `json:"category"`
		PublishedAt string `json:"published_at"`
	}

	for _, news := range newsList {
		newsArticles = append(newsArticles, struct {
			Title       string `json:"title"`
			Content     string `json:"content"`
			Summary     string `json:"summary"`
			Description string `json:"description"`
			Source      string `json:"source"`
			Category    string `json:"category"`
			PublishedAt string `json:"published_at"`
		}{
			Title:       news.Title,
			Content:     news.Content,
			Summary:     news.Summary,
			Description: news.Description,
			Source:      news.Source,
			Category:    news.Category,
			PublishedAt: news.PublishedAt.Format("2006-01-02 15:04:05"),
		})
	}

	// 构建AI提示词
	prompt := `请分析以下新闻文章，判断是否可以归纳为一个事件。如果可以，请提取事件信息：

分析要求：
1. 判断这些新闻是否描述同一个事件或相关事件
2. 如果是，请提取事件的核心信息
3. 生成合适的事件标题、描述和详细内容
4. 推断事件的开始和结束时间
5. 提取事件地点、相关标签等信息

返回格式为JSON：
{
  "title": "事件标题",
  "description": "事件简要描述",
  "content": "事件详细内容",
  "category": "事件分类",
  "tags": ["标签1", "标签2"],
  "location": "事件地点",
  "start_time": "2024-01-01 00:00:00",
  "end_time": "2024-01-02 00:00:00",
  "source": "主要信息源",
  "author": "主要作者",
  "related_links": ["相关链接"],
  "confidence": 0.8
}

如果这些新闻不构成一个明确的事件，请返回：{"confidence": 0}

请分析以下新闻：`

	// 调用AI API（使用新的配置）
	aiResponse, err := s.callAIAPIWithConfig(AIEventSummaryRequest{
		NewsArticles: newsArticles,
		Prompt:       prompt,
	}, aiConfig)

	if err != nil {
		return nil, fmt.Errorf("AI API调用失败: %w", err)
	}

	// 检查置信度（使用配置中的阈值）
	if aiResponse.Data.Confidence < aiConfig.EventGeneration.ConfidenceThreshold {
		log.Printf("AI置信度不足 (%.2f < %.2f)，跳过事件生成",
			aiResponse.Data.Confidence, aiConfig.EventGeneration.ConfidenceThreshold)
		return nil, nil
	}

	// 构建事件映射
	var newsIDs []uint
	for _, news := range newsList {
		newsIDs = append(newsIDs, news.ID)
	}

	mapping := &NewsEventMapping{
		NewsIDs: newsIDs,
	}

	mapping.EventData.Title = aiResponse.Data.Title
	mapping.EventData.Description = aiResponse.Data.Description
	mapping.EventData.Content = aiResponse.Data.Content
	mapping.EventData.Category = aiResponse.Data.Category
	mapping.EventData.Tags = aiResponse.Data.Tags
	mapping.EventData.Location = aiResponse.Data.Location
	mapping.EventData.Source = aiResponse.Data.Source
	mapping.EventData.Author = aiResponse.Data.Author
	mapping.EventData.RelatedLinks = aiResponse.Data.RelatedLinks

	return mapping, nil
}

// callAIAPI 调用AI API进行事件总结
func (s *SeedService) callAIAPI(request AIEventSummaryRequest, config EventGenerationConfig) (*AIEventSummaryResponse, error) {
	// 这里可以集成实际的AI API，例如OpenAI、百度文心等
	// 目前提供一个模拟实现

	log.Println("调用AI API进行事件分析...")

	// 模拟AI分析逻辑
	if len(request.NewsArticles) < 2 {
		return &AIEventSummaryResponse{
			Success: false,
			Data: struct {
				Title        string   `json:"title"`
				Description  string   `json:"description"`
				Content      string   `json:"content"`
				Category     string   `json:"category"`
				Tags         []string `json:"tags"`
				Location     string   `json:"location"`
				StartTime    string   `json:"start_time"`
				EndTime      string   `json:"end_time"`
				Source       string   `json:"source"`
				Author       string   `json:"author"`
				RelatedLinks []string `json:"related_links"`
				Confidence   float64  `json:"confidence"`
			}{Confidence: 0.0},
			Message: "新闻数量不足",
		}, nil
	}

	// 简单的规则式分析（实际应用中应替换为真实的AI API调用）
	firstNews := request.NewsArticles[0]

	// 计算标题相似度（简单实现）
	similarity := s.calculateTitleSimilarity(request.NewsArticles)

	if similarity < 0.3 {
		return &AIEventSummaryResponse{
			Success: true,
			Data: struct {
				Title        string   `json:"title"`
				Description  string   `json:"description"`
				Content      string   `json:"content"`
				Category     string   `json:"category"`
				Tags         []string `json:"tags"`
				Location     string   `json:"location"`
				StartTime    string   `json:"start_time"`
				EndTime      string   `json:"end_time"`
				Source       string   `json:"source"`
				Author       string   `json:"author"`
				RelatedLinks []string `json:"related_links"`
				Confidence   float64  `json:"confidence"`
			}{Confidence: 0.2},
			Message: "新闻相关性不足",
		}, nil
	}

	// 生成事件信息
	now := time.Now()
	response := &AIEventSummaryResponse{
		Success: true,
		Data: struct {
			Title        string   `json:"title"`
			Description  string   `json:"description"`
			Content      string   `json:"content"`
			Category     string   `json:"category"`
			Tags         []string `json:"tags"`
			Location     string   `json:"location"`
			StartTime    string   `json:"start_time"`
			EndTime      string   `json:"end_time"`
			Source       string   `json:"source"`
			Author       string   `json:"author"`
			RelatedLinks []string `json:"related_links"`
			Confidence   float64  `json:"confidence"`
		}{
			Title:        fmt.Sprintf("%s相关事件", firstNews.Category),
			Description:  fmt.Sprintf("基于%d条新闻总结的%s事件", len(request.NewsArticles), firstNews.Category),
			Content:      s.generateEventContent(request.NewsArticles),
			Category:     firstNews.Category,
			Tags:         []string{firstNews.Category, "AI生成"},
			Location:     "待确定",
			StartTime:    now.Add(-24 * time.Hour).Format("2006-01-02 15:04:05"),
			EndTime:      now.Add(24 * time.Hour).Format("2006-01-02 15:04:05"),
			Source:       firstNews.Source,
			Author:       "系统生成",
			RelatedLinks: []string{},
			Confidence:   0.7,
		},
		Message: "事件生成成功",
	}

	return response, nil
}

// callAIAPIWithConfig 使用AI配置调用AI API进行事件总结
func (s *SeedService) callAIAPIWithConfig(request AIEventSummaryRequest, aiConfig *AIServiceConfig) (*AIEventSummaryResponse, error) {
	// 这里可以集成实际的AI API，例如OpenAI、百度文心等
	// 目前提供一个基于配置的模拟实现

	log.Printf("调用AI API进行事件分析... (Provider: %s, Model: %s)", aiConfig.Provider, aiConfig.Model)

	// 检查API密钥
	if aiConfig.APIKey == "" || aiConfig.APIKey == "your-openai-api-key-here" {
		log.Println("警告：AI API密钥未设置，使用模拟分析")
	}

	// 模拟AI分析逻辑
	if len(request.NewsArticles) < aiConfig.EventGeneration.MinNewsCount {
		return &AIEventSummaryResponse{
			Success: false,
			Data: struct {
				Title        string   `json:"title"`
				Description  string   `json:"description"`
				Content      string   `json:"content"`
				Category     string   `json:"category"`
				Tags         []string `json:"tags"`
				Location     string   `json:"location"`
				StartTime    string   `json:"start_time"`
				EndTime      string   `json:"end_time"`
				Source       string   `json:"source"`
				Author       string   `json:"author"`
				RelatedLinks []string `json:"related_links"`
				Confidence   float64  `json:"confidence"`
			}{Confidence: 0.0},
			Message: "新闻数量不足",
		}, nil
	}

	// 简单的规则式分析（实际应用中应替换为真实的AI API调用）
	firstNews := request.NewsArticles[0]

	// 计算标题相似度（简单实现）
	similarity := s.calculateTitleSimilarity(request.NewsArticles)

	// 根据相似度和配置的置信度阈值判断
	threshold := aiConfig.EventGeneration.ConfidenceThreshold * 0.5 // 降低内部检查阈值
	if similarity < threshold {
		return &AIEventSummaryResponse{
			Success: true,
			Data: struct {
				Title        string   `json:"title"`
				Description  string   `json:"description"`
				Content      string   `json:"content"`
				Category     string   `json:"category"`
				Tags         []string `json:"tags"`
				Location     string   `json:"location"`
				StartTime    string   `json:"start_time"`
				EndTime      string   `json:"end_time"`
				Source       string   `json:"source"`
				Author       string   `json:"author"`
				RelatedLinks []string `json:"related_links"`
				Confidence   float64  `json:"confidence"`
			}{Confidence: similarity},
			Message: "新闻相关性不足",
		}, nil
	}

	// 生成事件信息
	now := time.Now()
	confidence := 0.7 + similarity*0.3 // 基于相似度调整置信度

	response := &AIEventSummaryResponse{
		Success: true,
		Data: struct {
			Title        string   `json:"title"`
			Description  string   `json:"description"`
			Content      string   `json:"content"`
			Category     string   `json:"category"`
			Tags         []string `json:"tags"`
			Location     string   `json:"location"`
			StartTime    string   `json:"start_time"`
			EndTime      string   `json:"end_time"`
			Source       string   `json:"source"`
			Author       string   `json:"author"`
			RelatedLinks []string `json:"related_links"`
			Confidence   float64  `json:"confidence"`
		}{
			Title:        fmt.Sprintf("%s相关事件", firstNews.Category),
			Description:  fmt.Sprintf("基于%d条新闻总结的%s事件", len(request.NewsArticles), firstNews.Category),
			Content:      s.generateEventContent(request.NewsArticles),
			Category:     firstNews.Category,
			Tags:         []string{firstNews.Category, "AI生成", aiConfig.Provider},
			Location:     "待确定",
			StartTime:    now.Add(-time.Duration(aiConfig.EventGeneration.TimeWindowHours) * time.Hour).Format("2006-01-02 15:04:05"),
			EndTime:      now.Add(24 * time.Hour).Format("2006-01-02 15:04:05"),
			Source:       firstNews.Source,
			Author:       fmt.Sprintf("AI生成 (%s)", aiConfig.Model),
			RelatedLinks: []string{},
			Confidence:   confidence,
		},
		Message: fmt.Sprintf("事件生成成功 (置信度: %.2f)", confidence),
	}

	return response, nil
}

// calculateTitleSimilarity 计算标题相似度（简单实现）
func (s *SeedService) calculateTitleSimilarity(articles []struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Source      string `json:"source"`
	Category    string `json:"category"`
	PublishedAt string `json:"published_at"`
}) float64 {
	if len(articles) < 2 {
		return 0.0
	}

	// 简单的关键词重叠率计算
	keywords := make(map[string]int)

	for _, article := range articles {
		// 提取标题关键词（简单分词）
		words := []string{} // 这里应该使用更好的分词算法
		for _, char := range article.Title {
			if char > 127 { // 简单判断中文字符
				words = append(words, string(char))
			}
		}

		for _, word := range words {
			if len(word) > 0 {
				keywords[word]++
			}
		}
	}

	// 计算重叠度
	overlap := 0
	total := 0
	for _, count := range keywords {
		total++
		if count > 1 {
			overlap++
		}
	}

	if total == 0 {
		return 0.0
	}

	return float64(overlap) / float64(total)
}

// generateEventContent 生成事件内容
func (s *SeedService) generateEventContent(articles []struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Source      string `json:"source"`
	Category    string `json:"category"`
	PublishedAt string `json:"published_at"`
}) string {
	content := "# 事件总结\n\n"

	content += "## 相关新闻\n\n"
	for i, article := range articles {
		content += fmt.Sprintf("### %d. %s\n", i+1, article.Title)
		content += fmt.Sprintf("**来源**: %s | **发布时间**: %s\n\n", article.Source, article.PublishedAt)

		if article.Summary != "" {
			content += fmt.Sprintf("**摘要**: %s\n\n", article.Summary)
		} else if article.Description != "" {
			content += fmt.Sprintf("**描述**: %s\n\n", article.Description)
		}

		content += "---\n\n"
	}

	content += "## 事件分析\n\n"
	content += "本事件由系统AI自动分析多条相关新闻生成，汇总了相关的新闻报道和信息。\n\n"

	return content
}

// createEventAndLinkNews 创建事件并关联新闻
func (s *SeedService) createEventAndLinkNews(mapping *NewsEventMapping) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		// 解析时间
		startTime, err := time.Parse("2006-01-02 15:04:05", "2024-01-01 00:00:00")
		if err != nil {
			startTime = time.Now().Add(-24 * time.Hour)
		}

		endTime, err := time.Parse("2006-01-02 15:04:05", "2024-01-02 00:00:00")
		if err != nil {
			endTime = time.Now().Add(24 * time.Hour)
		}

		// 创建事件
		event := models.Event{
			Title:        mapping.EventData.Title,
			Description:  mapping.EventData.Description,
			Content:      mapping.EventData.Content,
			StartTime:    startTime,
			EndTime:      endTime,
			Location:     mapping.EventData.Location,
			Status:       "进行中",
			CreatedBy:    1, // 系统管理员ID
			Category:     mapping.EventData.Category,
			Tags:         s.tagsToJSON(mapping.EventData.Tags),
			Source:       mapping.EventData.Source,
			Author:       mapping.EventData.Author,
			RelatedLinks: s.linksToJSON(mapping.EventData.RelatedLinks),
		}

		if err := tx.Create(&event).Error; err != nil {
			return fmt.Errorf("创建事件失败: %w", err)
		}

		log.Printf("成功创建事件: %s (ID: %d)", event.Title, event.ID)

		// 更新新闻关联
		if err := tx.Model(&models.News{}).
			Where("id IN ?", mapping.NewsIDs).
			Update("belonged_event_id", event.ID).Error; err != nil {
			return fmt.Errorf("关联新闻失败: %w", err)
		}

		log.Printf("成功关联 %d 条新闻到事件 %d", len(mapping.NewsIDs), event.ID)
		return nil
	})
}

// tagsToJSON 将标签数组转换为JSON字符串
func (s *SeedService) tagsToJSON(tags []string) string {
	if len(tags) == 0 {
		return "[]"
	}

	jsonData, err := json.Marshal(tags)
	if err != nil {
		return "[]"
	}

	return string(jsonData)
}

// linksToJSON 将链接数组转换为JSON字符串
func (s *SeedService) linksToJSON(links []string) string {
	if len(links) == 0 {
		return "[]"
	}

	jsonData, err := json.Marshal(links)
	if err != nil {
		return "[]"
	}

	return string(jsonData)
}

// GenerateEventsFromNewsWithDefaults 使用内置配置生成事件
func (s *SeedService) GenerateEventsFromNewsWithDefaults() error {
	if s.aiConfig == nil {
		s.aiConfig = DefaultAIConfig()
	}

	// 检查AI功能是否启用
	if !s.aiConfig.Enabled || !s.aiConfig.EventGeneration.Enabled {
		log.Println("AI事件生成功能未启用，跳过事件生成")
		return nil
	}

	// 使用内置配置
	config := EventGenerationConfig{
		APIKey:      s.aiConfig.APIKey,
		APIEndpoint: s.aiConfig.APIEndpoint,
		Model:       s.aiConfig.Model,
		MaxTokens:   s.aiConfig.MaxTokens,
	}

	return s.GenerateEventsFromNews(config)
}

// GenerateEventsFromNewsWithAIConfig 使用AI配置生成事件
func (s *SeedService) GenerateEventsFromNewsWithAIConfig() error {
	if s.aiConfig == nil {
		s.aiConfig = DefaultAIConfig()
	}

	return s.GenerateEventsFromNewsWithAISettings(s.aiConfig)
}

// GenerateEventsFromNewsWithAISettings 使用指定的AI设置生成事件
func (s *SeedService) GenerateEventsFromNewsWithAISettings(aiConfig *AIServiceConfig) error {
	log.Println("开始从新闻生成事件（使用AI配置）...")

	if s.db == nil {
		return fmt.Errorf("database connection not initialized")
	}

	if aiConfig == nil || !aiConfig.Enabled || !aiConfig.EventGeneration.Enabled {
		log.Println("AI事件生成功能未启用")
		return nil
	}

	// 获取所有未关联事件的新闻（包含所有来源类型），使用配置中的限制
	var newsList []models.News
	if err := s.db.Where("belonged_event_id IS NULL").
		Order("published_at DESC").
		Limit(aiConfig.EventGeneration.MaxNewsLimit).
		Find(&newsList).Error; err != nil {
		return fmt.Errorf("failed to fetch unassigned news: %w", err)
	}

	if len(newsList) == 0 {
		log.Println("没有找到需要处理的新闻")
		return nil
	}

	log.Printf("找到 %d 条未关联事件的新闻", len(newsList))

	// 按类别和时间分组新闻，使用配置中的时间窗口
	timeWindow := time.Duration(aiConfig.EventGeneration.TimeWindowHours) * time.Hour
	newsGroups := s.groupNewsByCategoryWithTimeWindow(newsList, timeWindow, aiConfig.EventGeneration.MinNewsCount)

	generatedCount := 0
	for category, categoryNews := range newsGroups {
		log.Printf("处理分类: %s, 新闻数量: %d", category, len(categoryNews))

		// 为每个分组生成事件
		eventMapping, err := s.generateEventFromNewsGroupWithAI(categoryNews, aiConfig)
		if err != nil {
			log.Printf("为分类 %s 生成事件失败: %v", category, err)
			continue
		}

		if eventMapping == nil {
			log.Printf("分类 %s 不需要生成事件", category)
			continue
		}

		// 创建事件并关联新闻
		if err := s.createEventAndLinkNews(eventMapping); err != nil {
			log.Printf("创建事件并关联新闻失败: %v", err)
			continue
		}

		generatedCount++
	}

	log.Printf("事件生成完成！成功生成 %d 个事件", generatedCount)
	return nil
}

// SeedCompleteData 完整的数据种子化（包含事件生成）
func (s *SeedService) SeedCompleteData() error {
	log.Println("开始完整的数据种子化...")

	// 确保事件生成功能启用
	s.enableEventGeneration = true

	// 1. 导入基础数据（包含自动事件生成）
	if err := s.SeedAllData(); err != nil {
		return fmt.Errorf("基础数据导入失败: %w", err)
	}

	// 2. 创建默认数据
	if err := s.SeedDefaultData(); err != nil {
		return fmt.Errorf("默认数据创建失败: %w", err)
	}

	// 3. 创建RSS源
	if err := s.SeedRSSources(); err != nil {
		return fmt.Errorf("RSS源创建失败: %w", err)
	}

	log.Println("完整的数据种子化完成！")
	return nil
}

// SeedWithEventGeneration 向后兼容的方法名
func (s *SeedService) SeedWithEventGeneration() error {
	return s.SeedCompleteData()
}

/*
使用示例：

// 1. 基本使用（默认配置）
seedService := NewSeedService()
err := seedService.SeedAllData() // 导入新闻 + 自动生成事件

// 2. 设置AI配置
seedService := NewSeedService()
seedService.SetAIAPIKey("your-openai-api-key")
seedService.SetAIModel("gpt-4")
seedService.SetAIProvider("openai")
err := seedService.SeedAllData()

// 3. 自定义AI配置
aiConfig := &AIServiceConfig{
    Provider:    "openai",
    APIKey:      "your-api-key",
    APIEndpoint: "https://api.openai.com/v1/chat/completions",
    Model:       "gpt-3.5-turbo",
    MaxTokens:   2000,
    Timeout:     30,
    Enabled:     true,
}
aiConfig.EventGeneration.Enabled = true
aiConfig.EventGeneration.ConfidenceThreshold = 0.7
aiConfig.EventGeneration.MinNewsCount = 3
aiConfig.EventGeneration.TimeWindowHours = 48
aiConfig.EventGeneration.MaxNewsLimit = 100

seedService := NewSeedServiceWithAIConfig(aiConfig)
err := seedService.SeedCompleteData()

// 4. 禁用事件生成
seedService := NewSeedServiceWithConfig(false)
err := seedService.SeedNewsFromJSON("data/news.json")

// 5. 仅生成事件（不导入新闻）
seedService := NewSeedService()
seedService.SetAIAPIKey("your-api-key")
err := seedService.GenerateEventsFromNewsWithDefaults()

配置说明：
- Provider: AI服务提供商 ("openai", "baidu", "custom")
- APIKey: API密钥，必须设置有效密钥才能使用真实AI API
- APIEndpoint: API端点地址
- Model: 使用的AI模型
- MaxTokens: 最大token数量
- Timeout: 请求超时时间
- EventGeneration.Enabled: 是否启用事件生成
- EventGeneration.ConfidenceThreshold: 置信度阈值（0.0-1.0）
- EventGeneration.MinNewsCount: 生成事件的最小新闻数量
- EventGeneration.TimeWindowHours: 时间窗口（小时）
- EventGeneration.MaxNewsLimit: 单次处理的最大新闻数量

注意事项：
1. 如果未设置有效的API密钥，系统将使用模拟AI分析
2. 置信度阈值越高，生成的事件越严格
3. 时间窗口决定了多长时间内的新闻会被认为是相关的
4. 所有配置都可以在运行时动态修改
*/
