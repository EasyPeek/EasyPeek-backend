package services

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
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
	// 默认使用OpenAI兼容的API（如通义千问、文心一言等）
	provider := NewOpenAICompatibleProvider()
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
	apiKey   string
	baseURL  string
	model    string
	siteURL  string
	siteName string
}

// NewOpenAICompatibleProvider 创建OpenAI兼容的提供商
func NewOpenAICompatibleProvider() *OpenAICompatibleProvider {
	// 从配置文件读取
	cfg := config.AppConfig

	// 默认配置 - 修改为OpenRouter的默认配置
	provider := &OpenAICompatibleProvider{
		apiKey:   "your-api-key",
		baseURL:  "https://openrouter.ai/api/v1",
		model:    "google/gemini-2.0-flash-001",
		siteURL:  "http://localhost:5173/",
		siteName: "EasyPeek",
	}

	// 如果配置存在，则使用配置值
	if cfg != nil {
		// 直接使用配置文件中的API Key，不再处理环境变量引用
		if cfg.AI.APIKey != "" {
			provider.apiKey = cfg.AI.APIKey
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
	}

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

	// 解析JSON响应
	var result struct {
		Analysis      string                `json:"analysis"`
		ImpactLevel   string                `json:"impact_level"`
		ImpactScore   float64               `json:"impact_score"`
		ImpactScope   string                `json:"impact_scope"`
		RelatedTopics []string              `json:"related_topics"`
		AnalysisSteps []models.AnalysisStep `json:"analysis_steps"`
	}

	if err := json.Unmarshal([]byte(response), &result); err != nil {
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

	var predictions []models.TrendPrediction
	if err := json.Unmarshal([]byte(response), &predictions); err != nil {
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
		"temperature": 0.7,
		"max_tokens":  2000,
	}

	jsonData, err := json.Marshal(requestBody)
	if err != nil {
		return "", err
	}

	req, err := http.NewRequest("POST", p.baseURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return "", err
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	// OpenRouter特有的请求头
	if p.siteURL != "" {
		req.Header.Set("HTTP-Referer", p.siteURL)
	}
	if p.siteName != "" {
		req.Header.Set("X-Title", p.siteName)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
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
			Code    int    `json:"code"` // 修改：从string改为int
		} `json:"error"`
		Usage struct {
			PromptTokens     int `json:"prompt_tokens"`
			CompletionTokens int `json:"completion_tokens"`
			TotalTokens      int `json:"total_tokens"`
		} `json:"usage"`
	}

	if err := json.Unmarshal(body, &response); err != nil {
		return "", err
	}

	if response.Error.Message != "" {
		return "", fmt.Errorf("API error [%s]: %s (code: %d)", response.Error.Type, response.Error.Message, response.Error.Code)
	}

	if len(response.Choices) > 0 {
		return strings.TrimSpace(response.Choices[0].Message.Content), nil
	}

	return "", fmt.Errorf("no response from API")
}
