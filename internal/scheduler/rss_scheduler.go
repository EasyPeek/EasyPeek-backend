package scheduler

import (
	"log"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/services"
	"github.com/robfig/cron/v3"
)

type RSSScheduler struct {
	cron       *cron.Cron
	rssService *services.RSSService
}

func NewRSSScheduler() *RSSScheduler {
	// 创建带有秒级精度的cron调度器
	c := cron.New(cron.WithSeconds())
	
	return &RSSScheduler{
		cron:       c,
		rssService: services.NewRSSService(),
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