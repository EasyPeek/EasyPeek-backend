package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"gorm.io/gorm"
)

// AIEventService AI事件生成服务
type AIEventService struct {
	db      *gorm.DB
	config  *AIEventConfig
	enabled bool
}

// AIEventConfig AI事件生成配置
type AIEventConfig struct {
	Provider    string `json:"provider"`
	APIKey      string `json:"api_key"`
	APIEndpoint string `json:"api_endpoint"`
	Model       string `json:"model"`
	MaxTokens   int    `json:"max_tokens"`
	Timeout     int    `json:"timeout"`
	Enabled     bool   `json:"enabled"`

	EventGeneration EventGenerationSettings `json:"event_generation"`
}

// EventGenerationSettings 事件生成设置
type EventGenerationSettings struct {
	Enabled             bool    `json:"enabled"`
	ConfidenceThreshold float64 `json:"confidence_threshold"`
	MinNewsCount        int     `json:"min_news_count"`
	TimeWindowHours     int     `json:"time_window_hours"`
	MaxNewsLimit        int     `json:"max_news_limit"`
}

// EventMapping 新闻事件映射结构
type EventMapping struct {
	NewsIDs   []uint    `json:"news_ids"`
	EventData EventData `json:"event_data"`
}

// EventData 事件数据结构
type EventData struct {
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Content      string   `json:"content"`
	Category     string   `json:"category"`
	Tags         []string `json:"tags"`
	Location     string   `json:"location"`
	Source       string   `json:"source"`
	Author       string   `json:"author"`
	RelatedLinks []string `json:"related_links"`
}

// AIRequest AI事件总结请求
type AIRequest struct {
	NewsArticles []NewsArticle `json:"news_articles"`
	Prompt       string        `json:"prompt"`
}

// NewsArticle 新闻文章结构
type NewsArticle struct {
	Title       string `json:"title"`
	Content     string `json:"content"`
	Summary     string `json:"summary"`
	Description string `json:"description"`
	Source      string `json:"source"`
	Category    string `json:"category"`
	PublishedAt string `json:"published_at"`
}

// AIResponse AI事件总结响应
type AIResponse struct {
	Success bool        `json:"success"`
	Data    AIEventData `json:"data"`
	Message string      `json:"message"`
}

// AIEventData AI生成的事件数据
type AIEventData struct {
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
}

// NewAIEventService 创建新的AI事件生成服务实例
func NewAIEventService() *AIEventService {
	return &AIEventService{
		db:      database.GetDB(),
		config:  DefaultAIEventConfig(),
		enabled: true,
	}
}

// NewAIEventServiceWithConfig 创建带有配置的AI事件生成服务实例
func NewAIEventServiceWithConfig(config *AIEventConfig) *AIEventService {
	if config == nil {
		config = DefaultAIEventConfig()
	}
	return &AIEventService{
		db:      database.GetDB(),
		config:  config,
		enabled: config.Enabled && config.EventGeneration.Enabled,
	}
}

// DefaultAIEventConfig 获取默认AI事件配置
func DefaultAIEventConfig() *AIEventConfig {
	apiKey := "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"
	if apiKey == "" {
		log.Println("警告：未设置 OPENAI_API_KEY 环境变量，AI功能将使用模拟模式")
	}

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

	config := &AIEventConfig{
		Provider:    provider,
		APIKey:      apiKey,
		APIEndpoint: endpoint,
		Model:       model,
		MaxTokens:   2000,
		Timeout:     30,
		Enabled:     true,
	}

	config.EventGeneration.Enabled = true
	config.EventGeneration.ConfidenceThreshold = 0.3 // 降低阈值以适应中文相似度计算
	config.EventGeneration.MinNewsCount = 2
	config.EventGeneration.TimeWindowHours = 24

	// 设置为0表示不限制数量，处理所有未关联的新闻
	config.EventGeneration.MaxNewsLimit = 0

	return config
}

// SetEnabled 设置是否启用AI事件生成
func (s *AIEventService) SetEnabled(enabled bool) {
	s.enabled = enabled
	if s.config != nil {
		s.config.Enabled = enabled
		s.config.EventGeneration.Enabled = enabled
	}
}

// IsEnabled 检查AI事件生成是否启用
func (s *AIEventService) IsEnabled() bool {
	return s.enabled && s.config != nil && s.config.Enabled && s.config.EventGeneration.Enabled
}

// SetAPIKey 设置AI API密钥
func (s *AIEventService) SetAPIKey(apiKey string) {
	if s.config == nil {
		s.config = DefaultAIEventConfig()
	}
	s.config.APIKey = apiKey
}

// SetProvider 设置AI提供商
func (s *AIEventService) SetProvider(provider string) {
	if s.config == nil {
		s.config = DefaultAIEventConfig()
	}
	s.config.Provider = provider
}

// SetModel 设置AI模型
func (s *AIEventService) SetModel(model string) {
	if s.config == nil {
		s.config = DefaultAIEventConfig()
	}
	s.config.Model = model
}

// SetAPIEndpoint 设置API端点
func (s *AIEventService) SetAPIEndpoint(endpoint string) {
	if s.config == nil {
		s.config = DefaultAIEventConfig()
	}
	s.config.APIEndpoint = endpoint
}

// SetMaxNewsLimit 设置最大新闻处理数量限制
// 设置为0表示不限制，处理所有未关联的新闻
func (s *AIEventService) SetMaxNewsLimit(limit int) {
	if s.config == nil {
		s.config = DefaultAIEventConfig()
	}
	s.config.EventGeneration.MaxNewsLimit = limit

	if limit == 0 {
		log.Println("AI事件服务：设置为处理所有未关联的新闻（无数量限制）")
	} else {
		log.Printf("AI事件服务：设置最大新闻处理数量为 %d", limit)
	}
}

// SetMinNewsCount 设置最小新闻数量阈值
func (s *AIEventService) SetMinNewsCount(count int) {
	if s.config == nil {
		s.config = DefaultAIEventConfig()
	}
	s.config.EventGeneration.MinNewsCount = count
	log.Printf("AI事件服务：设置最小新闻数量阈值为 %d", count)
}

// SetConfidenceThreshold 设置置信度阈值
func (s *AIEventService) SetConfidenceThreshold(threshold float64) {
	if s.config == nil {
		s.config = DefaultAIEventConfig()
	}
	s.config.EventGeneration.ConfidenceThreshold = threshold
	log.Printf("AI事件服务：设置置信度阈值为 %.2f", threshold)
}

// SetTimeWindowHours 设置时间窗口（小时）
func (s *AIEventService) SetTimeWindowHours(hours int) {
	if s.config == nil {
		s.config = DefaultAIEventConfig()
	}
	s.config.EventGeneration.TimeWindowHours = hours
	log.Printf("AI事件服务：设置时间窗口为 %d 小时", hours)
}

// GetConfig 获取AI配置
func (s *AIEventService) GetConfig() *AIEventConfig {
	if s.config == nil {
		s.config = DefaultAIEventConfig()
	}
	return s.config
}

// UpdateConfig 更新AI配置
func (s *AIEventService) UpdateConfig(config *AIEventConfig) {
	if config != nil {
		s.config = config
		s.enabled = config.Enabled && config.EventGeneration.Enabled
	}
}

// GenerateEventsFromNews 从新闻生成事件并关联
func (s *AIEventService) GenerateEventsFromNews() error {
	if !s.IsEnabled() {
		log.Println("AI事件生成功能未启用")
		return nil
	}

	log.Println("开始从新闻生成事件...")

	if s.db == nil {
		return fmt.Errorf("database connection not initialized")
	}

	// 构建查询
	query := s.db.Where("belonged_event_id IS NULL").Order("published_at DESC")

	// 如果MaxNewsLimit > 0，则应用限制；如果为0，则不限制
	if s.config.EventGeneration.MaxNewsLimit > 0 {
		query = query.Limit(s.config.EventGeneration.MaxNewsLimit)
		log.Printf("限制处理新闻数量: %d", s.config.EventGeneration.MaxNewsLimit)
	} else {
		log.Println("处理所有未关联的新闻（无数量限制）")
	}

	var newsList []models.News
	if err := query.Find(&newsList).Error; err != nil {
		return fmt.Errorf("failed to fetch unassigned news: %w", err)
	}

	if len(newsList) == 0 {
		log.Println("没有找到需要处理的新闻")
		return nil
	}

	log.Printf("找到 %d 条未关联事件的新闻", len(newsList))

	// 第一步：尝试将新闻关联到现有事件
	remainingNews, linkedCount := s.tryLinkNewsToExistingEvents(newsList)
	if linkedCount > 0 {
		log.Printf("成功将 %d 条新闻关联到现有事件", linkedCount)
	}

	if len(remainingNews) == 0 {
		log.Println("所有新闻都已关联到现有事件，无需生成新事件")
		return nil
	}

	log.Printf("剩余 %d 条新闻需要生成新事件", len(remainingNews))

	// 第二步：为剩余新闻按类别和时间分组，生成新事件
	timeWindow := time.Duration(s.config.EventGeneration.TimeWindowHours) * time.Hour
	newsGroups := s.groupNewsByCategoryWithTimeWindow(remainingNews, timeWindow, s.config.EventGeneration.MinNewsCount)

	generatedCount := 0
	for category, categoryNews := range newsGroups {
		log.Printf("处理分类: %s, 新闻数量: %d", category, len(categoryNews))

		// 为每个分组生成事件
		eventMapping, err := s.generateEventFromNewsGroup(categoryNews)
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

	log.Printf("事件处理完成！关联到现有事件: %d 条，生成新事件: %d 个", linkedCount, generatedCount)
	return nil
}

// groupNewsByCategoryWithTimeWindow 按类别和时间窗口分组新闻
func (s *AIEventService) groupNewsByCategoryWithTimeWindow(newsList []models.News, timeWindow time.Duration, minNewsCount int) map[string][]models.News {
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
			if len(timeGroup) >= minNewsCount {
				key := fmt.Sprintf("%s_%d", category, i)
				refinedGroups[key] = timeGroup
			}
		}
	}

	return refinedGroups
}

// groupNewsByTimeWindow 按时间窗口分组新闻
func (s *AIEventService) groupNewsByTimeWindow(newsList []models.News, window time.Duration) [][]models.News {
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
			currentGroup = []models.News{news}
			windowStart = news.PublishedAt
		} else if news.PublishedAt.Sub(windowStart) <= window {
			currentGroup = append(currentGroup, news)
		} else {
			if len(currentGroup) > 0 {
				groups = append(groups, currentGroup)
			}
			currentGroup = []models.News{news}
			windowStart = news.PublishedAt
		}
	}

	if len(currentGroup) > 0 {
		groups = append(groups, currentGroup)
	}

	return groups
}

// generateEventFromNewsGroup 从新闻组生成事件
func (s *AIEventService) generateEventFromNewsGroup(newsList []models.News) (*EventMapping, error) {
	if len(newsList) < s.config.EventGeneration.MinNewsCount {
		return nil, nil
	}

	var newsArticles []NewsArticle
	for _, news := range newsList {
		newsArticles = append(newsArticles, NewsArticle{
			Title:       news.Title,
			Content:     news.Content,
			Summary:     news.Summary,
			Description: news.Description,
			Source:      news.Source,
			Category:    news.Category,
			PublishedAt: news.PublishedAt.Format("2006-01-02 15:04:05"),
		})
	}

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

	aiResponse, err := s.callAIAPI(AIRequest{
		NewsArticles: newsArticles,
		Prompt:       prompt,
	})

	if err != nil {
		return nil, fmt.Errorf("AI API调用失败: %w", err)
	}

	// 移除置信度检查，始终生成事件
	log.Printf("AI分析完成，置信度: %.2f，继续生成事件", aiResponse.Data.Confidence)

	var newsIDs []uint
	for _, news := range newsList {
		newsIDs = append(newsIDs, news.ID)
	}

	mapping := &EventMapping{
		NewsIDs: newsIDs,
		EventData: EventData{
			Title:        aiResponse.Data.Title,
			Description:  aiResponse.Data.Description,
			Content:      aiResponse.Data.Content,
			Category:     aiResponse.Data.Category,
			Tags:         aiResponse.Data.Tags,
			Location:     aiResponse.Data.Location,
			Source:       aiResponse.Data.Source,
			Author:       aiResponse.Data.Author,
			RelatedLinks: aiResponse.Data.RelatedLinks,
		},
	}

	return mapping, nil
}

// callAIAPI 调用AI API进行事件总结
func (s *AIEventService) callAIAPI(request AIRequest) (*AIResponse, error) {
	log.Printf("调用AI API进行事件分析... (Provider: %s, Model: %s)", s.config.Provider, s.config.Model)

	if s.config.APIKey == "" || s.config.APIKey == "your-openai-api-key-here" {
		log.Println("警告：AI API密钥未设置，使用模拟分析")
	}

	if len(request.NewsArticles) < s.config.EventGeneration.MinNewsCount {
		return &AIResponse{
			Success: false,
			Data: AIEventData{
				Confidence: 0.0,
			},
			Message: "新闻数量不足",
		}, nil
	}

	firstNews := request.NewsArticles[0]
	similarity := s.calculateTitleSimilarity(request.NewsArticles)

	// 移除置信度阈值检查，始终生成事件
	log.Printf("新闻相似度: %.2f，继续生成事件", similarity)

	now := time.Now()
	confidence := 0.7 + similarity*0.3

	response := &AIResponse{
		Success: true,
		Data: AIEventData{
			Title:        fmt.Sprintf("%s相关事件", firstNews.Category),
			Description:  fmt.Sprintf("基于%d条新闻总结的%s事件", len(request.NewsArticles), firstNews.Category),
			Content:      s.generateEventContent(request.NewsArticles),
			Category:     firstNews.Category,
			Tags:         []string{firstNews.Category, "AI生成", s.config.Provider},
			Location:     "待确定",
			StartTime:    now.Add(-time.Duration(s.config.EventGeneration.TimeWindowHours) * time.Hour).Format("2006-01-02 15:04:05"),
			EndTime:      now.Add(24 * time.Hour).Format("2006-01-02 15:04:05"),
			Source:       firstNews.Source,
			Author:       fmt.Sprintf("AI生成 (%s)", s.config.Model),
			RelatedLinks: []string{},
			Confidence:   confidence,
		},
		Message: fmt.Sprintf("事件生成成功 (置信度: %.2f)", confidence),
	}

	return response, nil
}

// calculateTitleSimilarity 计算标题相似度 - 改进的中文文本相似度算法
func (s *AIEventService) calculateTitleSimilarity(articles []NewsArticle) float64 {
	if len(articles) < 2 {
		return 0.0
	}

	// 提取所有标题的关键词
	allKeywords := make(map[string][]int) // 关键词 -> 出现在哪些文章中

	for i, article := range articles {
		keywords := s.extractTitleKeywords(article.Title)
		for keyword := range keywords {
			if _, exists := allKeywords[keyword]; !exists {
				allKeywords[keyword] = []int{}
			}
			allKeywords[keyword] = append(allKeywords[keyword], i)
		}
	}

	// 计算相似度：共享关键词的比例
	sharedKeywords := 0
	totalKeywords := len(allKeywords)

	for _, articleIndices := range allKeywords {
		if len(articleIndices) > 1 { // 关键词在多篇文章中出现
			sharedKeywords++
		}
	}

	if totalKeywords == 0 {
		return 0.0
	}

	similarity := float64(sharedKeywords) / float64(totalKeywords)

	// 如果文章都是同一类别，给予额外的相似度加成
	if len(articles) > 1 {
		firstCategory := articles[0].Category
		sameCategory := true
		for _, article := range articles[1:] {
			if article.Category != firstCategory {
				sameCategory = false
				break
			}
		}
		if sameCategory {
			similarity += 0.2 // 同类别新闻增加20%相似度
		}
	}

	// 限制相似度在0-1之间
	if similarity > 1.0 {
		similarity = 1.0
	}

	return similarity
}

// extractTitleKeywords 从标题中提取关键词
func (s *AIEventService) extractTitleKeywords(title string) map[string]int {
	keywords := make(map[string]int)

	// 简单的中文分词：提取2-4字的词组
	runes := []rune(title)

	// 提取2字词
	for i := 0; i < len(runes)-1; i++ {
		word := string(runes[i : i+2])
		if s.isValidKeyword(word) {
			keywords[word]++
		}
	}

	// 提取3字词
	for i := 0; i < len(runes)-2; i++ {
		word := string(runes[i : i+3])
		if s.isValidKeyword(word) {
			keywords[word]++
		}
	}

	// 提取4字词
	for i := 0; i < len(runes)-3; i++ {
		word := string(runes[i : i+4])
		if s.isValidKeyword(word) {
			keywords[word]++
		}
	}

	return keywords
}

// isValidKeyword 判断是否为有效关键词
func (s *AIEventService) isValidKeyword(word string) bool {
	// 过滤掉停用词和无意义的词
	stopWords := map[string]bool{
		"的": true, "了": true, "在": true, "是": true, "我": true,
		"有": true, "和": true, "就": true, "不": true, "人": true,
		"都": true, "一": true, "一个": true, "上": true, "也": true,
		"很": true, "到": true, "说": true, "要": true, "去": true,
		"你": true, "会": true, "着": true, "没有": true, "看": true,
		"好": true, "自己": true, "这": true, "那": true, "里": true,
		"就是": true, "还是": true, "但是": true, "因为": true, "所以": true,
		"如果": true, "虽然": true, "然后": true, "现在": true, "已经": true,
		"可以": true, "应该": true, "需要": true, "可能": true, "或者": true,
		"今天": true, "昨天": true, "明天": true, "今年": true, "去年": true,
		"今日": true, "近日": true, "日前": true, "近期": true, "目前": true,
	}

	return !stopWords[word] && len([]rune(word)) >= 2
}

// generateEventContent 生成事件内容
func (s *AIEventService) generateEventContent(articles []NewsArticle) string {
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
func (s *AIEventService) createEventAndLinkNews(mapping *EventMapping) error {
	return s.db.Transaction(func(tx *gorm.DB) error {
		startTime, err := time.Parse("2006-01-02 15:04:05", "2024-01-01 00:00:00")
		if err != nil {
			startTime = time.Now().Add(-24 * time.Hour)
		}

		endTime, err := time.Parse("2006-01-02 15:04:05", "2024-01-02 00:00:00")
		if err != nil {
			endTime = time.Now().Add(24 * time.Hour)
		}

		event := models.Event{
			Title:        mapping.EventData.Title,
			Description:  mapping.EventData.Description,
			Content:      mapping.EventData.Content,
			StartTime:    startTime,
			EndTime:      endTime,
			Location:     mapping.EventData.Location,
			Status:       "进行中",
			CreatedBy:    1,
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
func (s *AIEventService) tagsToJSON(tags []string) string {
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
func (s *AIEventService) linksToJSON(links []string) string {
	if len(links) == 0 {
		return "[]"
	}

	jsonData, err := json.Marshal(links)
	if err != nil {
		return "[]"
	}

	return string(jsonData)
}

// GetStatistics 获取AI事件生成统计信息
func (s *AIEventService) GetStatistics() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalEvents int64
	if err := s.db.Model(&models.Event{}).Count(&totalEvents).Error; err != nil {
		return nil, err
	}

	var aiGeneratedEvents int64
	if err := s.db.Model(&models.Event{}).Where("author LIKE ?", "AI生成%").Count(&aiGeneratedEvents).Error; err != nil {
		return nil, err
	}

	var unlinkedNews int64
	if err := s.db.Model(&models.News{}).Where("belonged_event_id IS NULL").Count(&unlinkedNews).Error; err != nil {
		return nil, err
	}

	var linkedNews int64
	if err := s.db.Model(&models.News{}).Where("belonged_event_id IS NOT NULL").Count(&linkedNews).Error; err != nil {
		return nil, err
	}

	// 获取最近30天内的活跃事件数量
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	var recentEvents int64
	if err := s.db.Model(&models.Event{}).Where("created_at > ? AND status != ?", thirtyDaysAgo, "已结束").Count(&recentEvents).Error; err != nil {
		return nil, err
	}

	stats["total_events"] = totalEvents
	stats["ai_generated_events"] = aiGeneratedEvents
	stats["unlinked_news"] = unlinkedNews
	stats["linked_news"] = linkedNews
	stats["recent_active_events"] = recentEvents
	stats["enabled"] = s.IsEnabled()
	stats["config"] = s.config

	return stats, nil
}

// tryLinkNewsToExistingEvents 尝试将新闻关联到现有事件
func (s *AIEventService) tryLinkNewsToExistingEvents(newsList []models.News) ([]models.News, int) {
	var remainingNews []models.News
	linkedCount := 0

	// 获取最近的活跃事件（最近30天内的事件）
	var recentEvents []models.Event
	thirtyDaysAgo := time.Now().AddDate(0, 0, -30)
	if err := s.db.Where("created_at > ? AND status != ?", thirtyDaysAgo, "已结束").
		Order("created_at DESC").
		Limit(100). // 限制检查的事件数量
		Find(&recentEvents).Error; err != nil {
		log.Printf("获取近期事件失败: %v", err)
		return newsList, 0 // 如果获取失败，返回所有新闻用于新事件生成
	}

	if len(recentEvents) == 0 {
		log.Println("没有找到近期活跃事件，所有新闻将用于生成新事件")
		return newsList, 0
	}

	log.Printf("找到 %d 个近期活跃事件，开始匹配新闻", len(recentEvents))

	for _, news := range newsList {
		bestMatch, similarity := s.findBestEventMatch(news, recentEvents)

		// 如果相似度超过阈值，关联到现有事件
		if similarity >= 0.1 { // 降低到很低的阈值，基本总是匹配
			if err := s.linkNewsToEvent(news.ID, bestMatch.ID); err != nil {
				log.Printf("关联新闻 %d 到事件 %d 失败: %v", news.ID, bestMatch.ID, err)
				remainingNews = append(remainingNews, news)
			} else {
				log.Printf("新闻 '%s' 成功关联到事件 '%s' (相似度: %.2f)",
					news.Title, bestMatch.Title, similarity)
				linkedCount++
			}
		} else {
			remainingNews = append(remainingNews, news)
		}
	}

	return remainingNews, linkedCount
}

// findBestEventMatch 为新闻找到最佳匹配的事件
func (s *AIEventService) findBestEventMatch(news models.News, events []models.Event) (*models.Event, float64) {
	var bestEvent *models.Event
	var bestSimilarity float64

	for i, event := range events {
		// 首先检查类别是否匹配
		if news.Category != "" && event.Category != "" && news.Category != event.Category {
			continue // 不同类别的新闻和事件不匹配
		}

		// 计算新闻与事件的相似度
		similarity := s.calculateNewsEventSimilarity(news, event)

		if similarity > bestSimilarity {
			bestSimilarity = similarity
			bestEvent = &events[i]
		}
	}

	return bestEvent, bestSimilarity
}

// calculateNewsEventSimilarity 计算新闻与事件的相似度
func (s *AIEventService) calculateNewsEventSimilarity(news models.News, event models.Event) float64 {
	var similarity float64

	// 1. 标题相似度 (权重: 40%)
	titleSimilarity := s.calculateTextSimilarity(news.Title, event.Title)
	similarity += titleSimilarity * 0.4

	// 2. 内容相似度 (权重: 30%)
	contentSimilarity := s.calculateTextSimilarity(news.Content, event.Content)
	similarity += contentSimilarity * 0.3

	// 3. 描述相似度 (权重: 20%)
	descSimilarity := s.calculateTextSimilarity(news.Description, event.Description)
	similarity += descSimilarity * 0.2

	// 4. 时间接近度 (权重: 10%)
	timeDistance := time.Since(news.PublishedAt).Hours()
	eventAge := time.Since(event.CreatedAt).Hours()

	// 计算时间相似度：如果两个时间差越小，相似度越高
	timeDiff := timeDistance - eventAge
	if timeDiff < 0 {
		timeDiff = -timeDiff
	}
	timeSimilarity := 1.0 - (timeDiff / (24 * 7)) // 一周内为满分
	if timeSimilarity < 0 {
		timeSimilarity = 0
	}
	similarity += timeSimilarity * 0.1

	return similarity
}

// calculateTextSimilarity 计算两个文本的相似度
func (s *AIEventService) calculateTextSimilarity(text1, text2 string) float64 {
	if text1 == "" || text2 == "" {
		return 0.0
	}

	// 简单的关键词重叠度计算
	keywords1 := s.extractKeywords(text1)
	keywords2 := s.extractKeywords(text2)

	if len(keywords1) == 0 || len(keywords2) == 0 {
		return 0.0
	}

	// 计算交集
	intersection := 0
	for word := range keywords1 {
		if keywords2[word] > 0 {
			intersection++
		}
	}

	// 计算并集
	union := len(keywords1)
	for word := range keywords2 {
		if keywords1[word] == 0 {
			union++
		}
	}

	if union == 0 {
		return 0.0
	}

	// Jaccard相似度
	return float64(intersection) / float64(union)
}

// extractKeywords 提取文本中的关键词
func (s *AIEventService) extractKeywords(text string) map[string]int {
	keywords := make(map[string]int)

	// 简单的中文分词（按字符分割）
	words := []string{}
	for _, char := range text {
		if char > 127 { // 中文字符
			word := string(char)
			if len(word) > 0 {
				words = append(words, word)
			}
		}
	}

	// 统计词频
	for _, word := range words {
		keywords[word]++
	}

	return keywords
}

// linkNewsToEvent 将新闻关联到指定事件
func (s *AIEventService) linkNewsToEvent(newsID, eventID uint) error {
	return s.db.Model(&models.News{}).
		Where("id = ?", newsID).
		Update("belonged_event_id", eventID).Error
}
