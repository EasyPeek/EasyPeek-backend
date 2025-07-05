package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strings"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"gorm.io/gorm"
)

// AIProvider AI提供商接口
type AIProvider interface {
	GenerateSummary(content string) (string, error)
	ExtractKeywords(content string) ([]string, error)
	AnalyzeSentiment(content string) (string, float64, error)
	AnalyzeEvent(content string, context string) (*EventAnalysisResult, error)
	PredictTrends(content string, historicalData []models.News) ([]models.TrendPrediction, error)
}

// EventAnalysisResult 事件分析结果
type EventAnalysisResult struct {
	Analysis      string
	ImpactLevel   string
	ImpactScore   float64
	ImpactScope   string
	RelatedTopics []string
	Steps         []models.AnalysisStep
}

// AIService AI服务
type AIService struct {
	db       *gorm.DB
	provider AIProvider
}

// NewAIService 创建AI服务实例
func NewAIService(db *gorm.DB) *AIService {
	provider := NewOpenAICompatibleProvider()
	return &AIService{
		db:       db,
		provider: provider,
	}
}

// NewAIServiceWithConfig 使用指定配置创建AI服务实例
func NewAIServiceWithConfig(db *gorm.DB, cfg *config.Config) *AIService {
	provider := NewOpenAICompatibleProviderWithConfig(cfg)
	return &AIService{
		db:       db,
		provider: provider,
	}
}

// AnalyzeNews 分析新闻
func (s *AIService) AnalyzeNews(newsID uint, options models.AIAnalysisRequest) (*models.AIAnalysis, error) {
	startTime := time.Now()

	// 获取新闻数据
	var news models.News
	if err := s.db.First(&news, newsID).Error; err != nil {
		return nil, fmt.Errorf("新闻未找到: %w", err)
	}

	// 检查是否已有分析结果
	var existingAnalysis models.AIAnalysis
	if err := s.db.Where("type = ? AND target_id = ?", models.AIAnalysisTypeNews, newsID).First(&existingAnalysis).Error; err == nil {
		// 如果已有分析且状态为completed，直接返回
		if existingAnalysis.Status == "completed" {
			return &existingAnalysis, nil
		}
	}

	// 创建新的分析记录
	analysis := &models.AIAnalysis{
		Type:         models.AIAnalysisTypeNews,
		TargetID:     newsID,
		Status:       "processing",
		ModelName:    s.provider.(*OpenAICompatibleProvider).model, // 从配置读取模型名称
		ModelVersion: "v1",
	}

	// 保存初始记录
	if err := s.db.Create(analysis).Error; err != nil {
		return nil, fmt.Errorf("创建分析记录失败: %w", err)
	}

	// 准备分析内容
	content := news.Title + "\n\n" + news.Content
	if news.Description != "" {
		content = news.Title + "\n\n" + news.Description + "\n\n" + news.Content
	}

	analysisSteps := []models.AnalysisStep{}
	stepCount := 1

	// 1. 生成摘要
	if options.Options.EnableSummary {
		summary, err := s.provider.GenerateSummary(content)
		if err != nil {
			s.updateAnalysisStatus(analysis.ID, "failed", 0)
			return nil, fmt.Errorf("生成摘要失败: %w", err)
		}
		analysis.Summary = summary
		analysisSteps = append(analysisSteps, models.AnalysisStep{
			Step:        stepCount,
			Title:       "内容摘要提取",
			Description: "使用AI模型对新闻内容进行摘要提取",
			Result:      "已生成简洁的新闻摘要",
			Confidence:  0.95,
		})
		stepCount++
	}

	// 2. 提取关键词
	if options.Options.EnableKeywords {
		keywords, err := s.provider.ExtractKeywords(content)
		if err != nil {
			s.updateAnalysisStatus(analysis.ID, "failed", 0)
			return nil, fmt.Errorf("提取关键词失败: %w", err)
		}
		keywordsJSON, _ := json.Marshal(keywords)
		analysis.Keywords = string(keywordsJSON)
		analysisSteps = append(analysisSteps, models.AnalysisStep{
			Step:        stepCount,
			Title:       "关键词提取",
			Description: "从新闻内容中提取核心关键词",
			Result:      fmt.Sprintf("提取到%d个关键词", len(keywords)),
			Confidence:  0.92,
		})
		stepCount++
	}

	// 3. 情感分析
	if options.Options.EnableSentiment {
		sentiment, score, err := s.provider.AnalyzeSentiment(content)
		if err != nil {
			s.updateAnalysisStatus(analysis.ID, "failed", 0)
			return nil, fmt.Errorf("情感分析失败: %w", err)
		}
		analysis.Sentiment = sentiment
		analysis.SentimentScore = score
		analysisSteps = append(analysisSteps, models.AnalysisStep{
			Step:        stepCount,
			Title:       "情感倾向分析",
			Description: "分析新闻内容的情感倾向",
			Result:      fmt.Sprintf("情感倾向: %s (得分: %.2f)", sentiment, score),
			Confidence:  0.88,
		})
		stepCount++
	}

	// 4. 事件分析和影响力评估
	if options.Options.EnableImpact {
		// 获取相关历史新闻作为上下文
		var relatedNews []models.News
		s.db.Where("category = ? AND id != ?", news.Category, news.ID).
			Order("published_at DESC").
			Limit(10).
			Find(&relatedNews)

		context := s.buildContext(relatedNews)
		eventResult, err := s.provider.AnalyzeEvent(content, context)
		if err != nil {
			s.updateAnalysisStatus(analysis.ID, "failed", 0)
			return nil, fmt.Errorf("事件分析失败: %w", err)
		}

		analysis.EventAnalysis = eventResult.Analysis
		analysis.ImpactLevel = eventResult.ImpactLevel
		analysis.ImpactScore = eventResult.ImpactScore
		analysis.ImpactScope = eventResult.ImpactScope

		topicsJSON, _ := json.Marshal(eventResult.RelatedTopics)
		analysis.RelatedTopics = string(topicsJSON)

		analysisSteps = append(analysisSteps, models.AnalysisStep{
			Step:        stepCount,
			Title:       "事件影响力分析",
			Description: "评估事件的潜在影响力和范围",
			Result:      fmt.Sprintf("影响级别: %s, 影响力分数: %.2f", eventResult.ImpactLevel, eventResult.ImpactScore),
			Confidence:  0.85,
		})
		stepCount++

		// 添加AI生成的分析步骤
		analysisSteps = append(analysisSteps, eventResult.Steps...)
	}

	// 5. 趋势预测
	if options.Options.EnableTrends {
		// 获取历史数据
		var historicalNews []models.News
		s.db.Where("category = ?", news.Category).
			Order("published_at DESC").
			Limit(20).
			Find(&historicalNews)

		predictions, err := s.provider.PredictTrends(content, historicalNews)
		if err != nil {
			s.updateAnalysisStatus(analysis.ID, "failed", 0)
			return nil, fmt.Errorf("趋势预测失败: %w", err)
		}

		analysis.TrendPredictions = predictions
		analysisSteps = append(analysisSteps, models.AnalysisStep{
			Step:        stepCount,
			Title:       "未来趋势预测",
			Description: "基于历史数据和当前事件预测未来发展趋势",
			Result:      fmt.Sprintf("生成了%d个时间维度的趋势预测", len(predictions)),
			Confidence:  0.78,
		})
	}

	// 更新分析步骤
	if options.Options.ShowAnalysisSteps {
		analysis.AnalysisSteps = analysisSteps
	}

	// 计算整体置信度
	if len(analysisSteps) > 0 {
		totalConfidence := 0.0
		for _, step := range analysisSteps {
			totalConfidence += step.Confidence
		}
		analysis.Confidence = totalConfidence / float64(len(analysisSteps))
	}

	// 更新处理时间和状态
	processingTime := int(time.Since(startTime).Seconds())
	analysis.ProcessingTime = processingTime
	analysis.Status = "completed"

	// 保存完整的分析结果
	if err := s.db.Save(analysis).Error; err != nil {
		return nil, fmt.Errorf("保存分析结果失败: %w", err)
	}

	// 更新新闻的摘要字段
	if analysis.Summary != "" {
		s.db.Model(&news).Update("summary", analysis.Summary)
	}

	return analysis, nil
}

// AnalyzeEvent 分析事件
func (s *AIService) AnalyzeEvent(eventID uint, options models.AIAnalysisRequest) (*models.AIAnalysis, error) {
	startTime := time.Now()

	// 获取事件数据
	var event models.Event
	if err := s.db.First(&event, eventID).Error; err != nil {
		return nil, fmt.Errorf("事件未找到: %w", err)
	}

	// 检查是否已有分析结果
	var existingAnalysis models.AIAnalysis
	if err := s.db.Where("type = ? AND target_id = ?", models.AIAnalysisTypeEvent, eventID).First(&existingAnalysis).Error; err == nil {
		// 如果已有分析且状态为completed，直接返回
		if existingAnalysis.Status == "completed" {
			return &existingAnalysis, nil
		}
	}

	// 获取关联的新闻
	var relatedNews []models.News
	s.db.Where("belonged_event_id = ?", eventID).Find(&relatedNews)

	// 构建分析内容
	content := s.buildEventContent(event, relatedNews)

	// 创建分析记录
	analysis := &models.AIAnalysis{
		Type:         models.AIAnalysisTypeEvent,
		TargetID:     eventID,
		Status:       "processing",
		ModelName:    s.provider.(*OpenAICompatibleProvider).model, // 从配置读取模型名称
		ModelVersion: "v1",
	}

	if err := s.db.Create(analysis).Error; err != nil {
		return nil, fmt.Errorf("创建分析记录失败: %w", err)
	}

	// 执行分析（类似新闻分析流程）
	analysisSteps := []models.AnalysisStep{}
	stepCount := 1

	// 1. 生成摘要
	if options.Options.EnableSummary {
		summary, err := s.provider.GenerateSummary(content)
		if err != nil {
			s.updateAnalysisStatus(analysis.ID, "failed", 0)
			return nil, fmt.Errorf("生成摘要失败: %w", err)
		}
		analysis.Summary = summary
		analysisSteps = append(analysisSteps, models.AnalysisStep{
			Step:        stepCount,
			Title:       "事件摘要提取",
			Description: "使用AI模型对事件内容进行摘要提取",
			Result:      "已生成事件摘要",
			Confidence:  0.95,
		})
		stepCount++
	}

	// 2. 提取关键词
	if options.Options.EnableKeywords {
		keywords, err := s.provider.ExtractKeywords(content)
		if err != nil {
			s.updateAnalysisStatus(analysis.ID, "failed", 0)
			return nil, fmt.Errorf("提取关键词失败: %w", err)
		}
		keywordsJSON, _ := json.Marshal(keywords)
		analysis.Keywords = string(keywordsJSON)
		analysisSteps = append(analysisSteps, models.AnalysisStep{
			Step:        stepCount,
			Title:       "关键词提取",
			Description: "从事件内容中提取核心关键词",
			Result:      fmt.Sprintf("提取到%d个关键词", len(keywords)),
			Confidence:  0.92,
		})
		stepCount++
	}

	// 3. 情感分析
	if options.Options.EnableSentiment {
		sentiment, score, err := s.provider.AnalyzeSentiment(content)
		if err != nil {
			s.updateAnalysisStatus(analysis.ID, "failed", 0)
			return nil, fmt.Errorf("情感分析失败: %w", err)
		}
		analysis.Sentiment = sentiment
		analysis.SentimentScore = score
		analysisSteps = append(analysisSteps, models.AnalysisStep{
			Step:        stepCount,
			Title:       "情感倾向分析",
			Description: "分析事件的情感倾向",
			Result:      fmt.Sprintf("情感倾向: %s (得分: %.2f)", sentiment, score),
			Confidence:  0.88,
		})
		stepCount++
	}

	// 4. 事件深度分析和影响力评估
	if options.Options.EnableImpact {
		context := s.buildContext(relatedNews)
		eventResult, err := s.provider.AnalyzeEvent(content, context)
		if err != nil {
			s.updateAnalysisStatus(analysis.ID, "failed", 0)
			return nil, fmt.Errorf("事件分析失败: %w", err)
		}

		analysis.EventAnalysis = eventResult.Analysis
		analysis.ImpactLevel = eventResult.ImpactLevel
		analysis.ImpactScore = eventResult.ImpactScore
		analysis.ImpactScope = eventResult.ImpactScope

		topicsJSON, _ := json.Marshal(eventResult.RelatedTopics)
		analysis.RelatedTopics = string(topicsJSON)

		analysisSteps = append(analysisSteps, models.AnalysisStep{
			Step:        stepCount,
			Title:       "事件影响力分析",
			Description: "评估事件的潜在影响力和范围",
			Result:      fmt.Sprintf("影响级别: %s, 影响力分数: %.2f", eventResult.ImpactLevel, eventResult.ImpactScore),
			Confidence:  0.85,
		})
		stepCount++

		// 添加AI生成的分析步骤
		analysisSteps = append(analysisSteps, eventResult.Steps...)
	}

	// 5. 趋势预测
	if options.Options.EnableTrends {
		// 使用相关新闻作为历史数据
		predictions, err := s.provider.PredictTrends(content, relatedNews)
		if err != nil {
			s.updateAnalysisStatus(analysis.ID, "failed", 0)
			return nil, fmt.Errorf("趋势预测失败: %w", err)
		}

		analysis.TrendPredictions = predictions
		analysisSteps = append(analysisSteps, models.AnalysisStep{
			Step:        stepCount,
			Title:       "未来趋势预测",
			Description: "基于事件和相关新闻预测未来发展趋势",
			Result:      fmt.Sprintf("生成了%d个时间维度的趋势预测", len(predictions)),
			Confidence:  0.78,
		})
	}

	// 更新分析步骤
	if options.Options.ShowAnalysisSteps {
		analysis.AnalysisSteps = analysisSteps
	}

	// 计算整体置信度
	if len(analysisSteps) > 0 {
		totalConfidence := 0.0
		for _, step := range analysisSteps {
			totalConfidence += step.Confidence
		}
		analysis.Confidence = totalConfidence / float64(len(analysisSteps))
	}

	// 更新处理时间和状态
	analysis.Status = "completed"
	analysis.ProcessingTime = int(time.Since(startTime).Seconds())

	if err := s.db.Save(analysis).Error; err != nil {
		return nil, fmt.Errorf("保存分析结果失败: %w", err)
	}

	return analysis, nil
}

// GetAnalysis 获取分析结果
func (s *AIService) GetAnalysis(analysisType models.AIAnalysisType, targetID uint) (*models.AIAnalysis, error) {
	var analysis models.AIAnalysis

	// 优先获取completed状态的最新记录
	err := s.db.Where("type = ? AND target_id = ? AND status = ?", analysisType, targetID, "completed").
		Order("created_at DESC").
		First(&analysis).Error

	if err != nil {
		// 如果没有completed的记录，获取最新的记录（可能是processing或failed）
		err = s.db.Where("type = ? AND target_id = ?", analysisType, targetID).
			Order("created_at DESC").
			First(&analysis).Error
		if err != nil {
			return nil, err
		}
	}

	return &analysis, nil
}

// buildContext 构建上下文信息
func (s *AIService) buildContext(relatedNews []models.News) string {
	var context strings.Builder
	context.WriteString("相关历史新闻:\n")
	for i, news := range relatedNews {
		context.WriteString(fmt.Sprintf("%d. %s (%s)\n", i+1, news.Title, news.PublishedAt.Format("2006-01-02")))
		if news.Summary != "" {
			context.WriteString(fmt.Sprintf("   摘要: %s\n", news.Summary))
		}
	}
	return context.String()
}

// buildEventContent 构建事件内容
func (s *AIService) buildEventContent(event models.Event, relatedNews []models.News) string {
	var content strings.Builder
	content.WriteString(fmt.Sprintf("事件标题: %s\n", event.Title))
	content.WriteString(fmt.Sprintf("事件描述: %s\n", event.Description))
	if event.Content != "" {
		content.WriteString(fmt.Sprintf("\n事件详情:\n%s\n", event.Content))
	}

	if len(relatedNews) > 0 {
		content.WriteString("\n相关新闻:\n")
		for i, news := range relatedNews {
			content.WriteString(fmt.Sprintf("%d. %s\n", i+1, news.Title))
			if news.Summary != "" {
				content.WriteString(fmt.Sprintf("   %s\n", news.Summary))
			}
		}
	}

	return content.String()
}

// GetProvider 获取AI提供商实例（用于测试）
func (s *AIService) GetProvider() AIProvider {
	return s.provider
}

// updateAnalysisStatus 更新分析状态
func (s *AIService) updateAnalysisStatus(analysisID uint, status string, processingTime int) {
	updates := map[string]interface{}{
		"status": status,
	}
	if processingTime > 0 {
		updates["processing_time"] = processingTime
	}
	s.db.Model(&models.AIAnalysis{}).Where("id = ?", analysisID).Updates(updates)
}

// OpenAICompatibleProvider OpenAI兼容API提供商
type OpenAICompatibleProvider struct {
	apiKey      string
	baseURL     string
	model       string
	siteURL     string
	siteName    string
	maxTokens   int
	temperature float64
	timeout     int
}

// NewOpenAICompatibleProvider 创建OpenAI兼容的提供商（使用全局配置）
func NewOpenAICompatibleProvider() *OpenAICompatibleProvider {
	return NewOpenAICompatibleProviderWithConfig(config.AppConfig)
}

// NewOpenAICompatibleProviderWithConfig 使用指定配置创建OpenAI兼容的提供商
func NewOpenAICompatibleProviderWithConfig(cfg *config.Config) *OpenAICompatibleProvider {
	// 默认配置 - 修改为OpenRouter的默认配置
	provider := &OpenAICompatibleProvider{
		apiKey:      "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",
		baseURL:     "https://openrouter.ai/api/v1",
		model:       "google/gemini-2.0-flash-001", // 使用稳定的thinking版本
		siteURL:     "http://localhost:5173/",
		siteName:    "EasyPeek",
		maxTokens:   4000,
		temperature: 0.7,
		timeout:     30,
	}

	// 如果配置存在，则使用配置值
	if cfg != nil {
		log.Printf("🔧 AI配置加载: Provider=%s, Model=%s", cfg.AI.Provider, cfg.AI.Model)
		// 直接使用配置文件中的API Key
		if cfg.AI.APIKey != "" {
			provider.apiKey = cfg.AI.APIKey
			// 安全地显示API Key前缀用于调试
			keyPrefix := cfg.AI.APIKey
			if len(keyPrefix) > 20 {
				keyPrefix = keyPrefix[:20] + "..."
			}
			log.Printf("🔑 API Key已从配置文件加载: %s (长度: %d)", keyPrefix, len(cfg.AI.APIKey))
		} else {
			log.Printf("⚠️ 警告: 配置文件中API Key为空: %q", cfg.AI.APIKey)
		}
		if cfg.AI.BaseURL != "" {
			provider.baseURL = cfg.AI.BaseURL
		}
		if cfg.AI.Model != "" {
			provider.model = cfg.AI.Model
		}
		if cfg.AI.SiteURL != "" {
			provider.siteURL = cfg.AI.SiteURL
		}
		if cfg.AI.SiteName != "" {
			provider.siteName = cfg.AI.SiteName
		}
		if cfg.AI.MaxTokens > 0 {
			provider.maxTokens = cfg.AI.MaxTokens
		}
		if cfg.AI.Temperature > 0 {
			provider.temperature = cfg.AI.Temperature
		}
		if cfg.AI.Timeout > 0 {
			provider.timeout = cfg.AI.Timeout
		}
	} else {
		log.Printf("❌ 错误: 配置文件未加载")
	}

	log.Printf("✅ AI Provider初始化完成 - BaseURL: %s, Model: %s", provider.baseURL, provider.model)
	return provider
}

// GenerateSummary 生成摘要
func (p *OpenAICompatibleProvider) GenerateSummary(content string) (string, error) {
	prompt := fmt.Sprintf(`请为以下新闻内容生成一个简洁的摘要（不超过200字）：

%s

摘要：`, content)

	response, err := p.callAPI(prompt)
	if err != nil {
		return "", err
	}

	return response, nil
}

// ExtractKeywords 提取关键词
func (p *OpenAICompatibleProvider) ExtractKeywords(content string) ([]string, error) {
	prompt := fmt.Sprintf(`请从以下新闻内容中提取5-8个最重要的关键词，用逗号分隔：

%s

关键词：`, content)

	response, err := p.callAPI(prompt)
	if err != nil {
		return nil, err
	}

	// 解析关键词
	keywords := strings.Split(response, ",")
	for i := range keywords {
		keywords[i] = strings.TrimSpace(keywords[i])
	}

	return keywords, nil
}

// AnalyzeSentiment 情感分析
func (p *OpenAICompatibleProvider) AnalyzeSentiment(content string) (string, float64, error) {
	prompt := fmt.Sprintf(`请分析以下新闻内容的情感倾向，返回格式为"情感类型|分数"（情感类型：positive/negative/neutral，分数：0-1）：

%s

分析结果：`, content)

	response, err := p.callAPI(prompt)
	if err != nil {
		return "", 0, err
	}

	// 解析结果
	parts := strings.Split(response, "|")
	if len(parts) != 2 {
		return "neutral", 0.5, nil
	}

	sentiment := strings.TrimSpace(parts[0])
	score := 0.5
	fmt.Sscanf(strings.TrimSpace(parts[1]), "%f", &score)

	return sentiment, score, nil
}

// AnalyzeEvent 分析事件
func (p *OpenAICompatibleProvider) AnalyzeEvent(content string, context string) (*EventAnalysisResult, error) {
	prompt := fmt.Sprintf(`请对以下新闻事件进行深度分析：

新闻内容：
%s

%s

请按以下格式返回分析结果（使用JSON格式）：
{
  "analysis": "事件的详细分析",
  "impact_level": "high/medium/low",
  "impact_score": 0.0-10.0,
  "impact_scope": "影响范围描述",
  "related_topics": ["相关话题1", "相关话题2"],
  "analysis_steps": [
    {
      "step": 1,
      "title": "步骤标题",
      "description": "步骤描述",
      "result": "步骤结果",
      "confidence": 0.0-1.0
    }
  ]
}`, content, context)

	response, err := p.callAPI(prompt)
	if err != nil {
		return nil, err
	}

	// 清理响应，提取JSON部分
	jsonStart := strings.Index(response, "{")
	jsonEnd := strings.LastIndex(response, "}")

	var jsonContent string
	if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
		jsonContent = response[jsonStart : jsonEnd+1]
	} else {
		jsonContent = response
	}

	// 解析JSON响应
	var result struct {
		Analysis      string                `json:"analysis"`
		ImpactLevel   string                `json:"impact_level"`
		ImpactScore   float64               `json:"impact_score"`
		ImpactScope   string                `json:"impact_scope"`
		RelatedTopics []string              `json:"related_topics"`
		AnalysisSteps []models.AnalysisStep `json:"analysis_steps"`
	}

	if err := json.Unmarshal([]byte(jsonContent), &result); err != nil {
		log.Printf("⚠️ JSON解析失败: %v, 原始响应: %s", err, response)
		// 如果JSON解析失败，返回默认值
		return &EventAnalysisResult{
			Analysis:      response,
			ImpactLevel:   "medium",
			ImpactScore:   5.0,
			ImpactScope:   "影响范围待评估",
			RelatedTopics: []string{},
			Steps:         []models.AnalysisStep{},
		}, nil
	}

	return &EventAnalysisResult{
		Analysis:      result.Analysis,
		ImpactLevel:   result.ImpactLevel,
		ImpactScore:   result.ImpactScore,
		ImpactScope:   result.ImpactScope,
		RelatedTopics: result.RelatedTopics,
		Steps:         result.AnalysisSteps,
	}, nil
}

// PredictTrends 预测趋势
func (p *OpenAICompatibleProvider) PredictTrends(content string, historicalData []models.News) ([]models.TrendPrediction, error) {
	// 构建历史数据摘要
	historyContext := "历史数据趋势：\n"
	for i, news := range historicalData {
		if i >= 5 { // 只取最近5条
			break
		}
		historyContext += fmt.Sprintf("- %s: %s\n", news.PublishedAt.Format("2006-01-02"), news.Title)
	}

	prompt := fmt.Sprintf(`基于以下新闻内容和历史数据，预测该事件的未来发展趋势：

当前新闻：
%s

%s

请按以下JSON格式返回预测结果：
[
  {
    "timeframe": "短期（1-7天）",
    "trend": "趋势描述",
    "probability": 0.0-1.0,
    "factors": ["影响因素1", "影响因素2"]
  },
  {
    "timeframe": "中期（1-4周）",
    "trend": "趋势描述",
    "probability": 0.0-1.0,
    "factors": ["影响因素1", "影响因素2"]
  },
  {
    "timeframe": "长期（1-3月）",
    "trend": "趋势描述",
    "probability": 0.0-1.0,
    "factors": ["影响因素1", "影响因素2"]
  }
]`, content, historyContext)

	response, err := p.callAPI(prompt)
	if err != nil {
		return nil, err
	}

	// 清理响应，提取JSON数组部分
	jsonStart := strings.Index(response, "[")
	jsonEnd := strings.LastIndex(response, "]")

	var jsonContent string
	if jsonStart != -1 && jsonEnd != -1 && jsonEnd > jsonStart {
		jsonContent = response[jsonStart : jsonEnd+1]
	} else {
		jsonContent = response
	}

	var predictions []models.TrendPrediction
	if err := json.Unmarshal([]byte(jsonContent), &predictions); err != nil {
		log.Printf("⚠️ 趋势预测JSON解析失败: %v, 原始响应: %s", err, response)
		// 返回默认预测
		return []models.TrendPrediction{
			{
				Timeframe:   "短期（1-7天）",
				Trend:       "事态将持续发展",
				Probability: 0.7,
				Factors:     []string{"舆论关注度", "政策影响"},
			},
		}, nil
	}

	return predictions, nil
}

// callAPI 调用API
func (p *OpenAICompatibleProvider) callAPI(prompt string) (string, error) {
	log.Printf("🤖 AI API 调用开始 - 模型: %s, baseURL: %s", p.model, p.baseURL)

	requestBody := map[string]interface{}{
		"model": p.model,
		"messages": []map[string]string{
			{
				"role":    "system",
				"content": "你是一个专业的新闻分析助手，擅长新闻摘要、关键词提取、事件分析和趋势预测。",
			},
			{
				"role":    "user",
				"content": prompt,
			},
		},
		"temperature": p.temperature,
		"max_tokens":  p.maxTokens,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		log.Printf("❌ JSON序列化失败: %v", err)
		return "", err
	}

	log.Printf("📤 发送请求到: %s", p.baseURL+"/chat/completions")
	req, err := http.NewRequest("POST", p.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		log.Printf("❌ 创建HTTP请求失败: %v", err)
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// 安全地打印API Key的前几位
	keyPreview := "未设置"
	if len(p.apiKey) > 0 {
		if len(p.apiKey) >= 10 {
			keyPreview = p.apiKey[:10] + "..."
		} else {
			keyPreview = p.apiKey[:len(p.apiKey)] + "..."
		}
	}
	log.Printf("🔑 API Key: %s", keyPreview)

	// OpenRouter特有的请求头
	if p.siteURL != "" {
		req.Header.Set("HTTP-Referer", p.siteURL)
		log.Printf("🌐 设置Referer: %s", p.siteURL)
	}
	if p.siteName != "" {
		req.Header.Set("X-Title", p.siteName)
		log.Printf("📝 设置站点名称: %s", p.siteName)
	}

	client := &http.Client{Timeout: time.Duration(p.timeout) * time.Second}
	start := time.Now()
	resp, err := client.Do(req)
	duration := time.Since(start)

	if err != nil {
		log.Printf("❌ HTTP请求失败 (耗时: %v): %v", duration, err)
		return "", err
	}
	defer resp.Body.Close()

	log.Printf("📥 收到响应 - 状态码: %d, 耗时: %v", resp.StatusCode, duration)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		log.Printf("❌ 读取响应体失败: %v", err)
		return "", err
	}

	log.Printf("📄 响应体长度: %d 字节", len(body))

	// 如果状态码不是200，先打印响应体用于调试
	if resp.StatusCode != 200 {
		log.Printf("❌ HTTP错误 %d - 响应内容: %s", resp.StatusCode, string(body))
		return "", fmt.Errorf("HTTP error %d: %s", resp.StatusCode, string(body))
	}

	var response struct {
		Choices []struct {
			Message struct {
				Content string `json:"content"`
			} `json:"message"`
		} `json:"choices"`
		Error struct {
			Message string `json:"message"`
			Type    string `json:"type"`
			Code    int    `json:"code"`
		} `json:"error"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		log.Printf("❌ JSON解析失败: %v - 响应内容: %s", err, string(body))
		return "", err
	}

	if response.Error.Message != "" {
		log.Printf("❌ API返回错误: [%s] %s (code: %d)", response.Error.Type, response.Error.Message, response.Error.Code)
		return "", fmt.Errorf("API error [%s]: %s (code: %d)", response.Error.Type, response.Error.Message, response.Error.Code)
	}

	if len(response.Choices) > 0 {
		content := strings.TrimSpace(response.Choices[0].Message.Content)
		log.Printf("✅ AI分析成功 - 内容长度: %d 字符, Token使用: %d", len(content), response.Usage.TotalTokens)
		return content, nil
	}

	log.Printf("❌ API响应无有效内容 - 响应: %s", string(body))
	return "", fmt.Errorf("no response from API")
}

// BatchAnalyzeUnprocessedNews 批量分析未处理的新闻
func (s *AIService) BatchAnalyzeUnprocessedNews() error {
	log.Printf("[AI BATCH] 开始批量分析未处理的新闻...")

	// 查找没有AI分析结果的新闻
	var newsWithoutAnalysis []models.News
	subQuery := s.db.Table("ai_analyses").
		Select("target_id").
		Where("type = ? AND status = ?", models.AIAnalysisTypeNews, "completed")

	err := s.db.Where("source_type = ? AND id NOT IN (?)", models.NewsTypeRSS, subQuery).
		Order("created_at DESC").
		Limit(50). // 每次处理50条，避免负载过高
		Find(&newsWithoutAnalysis).Error

	if err != nil {
		log.Printf("[AI BATCH ERROR] 查询未分析新闻失败: %v", err)
		return err
	}

	if len(newsWithoutAnalysis) == 0 {
		log.Printf("[AI BATCH] 没有需要分析的新闻")
		return nil
	}

	log.Printf("[AI BATCH] 找到 %d 条需要分析的新闻", len(newsWithoutAnalysis))

	successCount := 0
	for i, news := range newsWithoutAnalysis {
		log.Printf("[AI BATCH] 分析新闻 %d/%d: %s", i+1, len(newsWithoutAnalysis), news.Title)

		analysisReq := models.AIAnalysisRequest{
			Type:     models.AIAnalysisTypeNews,
			TargetID: news.ID,
			Options: struct {
				EnableSummary     bool `json:"enable_summary"`
				EnableKeywords    bool `json:"enable_keywords"`
				EnableSentiment   bool `json:"enable_sentiment"`
				EnableTrends      bool `json:"enable_trends"`
				EnableImpact      bool `json:"enable_impact"`
				ShowAnalysisSteps bool `json:"show_analysis_steps"`
			}{
				EnableSummary:   true,
				EnableKeywords:  true,
				EnableSentiment: true,
				EnableTrends:    false, // 批量处理时关闭趋势分析，提高速度
				EnableImpact:    false, // 批量处理时关闭影响分析，提高速度
			},
		}

		// 同步执行，避免并发过多
		if _, err := s.AnalyzeNews(news.ID, analysisReq); err != nil {
			log.Printf("[AI BATCH WARNING] 分析新闻 %d 失败: %v", news.ID, err)
		} else {
			successCount++
			log.Printf("[AI BATCH] 成功分析新闻 %d", news.ID)
		}

		// 添加延迟避免API限制
		time.Sleep(2 * time.Second)
	}

	log.Printf("[AI BATCH] 批量分析完成: 成功 %d/%d", successCount, len(newsWithoutAnalysis))
	return nil
}

// AnalyzeNewsWithRetry 带重试机制的新闻AI分析
func (s *AIService) AnalyzeNewsWithRetry(newsID uint, maxRetries int) {
	analysisReq := models.AIAnalysisRequest{
		Type:     models.AIAnalysisTypeNews,
		TargetID: newsID,
		Options: struct {
			EnableSummary     bool `json:"enable_summary"`
			EnableKeywords    bool `json:"enable_keywords"`
			EnableSentiment   bool `json:"enable_sentiment"`
			EnableTrends      bool `json:"enable_trends"`
			EnableImpact      bool `json:"enable_impact"`
			ShowAnalysisSteps bool `json:"show_analysis_steps"`
		}{
			EnableSummary:   true,
			EnableKeywords:  true,
			EnableSentiment: true,
			EnableTrends:    false, // RSS实时分析时关闭趋势分析，提高速度
			EnableImpact:    false, // RSS实时分析时关闭影响分析，提高速度
		},
	}

	for attempt := 1; attempt <= maxRetries; attempt++ {
		if _, err := s.AnalyzeNews(newsID, analysisReq); err != nil {
			log.Printf("[AI WARNING] AI分析失败 (尝试 %d/%d): 新闻ID %d, 错误: %v", attempt, maxRetries, newsID, err)
			if attempt < maxRetries {
				// 等待后重试（指数退避）
				waitTime := time.Duration(attempt) * 5 * time.Second
				log.Printf("[AI DEBUG] 等待 %v 后重试AI分析...", waitTime)
				time.Sleep(waitTime)
			}
		} else {
			log.Printf("[AI DEBUG] AI分析成功: 新闻ID %d (尝试 %d/%d)", newsID, attempt, maxRetries)
			return
		}
	}

	log.Printf("[AI ERROR] AI分析最终失败: 新闻ID %d, 已重试 %d 次", newsID, maxRetries)
}
