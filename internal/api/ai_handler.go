package api

import (
	"strconv"

	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/services"
	"github.com/EasyPeek/EasyPeek-backend/internal/utils"
	"github.com/gin-gonic/gin"
)

// AIHandler AI相关的处理器
type AIHandler struct {
	aiService   *services.AIService
	newsService *services.NewsService
}

// NewAIHandler 创建AI处理器实例
func NewAIHandler(newsService *services.NewsService) *AIHandler {
	return &AIHandler{
		aiService:   services.NewAIService(database.GetDB()),
		newsService: newsService,
	}
}

// NewAIHandlerWithConfig 使用指定配置创建AI处理器实例
func NewAIHandlerWithConfig(newsService *services.NewsService, cfg *config.Config) *AIHandler {
	return &AIHandler{
		aiService:   services.NewAIServiceWithConfig(database.GetDB(), cfg),
		newsService: newsService,
	}
}

// AnalyzeNews 分析新闻
// @Summary 分析新闻内容
// @Description 对新闻进行AI分析，包括摘要、关键词提取、情感分析、趋势预测等
// @Tags AI
// @Accept json
// @Produce json
// @Param request body models.AIAnalysisRequest true "分析请求"
// @Success 200 {object} models.AIAnalysisResponse
// @Router /api/ai/analyze [post]
func (h *AIHandler) AnalyzeNews(c *gin.Context) {
	var req models.AIAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请求参数无效")
		return
	}

	// 如果没有指定选项，启用所有基础功能
	if !req.Options.EnableSummary && !req.Options.EnableKeywords &&
		!req.Options.EnableSentiment && !req.Options.EnableTrends &&
		!req.Options.EnableImpact {
		req.Options.EnableSummary = true
		req.Options.EnableKeywords = true
		req.Options.EnableSentiment = true
	}

	// 执行分析
	analysis, err := h.aiService.AnalyzeNews(req.TargetID, req)
	if err != nil {
		utils.InternalServerError(c, "分析失败")
		return
	}

	utils.Success(c, analysis.ToResponse())
}

// AnalyzeEvent 分析事件
// @Summary 分析事件
// @Description 对事件进行深度AI分析，预测未来走向
// @Tags AI
// @Accept json
// @Produce json
// @Param request body models.AIAnalysisRequest true "分析请求"
// @Success 200 {object} models.AIAnalysisResponse
// @Router /api/ai/analyze-event [post]
func (h *AIHandler) AnalyzeEvent(c *gin.Context) {
	var req models.AIAnalysisRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请求参数无效")
		return
	}

	// 确保类型正确
	req.Type = models.AIAnalysisTypeEvent

	// 如果没有指定选项，启用所有功能
	if !req.Options.EnableSummary && !req.Options.EnableKeywords &&
		!req.Options.EnableSentiment && !req.Options.EnableTrends &&
		!req.Options.EnableImpact {
		req.Options.EnableSummary = true
		req.Options.EnableKeywords = true
		req.Options.EnableSentiment = true
		req.Options.EnableTrends = true
		req.Options.EnableImpact = true
		req.Options.ShowAnalysisSteps = true
	}

	// 执行分析
	analysis, err := h.aiService.AnalyzeEvent(req.TargetID, req)
	if err != nil {
		utils.InternalServerError(c, "分析失败")
		return
	}

	utils.Success(c, analysis.ToResponse())
}

// GetAnalysis 获取分析结果
// @Summary 获取分析结果
// @Description 根据类型和目标ID获取已有的分析结果
// @Tags AI
// @Accept json
// @Produce json
// @Param type query string true "分析类型（news/event）"
// @Param target_id query int true "目标ID"
// @Success 200 {object} models.AIAnalysisResponse
// @Router /api/ai/analysis [get]
func (h *AIHandler) GetAnalysis(c *gin.Context) {
	analysisType := c.Query("type")
	targetIDStr := c.Query("target_id")

	if analysisType == "" || targetIDStr == "" {
		utils.BadRequest(c, "缺少必要参数")
		return
	}

	targetID, err := strconv.ParseUint(targetIDStr, 10, 32)
	if err != nil {
		utils.BadRequest(c, "目标ID无效")
		return
	}

	var aiType models.AIAnalysisType
	switch analysisType {
	case "news":
		aiType = models.AIAnalysisTypeNews
	case "event":
		aiType = models.AIAnalysisTypeEvent
	default:
		utils.BadRequest(c, "分析类型无效")
		return
	}

	analysis, err := h.aiService.GetAnalysis(aiType, uint(targetID))
	if err != nil {
		utils.NotFound(c, "分析结果未找到")
		return
	}

	utils.Success(c, analysis.ToResponse())
}

// BatchAnalyzeNews 批量分析新闻
// @Summary 批量分析新闻
// @Description 对多条新闻进行批量AI分析
// @Tags AI
// @Accept json
// @Produce json
// @Param request body BatchAnalysisRequest true "批量分析请求"
// @Success 200 {object} BatchAnalysisResponse
// @Router /api/ai/batch-analyze [post]
func (h *AIHandler) BatchAnalyzeNews(c *gin.Context) {
	var req struct {
		NewsIDs []uint `json:"news_ids" binding:"required"`
		Options struct {
			EnableSummary     bool `json:"enable_summary"`
			EnableKeywords    bool `json:"enable_keywords"`
			EnableSentiment   bool `json:"enable_sentiment"`
			EnableTrends      bool `json:"enable_trends"`
			EnableImpact      bool `json:"enable_impact"`
			ShowAnalysisSteps bool `json:"show_analysis_steps"`
		} `json:"options"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		utils.BadRequest(c, "请求参数无效")
		return
	}

	// 如果没有指定选项，启用基础功能
	if !req.Options.EnableSummary && !req.Options.EnableKeywords && !req.Options.EnableSentiment {
		req.Options.EnableSummary = true
		req.Options.EnableKeywords = true
	}

	results := make([]models.AIAnalysisResponse, 0, len(req.NewsIDs))
	errors := make([]string, 0)

	for _, newsID := range req.NewsIDs {
		analysisReq := models.AIAnalysisRequest{
			Type:     models.AIAnalysisTypeNews,
			TargetID: newsID,
			Options:  req.Options,
		}

		analysis, err := h.aiService.AnalyzeNews(newsID, analysisReq)
		if err != nil {
			errors = append(errors, err.Error())
			continue
		}

		results = append(results, analysis.ToResponse())
	}

	response := gin.H{
		"total":   len(req.NewsIDs),
		"success": len(results),
		"failed":  len(errors),
		"results": results,
		"errors":  errors,
	}

	utils.Success(c, response)
}

// GetAnalysisStats 获取分析统计
// @Summary 获取AI分析统计信息
// @Description 获取AI分析的统计数据，如总分析数、平均处理时间等
// @Tags AI
// @Accept json
// @Produce json
// @Success 200 {object} AnalysisStatsResponse
// @Router /api/ai/stats [get]
func (h *AIHandler) GetAnalysisStats(c *gin.Context) {
	db := database.GetDB()

	var stats struct {
		TotalAnalyses  int64   `json:"total_analyses"`
		NewsAnalyses   int64   `json:"news_analyses"`
		EventAnalyses  int64   `json:"event_analyses"`
		AvgProcessTime float64 `json:"avg_process_time"`
		SuccessRate    float64 `json:"success_rate"`
		TotalProcessed int64   `json:"total_processed"`
		TotalFailed    int64   `json:"total_failed"`
	}

	// 总分析数
	db.Model(&models.AIAnalysis{}).Count(&stats.TotalAnalyses)

	// 新闻分析数
	db.Model(&models.AIAnalysis{}).Where("type = ?", models.AIAnalysisTypeNews).Count(&stats.NewsAnalyses)

	// 事件分析数
	db.Model(&models.AIAnalysis{}).Where("type = ?", models.AIAnalysisTypeEvent).Count(&stats.EventAnalyses)

	// 平均处理时间
	db.Model(&models.AIAnalysis{}).Select("AVG(processing_time)").Scan(&stats.AvgProcessTime)

	// 成功和失败数
	db.Model(&models.AIAnalysis{}).Where("status = ?", "completed").Count(&stats.TotalProcessed)
	db.Model(&models.AIAnalysis{}).Where("status = ?", "failed").Count(&stats.TotalFailed)

	// 计算成功率
	if stats.TotalAnalyses > 0 {
		stats.SuccessRate = float64(stats.TotalProcessed) / float64(stats.TotalAnalyses) * 100
	}

	utils.Success(c, stats)
}

// SummarizeNews godoc
// @Summary      Summarize a news article
// @Description  get a summary for a news article by its ID
// @Tags         AI
// @Accept       json
// @Produce      json
// @Param        id   path      int  true  "News ID"
// @Success      200  {object}  map[string]string
// @Failure      400  {object}  utils.ErrorResponse
// @Failure      404  {object}  utils.ErrorResponse
// @Failure      500  {object}  utils.ErrorResponse
// @Router       /news/{id}/summarize [post]
func (h *AIHandler) SummarizeNews(c *gin.Context) {
	var req struct {
		NewsID uint `json:"news_id"`
	}

	// 优先从 JSON 里读取
	if err := c.ShouldBindJSON(&req); err != nil || req.NewsID == 0 {
		// 如果 JSON 无效，尝试从 URL 参数获取
		idStr := c.Param("id")
		id, err := strconv.ParseUint(idStr, 10, 64)
		if err != nil || id == 0 {
			utils.BadRequest(c, "Invalid news ID")
			return
		}
		req.NewsID = uint(id)

		return
	}

	news, err := h.newsService.GetNewsByID(req.NewsID)
	if err != nil {
		utils.NotFound(c, "News not found")
		return
	}

	// 使用AI服务进行总结
	analysisReq := models.AIAnalysisRequest{
		Type:     models.AIAnalysisTypeNews,
		TargetID: news.ID,
	}
	analysisReq.Options.EnableSummary = true

	analysis, err := h.aiService.AnalyzeNews(news.ID, analysisReq)
	if err != nil {

		utils.InternalServerError(c, "Failed to generate summary")

		return
	}

	utils.Success(c, gin.H{"summary": analysis.Summary})
}

// BatchAnalyzeUnprocessedNews 批量分析未处理的新闻
// @Summary 批量分析未处理的新闻
// @Description 批量分析数据库中未进行AI分析的新闻
// @Tags AI
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{}
// @Router /api/v1/ai/batch-analyze-unprocessed [post]
func (h *AIHandler) BatchAnalyzeUnprocessedNews(c *gin.Context) {
	// 执行批量分析
	err := h.aiService.BatchAnalyzeUnprocessedNews()
	if err != nil {
		utils.InternalServerError(c, "Failed to start batch analysis: "+err.Error())
		return
	}

	utils.Success(c, map[string]interface{}{
		"message": "Batch analysis started successfully",
		"status":  "processing",
	})
}
