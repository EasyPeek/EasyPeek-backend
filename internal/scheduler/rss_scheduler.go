package scheduler

import (
	"fmt"
	"log"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/services"
	"github.com/robfig/cron/v3"
)

type RSSScheduler struct {
	cron       *cron.Cron
	rssService *services.RSSService
	aiService  *services.AIService
}

func NewRSSScheduler() *RSSScheduler {
	// 创建带有秒级精度的cron调度器
	c := cron.New(cron.WithSeconds())
	db := database.GetDB()

	return &RSSScheduler{
		cron:       c,
		rssService: services.NewRSSService(),
		aiService:  services.NewAIService(db),
	}
}

// Start 启动RSS调度器
func (s *RSSScheduler) Start() error {
	// 每30分钟抓取一次RSS
	_, err := s.cron.AddFunc("0 */30 * * * *", s.fetchAllRSSFeeds)
	if err != nil {
		return err
	}

	// 每小时清理一次过期数据
	_, err = s.cron.AddFunc("0 0 * * * *", s.cleanupOldNews)
	if err != nil {
		return err
	}

	// 每6小时重新计算热度
	_, err = s.cron.AddFunc("0 0 */6 * * *", s.recalculateHotness)
	if err != nil {
		return err
	}

	// 根据配置添加AI分析任务
	if config.AppConfig != nil && config.AppConfig.AI.AutoAnalysis.Enabled {
		interval := config.AppConfig.AI.AutoAnalysis.BatchProcessInterval
		if interval <= 0 {
			interval = 10 // 默认10分钟
		}

		// 构建cron表达式：每N分钟执行一次
		cronExpr := fmt.Sprintf("0 */%d * * * *", interval)
		_, err = s.cron.AddFunc(cronExpr, s.processUnanalyzedNews)
		if err != nil {
			return err
		}
		log.Printf("AI batch processing scheduled every %d minutes", interval)
	} else {
		log.Println("AI batch processing disabled in configuration")
	}

	// 启动调度器
	s.cron.Start()
	log.Println("RSS scheduler started")

	// 启动时立即执行一次抓取
	go s.fetchAllRSSFeeds()

	return nil
}

// Stop 停止RSS调度器
func (s *RSSScheduler) Stop() {
	s.cron.Stop()
	log.Println("RSS scheduler stopped")
}

// fetchAllRSSFeeds 抓取所有RSS源
func (s *RSSScheduler) fetchAllRSSFeeds() {
	log.Println("[RSS SCHEDULER] Starting scheduled RSS fetch...")

	result, err := s.rssService.FetchAllRSSFeeds()
	if err != nil {
		log.Printf("[RSS SCHEDULER ERROR] Scheduled RSS fetch failed: %v", err)
		return
	}

	log.Printf("[RSS SCHEDULER] Scheduled RSS fetch completed: %s", result.Message)

	// 记录详细统计信息
	totalNew := 0
	totalUpdated := 0
	totalErrors := 0

	for _, stats := range result.Stats {
		totalNew += stats.NewItems
		totalUpdated += stats.UpdatedItems
		totalErrors += stats.ErrorItems

		if stats.ErrorItems > 0 {
			log.Printf("RSS source %s had %d errors", stats.SourceName, stats.ErrorItems)
		}
	}

	log.Printf("RSS fetch summary - New: %d, Updated: %d, Errors: %d",
		totalNew, totalUpdated, totalErrors)
}

// cleanupOldNews 清理过期新闻
func (s *RSSScheduler) cleanupOldNews() {
	log.Println("Starting news cleanup...")

	// 这里可以实现清理逻辑，比如：
	// 1. 删除超过30天的新闻
	// 2. 归档低热度的旧新闻
	// 3. 清理重复的新闻条目

	// 示例：标记30天前的新闻为已归档
	cutoffDate := time.Now().AddDate(0, 0, -30)

	// 这里需要在RSSService中添加相应的方法
	// err := s.rssService.ArchiveOldNews(cutoffDate)
	// if err != nil {
	//     log.Printf("News cleanup failed: %v", err)
	//     return
	// }

	log.Printf("News cleanup completed for items older than %s", cutoffDate.Format("2006-01-02"))
}

// recalculateHotness 重新计算热度
func (s *RSSScheduler) recalculateHotness() {
	log.Println("Starting hotness recalculation...")

	// 这里可以实现批量重新计算热度的逻辑
	// 比如重新计算最近7天内的所有新闻热度

	// 示例实现需要在RSSService中添加相应方法
	// err := s.rssService.RecalculateAllHotness()
	// if err != nil {
	//     log.Printf("Hotness recalculation failed: %v", err)
	//     return
	// }

	log.Println("Hotness recalculation completed")
}

// processUnanalyzedNews 处理未分析的新闻
func (s *RSSScheduler) processUnanalyzedNews() {
	// 检查配置是否启用
	if config.AppConfig == nil || !config.AppConfig.AI.AutoAnalysis.Enabled {
		log.Println("[AI SCHEDULER] AI auto analysis disabled in config")
		return
	}

	log.Println("[AI SCHEDULER] Starting to process unanalyzed news...")

	db := database.GetDB()

	// 从配置获取批处理大小
	maxBatchSize := config.AppConfig.AI.AutoAnalysis.MaxBatchSize
	if maxBatchSize <= 0 {
		maxBatchSize = 10 // 默认值
	}

	// 从配置获取分析延迟
	analysisDelay := config.AppConfig.AI.AutoAnalysis.AnalysisDelay
	if analysisDelay <= 0 {
		analysisDelay = 2 // 默认值
	}

	// 查找未分析的新闻（最近24小时内的新闻且没有AI分析记录）
	var unanalyzedNews []models.News

	// 查询条件：
	// 1. 最近24小时内的新闻
	// 2. 没有AI分析记录
	// 3. 限制数量由配置决定
	cutoffTime := time.Now().Add(-24 * time.Hour)

	query := `
		SELECT n.* FROM news n 
		LEFT JOIN ai_analyses a ON n.id = a.target_id AND a.type = ?
		WHERE n.created_at > ? 
		AND a.id IS NULL 
		AND n.source_type = ?
		ORDER BY n.created_at DESC 
		LIMIT ?
	`

	if err := db.Raw(query, models.AIAnalysisTypeNews, cutoffTime, models.NewsTypeRSS, maxBatchSize).Find(&unanalyzedNews).Error; err != nil {
		log.Printf("[AI SCHEDULER ERROR] Failed to find unanalyzed news: %v", err)
		return
	}

	if len(unanalyzedNews) == 0 {
		log.Println("[AI SCHEDULER] No unanalyzed news found")
		return
	}

	log.Printf("[AI SCHEDULER] Found %d unanalyzed news items", len(unanalyzedNews))

	// 逐个处理
	successCount := 0
	errorCount := 0

	for _, news := range unanalyzedNews {
		log.Printf("[AI SCHEDULER] Processing news ID: %d - %s", news.ID, news.Title)

		// 设置AI分析选项
		analysisOptions := models.AIAnalysisRequest{
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
				EnableSummary:     true,
				EnableKeywords:    true,
				EnableSentiment:   true,
				EnableTrends:      false, // 定时任务中不开启高级功能
				EnableImpact:      false,
				ShowAnalysisSteps: false,
			},
		}

		// 执行AI分析
		analysis, err := s.aiService.AnalyzeNews(news.ID, analysisOptions)
		if err != nil {
			log.Printf("[AI SCHEDULER ERROR] Failed to analyze news ID %d: %v", news.ID, err)
			errorCount++
			continue
		}

		// 更新新闻摘要
		if analysis.Summary != "" && (news.Summary == "" || len(news.Summary) < 50) {
			if err := db.Model(&news).Update("summary", analysis.Summary).Error; err != nil {
				log.Printf("[AI SCHEDULER ERROR] Failed to update summary for news ID %d: %v", news.ID, err)
			}
		}

		successCount++
		log.Printf("[AI SCHEDULER] Successfully analyzed news ID: %d", news.ID)

		// 每个分析之间稍微延迟，避免API频率限制
		time.Sleep(time.Duration(analysisDelay) * time.Second)
	}

	log.Printf("[AI SCHEDULER] Completed processing unanalyzed news - Success: %d, Errors: %d",
		successCount, errorCount)
}

// AddCustomJob 添加自定义定时任务
func (s *RSSScheduler) AddCustomJob(spec string, cmd func()) error {
	_, err := s.cron.AddFunc(spec, cmd)
	return err
}

// GetNextRun 获取下次运行时间
func (s *RSSScheduler) GetNextRun() []time.Time {
	entries := s.cron.Entries()
	var nextRuns []time.Time

	for _, entry := range entries {
		nextRuns = append(nextRuns, entry.Next)
	}

	return nextRuns
}
