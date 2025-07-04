package scheduler

import (
	"log"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/services"
	"github.com/robfig/cron/v3"
)

type EventScheduler struct {
	cron         *cron.Cron
	eventService *services.EventService
}

func NewEventScheduler() *EventScheduler {
	// 创建带有秒级精度的cron调度器
	c := cron.New(cron.WithSeconds())

	return &EventScheduler{
		cron:         c,
		eventService: services.NewEventService(),
	}
}

// Start 启动事件调度器
func (s *EventScheduler) Start() error {
	// 每2小时更新一次事件统计信息
	_, err := s.cron.AddFunc("0 0 */2 * * *", s.updateAllEventStats)
	if err != nil {
		return err
	}

	// 每4小时刷新一次事件热度
	_, err = s.cron.AddFunc("0 30 */4 * * *", s.refreshAllEventHotness)
	if err != nil {
		return err
	}

	// 每天午夜清理过期事件状态
	_, err = s.cron.AddFunc("0 0 0 * * *", s.updateEventStatus)
	if err != nil {
		return err
	}

	// 启动调度器
	s.cron.Start()
	log.Println("Event scheduler started")

	// 启动时立即执行一次统计信息更新
	go s.updateAllEventStats()

	return nil
}

// Stop 停止事件调度器
func (s *EventScheduler) Stop() {
	s.cron.Stop()
	log.Println("Event scheduler stopped")
}

// updateAllEventStats 更新所有事件统计信息
func (s *EventScheduler) updateAllEventStats() {
	log.Println("[EVENT SCHEDULER] Starting scheduled event stats update...")

	startTime := time.Now()
	err := s.eventService.UpdateAllEventStats()
	if err != nil {
		log.Printf("[EVENT SCHEDULER ERROR] Event stats update failed: %v", err)
		return
	}

	elapsed := time.Since(startTime)
	log.Printf("[EVENT SCHEDULER] Event stats update completed in %s", elapsed)
}

// refreshAllEventHotness 刷新所有事件热度
func (s *EventScheduler) refreshAllEventHotness() {
	log.Println("[EVENT SCHEDULER] Starting scheduled event hotness refresh...")

	startTime := time.Now()

	// 获取所有事件ID
	events, err := s.eventService.GetEvents(&models.EventQueryRequest{
		Page:  1,
		Limit: 1000, // 假设系统中不会超过1000个事件
	})
	if err != nil {
		log.Printf("[EVENT SCHEDULER ERROR] Failed to get events for hotness refresh: %v", err)
		return
	}

	successCount := 0
	errorCount := 0

	for _, event := range events.Events {
		err := s.eventService.RefreshEventHotness(event.ID)
		if err != nil {
			log.Printf("[EVENT SCHEDULER ERROR] Failed to refresh hotness for event %d: %v", event.ID, err)
			errorCount++
		} else {
			successCount++
		}
	}

	elapsed := time.Since(startTime)
	log.Printf("[EVENT SCHEDULER] Event hotness refresh completed in %s - Success: %d, Errors: %d",
		elapsed, successCount, errorCount)
}

// updateEventStatus 更新事件状态
func (s *EventScheduler) updateEventStatus() {
	log.Println("[EVENT SCHEDULER] Starting scheduled event status update...")

	err := s.eventService.UpdateEventStatus()
	if err != nil {
		log.Printf("[EVENT SCHEDULER ERROR] Event status update failed: %v", err)
		return
	}

	log.Println("[EVENT SCHEDULER] Event status update completed")
}

// AddCustomJob 添加自定义定时任务
func (s *EventScheduler) AddCustomJob(spec string, cmd func()) error {
	_, err := s.cron.AddFunc(spec, cmd)
	return err
}

// GetNextRun 获取下次运行时间
func (s *EventScheduler) GetNextRun() []time.Time {
	entries := s.cron.Entries()
	var nextRuns []time.Time

	for _, entry := range entries {
		nextRuns = append(nextRuns, entry.Next)
	}

	return nextRuns
}

// ForceUpdateAllStats 强制立即更新所有事件统计信息
func (s *EventScheduler) ForceUpdateAllStats() {
	log.Println("[EVENT SCHEDULER] Force updating all event stats...")
	s.updateAllEventStats()
}

// ForceRefreshAllHotness 强制立即刷新所有事件热度
func (s *EventScheduler) ForceRefreshAllHotness() {
	log.Println("[EVENT SCHEDULER] Force refreshing all event hotness...")
	s.refreshAllEventHotness()
}
