package main

import (
	"fmt"
	"log"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/services"
)

func main() {
	fmt.Println("🔄 EasyPeek 新闻导入与事件生成测试工具")
	fmt.Println("========================================")

	startTime := time.Now()

	// 1. 加载配置
	cfg, err := config.LoadConfig("internal/config/config.yaml")
	if err != nil {
		log.Fatalf("❌ 无法加载配置: %v", err)
	}

	// 2. 初始化数据库
	if err := database.Initialize(cfg); err != nil {
		log.Fatalf("❌ 无法初始化数据库: %v", err)
	}
	defer database.CloseDatabase()

	// 3. 确保数据库表存在
	if err := database.Migrate(
		&models.User{},
		&models.Event{},
		&models.RSSSource{},
		&models.News{},
	); err != nil {
		log.Fatalf("❌ 数据库迁移失败: %v", err)
	}
	fmt.Println("✅ 数据库连接和迁移成功")

	// 4. 测试新闻导入
	fmt.Println("\n🔄 开始导入新闻数据...")
	seedService := services.NewSeedService()

	// 检查当前新闻数量
	db := database.GetDB()
	var newsCount int64
	if err := db.Model(&models.News{}).Count(&newsCount).Error; err != nil {
		log.Fatalf("❌ 查询新闻数量失败: %v", err)
	}
	fmt.Printf("📊 当前数据库中有 %d 条新闻\n", newsCount)

	// 导入新闻数据 (使用用户指定的路径)
	if err := seedService.SeedNewsFromJSON("data/new.json"); err != nil {
		log.Fatalf("❌ 新闻导入失败: %v", err)
	}

	// 检查导入后的新闻数量
	var newNewsCount int64
	if err := db.Model(&models.News{}).Count(&newNewsCount).Error; err != nil {
		log.Fatalf("❌ 查询导入后新闻数量失败: %v", err)
	}
	fmt.Printf("✅ 导入完成，现在数据库中有 %d 条新闻\n", newNewsCount)

	// 5. 测试事件生成
	fmt.Println("\n🔄 开始生成事件...")
	eventService := services.NewEventService()

	// 检查当前事件数量
	var eventCount int64
	if err := db.Model(&models.Event{}).Count(&eventCount).Error; err != nil {
		log.Fatalf("❌ 查询事件数量失败: %v", err)
	}
	fmt.Printf("📊 当前数据库中有 %d 个事件\n", eventCount)

	// 生成事件
	result, err := eventService.GenerateEventsFromNews()
	if err != nil {
		log.Fatalf("❌ 事件生成失败: %v", err)
	}

	// 显示生成结果
	fmt.Println("✅ 事件生成完成！")
	fmt.Printf("📈 生成统计:\n")
	fmt.Printf("   - 生成事件数: %d\n", result.TotalEvents)
	fmt.Printf("   - 处理新闻数: %d\n", result.ProcessedNews)
	fmt.Printf("   - 生成时间: %s\n", result.GenerationTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("   - 耗时: %s\n", result.ElapsedTime)

	// 分类统计
	fmt.Println("\n📊 按分类统计:")
	for category, count := range result.CategoryBreakdown {
		fmt.Printf("   - %s: %d 个事件\n", category, count)
	}

	// 6. 检查新闻-事件关联
	fmt.Println("\n🔄 检查新闻与事件关联...")
	newsService := services.NewNewsService()

	// 查询未关联事件的新闻
	unlinkedNews, total, err := newsService.GetUnlinkedNews(1, 5)
	if err != nil {
		log.Printf("⚠️  查询未关联新闻失败: %v\n", err)
	} else {
		fmt.Printf("📊 未关联事件的新闻: %d 条\n", total)
		if len(unlinkedNews) > 0 {
			fmt.Printf("   前 %d 条未关联新闻:\n", len(unlinkedNews))
			for i, news := range unlinkedNews {
				fmt.Printf("   %d. %s (分类: %s)\n", i+1, news.Title, news.Category)
			}
		}
	}

	elapsed := time.Since(startTime)
	fmt.Println("========================================")
	fmt.Printf("✅ 测试完成! 总耗时: %.2f秒\n", elapsed.Seconds())
	fmt.Println("\n💡 提示:")
	fmt.Println("   - 可通过 API POST /api/v1/admin/events/generate 手动生成事件")
	fmt.Println("   - 可通过 API GET /api/v1/events 查看生成的事件")
	fmt.Println("   - 可通过 API GET /api/v1/news 查看导入的新闻")
}
