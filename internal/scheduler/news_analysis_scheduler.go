package scheduler

import (
	"log"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/services"
)

// NewsAnalysisScheduler 新闻AI分析调度器
type NewsAnalysisScheduler struct {
	aiService *services.AIService
	ticker    *time.Ticker
	done      chan bool
	running   bool
}

// NewNewsAnalysisScheduler 创建新闻AI分析调度器
func NewNewsAnalysisScheduler() *NewsAnalysisScheduler {
	return &NewsAnalysisScheduler{
		aiService: services.NewAIService(database.GetDB()),
		done:      make(chan bool),
	}
}

// Start 启动调度器
func (s *NewsAnalysisScheduler) Start() error {
	if s.running {
		return nil
	}

	s.ticker = time.NewTicker(15 * time.Minute) // 每15分钟执行一次
	s.running = true

	go s.run()
	log.Println("📊 News AI Analysis Scheduler started (every 15 minutes)")

	// 立即执行一次
	go func() {
		time.Sleep(5 * time.Second) // 等待5秒后执行，避免启动时的资源争抢
		s.analyzeUnprocessedNews()
	}()

	return nil
}

// Stop 停止调度器
func (s *NewsAnalysisScheduler) Stop() {
	if !s.running {
		return
	}

	s.running = false
	if s.ticker != nil {
		s.ticker.Stop()
	}

	close(s.done)
	log.Println("📊 News AI Analysis Scheduler stopped")
}

// run 运行调度器主循环
func (s *NewsAnalysisScheduler) run() {
	for {
		select {
		case <-s.ticker.C:
			s.analyzeUnprocessedNews()
		case <-s.done:
			return
		}
	}
}

// analyzeUnprocessedNews 分析未处理的新闻
func (s *NewsAnalysisScheduler) analyzeUnprocessedNews() {
	log.Printf("🤖 [AI Scheduler] 开始定时分析未处理的新闻...")

	startTime := time.Now()
	err := s.aiService.BatchAnalyzeUnprocessedNews()
	duration := time.Since(startTime)

	if err != nil {
		log.Printf("❌ [AI Scheduler] 新闻AI分析失败: %v (耗时: %v)", err, duration)
	} else {
		log.Printf("✅ [AI Scheduler] 新闻AI分析完成 (耗时: %v)", duration)
	}
}

// AnalyzeNewsAsync 异步分析指定新闻（供其他服务调用）
func (s *NewsAnalysisScheduler) AnalyzeNewsAsync(newsID uint) {
	go func() {
		log.Printf("🤖 [AI Scheduler] 异步分析新闻 ID: %d", newsID)

		// 使用重试机制
		s.aiService.AnalyzeNewsWithRetry(newsID, 3)
	}()
}

// IsRunning 检查调度器是否正在运行
func (s *NewsAnalysisScheduler) IsRunning() bool {
	return s.running
}
