package main

import (
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
)

func main() {
	fmt.Println("🔄 EasyPeek 新闻-事件关联工具")
	fmt.Println("====================================")

	// 加载配置
	cfg, err := config.LoadConfig("internal/config/config.yaml")
	if err != nil {
		log.Fatalf("无法加载配置: %v", err)
	}

	// 初始化数据库
	if err := database.Initialize(cfg); err != nil {
		log.Fatalf("无法初始化数据库: %v", err)
	}
	defer database.CloseDatabase()

	db := database.GetDB()

	// 获取所有事件
	var events []models.Event
	if err := db.Find(&events).Error; err != nil {
		log.Fatalf("获取事件列表失败: %v", err)
	}
	fmt.Printf("✅ 获取到 %d 个事件\n", len(events))

	// 获取所有新闻
	var news []models.News
	if err := db.Find(&news).Error; err != nil {
		log.Fatalf("获取新闻列表失败: %v", err)
	}
	fmt.Printf("✅ 获取到 %d 条新闻\n", len(news))

	// 根据标题关键词和时间关联新闻到事件
	matchCount := 0
	for i := range news {
		newsItem := &news[i]

		// 如果已经关联了事件，跳过
		if newsItem.BelongedEventID != nil {
			continue
		}

		// 查找匹配的事件
		matchedEvent := findMatchingEvent(newsItem, events)
		if matchedEvent != nil {
			// 更新新闻的事件关联
			newsItem.BelongedEventID = &matchedEvent.ID
			if err := db.Save(newsItem).Error; err != nil {
				log.Printf("❌ 更新新闻 #%d 失败: %v", newsItem.ID, err)
				continue
			}
			matchCount++
			log.Printf("✅ 新闻 #%d [%s] 关联到事件 #%d [%s]",
				newsItem.ID, newsItem.Title, matchedEvent.ID, matchedEvent.Title)
		}
	}

	fmt.Printf("\n====================================\n")
	fmt.Printf("✅ 完成! 共关联 %d/%d 条新闻到事件\n", matchCount, len(news))
}

// findMatchingEvent 根据新闻内容和时间查找匹配的事件
func findMatchingEvent(news *models.News, events []models.Event) *models.Event {
	// 首先，根据分类进行初步匹配
	var categoryMatches []models.Event
	for _, event := range events {
		if news.Category == event.Category {
			categoryMatches = append(categoryMatches, event)
		}
	}

	// 如果同一分类下没有事件，尝试所有事件
	matchPool := categoryMatches
	if len(matchPool) == 0 {
		matchPool = events
	}

	// 根据标题关键词匹配
	var keywordMatches []models.Event
	for _, event := range matchPool {
		// 检查标题中是否包含关键词
		if containsEventKeywords(news.Title, event.Title) ||
			(news.Content != "" && containsEventKeywords(news.Content, event.Title)) {
			keywordMatches = append(keywordMatches, event)
		}
	}

	// 如果有基于关键词的匹配，返回最近的一个
	if len(keywordMatches) > 0 {
		closestEvent := findClosestByTime(news.PublishedAt, keywordMatches)
		return &closestEvent
	}

	// 如果没有关键词匹配，检查时间匹配
	// 新闻发布在事件期间或事件前后14天内的视为相关
	var timeMatches []models.Event
	for _, event := range matchPool {
		eventStartTime := event.StartTime.AddDate(0, 0, -14) // 事件开始前14天
		eventEndTime := event.EndTime.AddDate(0, 0, 14)      // 事件结束后14天

		if news.PublishedAt.After(eventStartTime) && news.PublishedAt.Before(eventEndTime) {
			timeMatches = append(timeMatches, event)
		}
	}

	// 如果有时间匹配，返回最接近的一个
	if len(timeMatches) > 0 {
		closestEvent := findClosestByTime(news.PublishedAt, timeMatches)
		return &closestEvent
	}

	return nil
}

// findClosestByTime 在事件列表中找到时间上最接近的事件
func findClosestByTime(newsTime time.Time, events []models.Event) models.Event {
	closestEvent := events[0]
	closestDiff := timeDifference(newsTime, events[0].StartTime)

	for i := 1; i < len(events); i++ {
		diff := timeDifference(newsTime, events[i].StartTime)
		if diff < closestDiff {
			closestEvent = events[i]
			closestDiff = diff
		}
	}

	return closestEvent
}

// containsEventKeywords 检查文本是否包含事件关键词
func containsEventKeywords(text string, eventTitle string) bool {
	// 将文本和事件标题转为小写，便于比较
	textLower := strings.ToLower(text)
	eventTitleLower := strings.ToLower(eventTitle)

	// 分割事件标题为关键词
	keywords := strings.Fields(eventTitleLower)

	// 过滤掉太短的关键词（如"的"、"和"等）
	var validKeywords []string
	for _, kw := range keywords {
		if len(kw) >= 2 {
			validKeywords = append(validKeywords, kw)
		}
	}

	// 如果没有有效关键词，返回false
	if len(validKeywords) == 0 {
		return false
	}

	// 至少需要匹配30%的有效关键词
	minMatchCount := int(float64(len(validKeywords)) * 0.3)
	if minMatchCount < 1 {
		minMatchCount = 1
	}

	// 计算匹配的关键词数量
	matchCount := 0
	for _, keyword := range validKeywords {
		if strings.Contains(textLower, keyword) {
			matchCount++
		}
	}

	return matchCount >= minMatchCount
}

// timeDifference 计算两个时间的绝对差值，返回秒数
func timeDifference(t1, t2 time.Time) int64 {
	diff := t1.Sub(t2)
	if diff < 0 {
		diff = -diff
	}
	return int64(diff.Seconds())
}
