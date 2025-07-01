package main

import (
	"log"

	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/services"
)

/*
测试低置信度阈值的事件生成

使用方法：
go run tests/test_low_threshold.go

这个脚本会：
1. 设置低置信度阈值（0.2）
2. 对现有新闻生成事件
3. 显示结果
*/

func main() {
	log.Println("🚀 开始测试低置信度阈值事件生成...")

	// 1. 加载配置并初始化数据库
	cfg, err := config.LoadConfig("internal/config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := database.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.CloseDatabase()

	// 2. 执行数据库迁移
	if err := database.Migrate(&models.User{}, &models.News{}, &models.Event{}, &models.RSSSource{}); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// 3. 创建自定义AI配置（低置信度阈值）
	customConfig := &services.AIServiceConfig{
		Provider:    "openai",
		APIKey:      "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",
		APIEndpoint: "https://api.openai.com/v1/chat/completions",
		Model:       "gpt-3.5-turbo",
		MaxTokens:   200000,
		Timeout:     30,
		Enabled:     true,
	}

	// 设置事件生成配置
	customConfig.EventGeneration.Enabled = true
	customConfig.EventGeneration.ConfidenceThreshold = 0.05 // 🎯 设置极低的阈值
	customConfig.EventGeneration.MinNewsCount = 2
	customConfig.EventGeneration.TimeWindowHours = 24
	customConfig.EventGeneration.MaxNewsLimit = 50

	// 4. 创建带自定义配置的种子服务
	seedService := services.NewSeedServiceWithAIConfig(customConfig)

	log.Printf("🤖 当前AI配置:")
	log.Printf("   - 置信度阈值: %.2f", customConfig.EventGeneration.ConfidenceThreshold)
	log.Printf("   - 最小新闻数量: %d", customConfig.EventGeneration.MinNewsCount)

	// 5. 清空事件数据库并重置新闻关联
	db := database.GetDB()

	log.Println("\n🗑️  清空事件数据库...")

	// 先将所有新闻的事件关联设置为NULL
	if err := db.Model(&models.News{}).Where("1=1").Update("belonged_event_id", nil).Error; err != nil {
		log.Printf("重置新闻事件关联失败: %v", err)
		return
	}

	// 然后删除所有事件
	if err := db.Unscoped().Delete(&models.Event{}, "1=1").Error; err != nil {
		log.Printf("清空事件表失败: %v", err)
		return
	}

	log.Println("✅ 事件数据库已清空，新闻事件关联已重置")

	// 统计测试前数据
	var newsCount, eventCountBefore int64
	db.Model(&models.News{}).Count(&newsCount)
	db.Model(&models.Event{}).Count(&eventCountBefore)

	log.Printf("\n📊 测试前数据:")
	log.Printf("   - 总新闻数量: %d", newsCount)
	log.Printf("   - 事件总数: %d", eventCountBefore)

	// 6. 执行事件生成
	log.Println("\n🎯 开始生成事件...")
	if err := seedService.GenerateEventsFromNewsWithDefaults(); err != nil {
		log.Printf("❌ 事件生成失败: %v", err)
		return
	}

	// 7. 统计结果
	var eventCountAfter, linkedNewsCount int64
	db.Model(&models.Event{}).Count(&eventCountAfter)
	db.Model(&models.News{}).Where("belonged_event_id IS NOT NULL").Count(&linkedNewsCount)

	log.Printf("\n📊 测试结果:")
	log.Printf("   - 新生成事件数量: %d", eventCountAfter-eventCountBefore)
	log.Printf("   - 已关联新闻总数: %d", linkedNewsCount)

	// 8. 显示生成的事件
	if eventCountAfter > eventCountBefore {
		log.Printf("\n🎉 成功生成 %d 个事件!", eventCountAfter-eventCountBefore)

		var events []models.Event
		db.Order("created_at DESC").Limit(3).Find(&events)

		log.Println("\n📋 最新生成的事件:")
		for i, event := range events {
			log.Printf("   %d. %s", i+1, event.Title)
			log.Printf("      分类: %s | 状态: %s", event.Category, event.Status)

			// 显示关联的新闻
			var relatedNews []models.News
			db.Where("belonged_event_id = ?", event.ID).Find(&relatedNews)
			log.Printf("      关联新闻: %d条", len(relatedNews))
			for j, news := range relatedNews {
				if j < 2 { // 只显示前2条
					log.Printf("        - %s", news.Title[:min(50, len(news.Title))])
				}
			}
		}
	} else {
		log.Println("\n⚠️  仍未生成新事件")
		log.Println("   可能需要进一步降低阈值或检查AI API")
	}

	log.Println("\n✅ 测试完成!")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
