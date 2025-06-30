package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type News struct {
	ID          uint `gorm:"primarykey"`
	Title       string
	Category    string
	ViewCount   int64  `gorm:"column:view_count"`
	LikeCount   int64  `gorm:"column:like_count"`
	PublishedAt string `gorm:"column:published_at"`
}

type CategoryStats struct {
	Category string
	Count    int64
}

func main() {
	// 使用环境变量或默认配置连接数据库
	dsn := "host=localhost user=postgres password=password dbname=easypeek port=5432 sslmode=disable TimeZone=Asia/Shanghai"

	// 如果有环境变量，使用环境变量
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Shanghai",
			getEnv("DB_HOST", "localhost"),
			getEnv("DB_USER", "postgres"),
			getEnv("DB_PASSWORD", "password"),
			getEnv("DB_NAME", "easypeek"),
			getEnv("DB_PORT", "5432"),
			getEnv("DB_SSLMODE", "disable"),
		)
	}

	// 连接数据库
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// 获取原始SQL连接
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get SQL DB:", err)
	}
	defer sqlDB.Close()

	fmt.Println("🔍 EasyPeek 数据库验证报告")
	fmt.Println(strings.Repeat("=", 50))

	// 1. 检查表是否存在
	var tableCount int64
	db.Raw("SELECT COUNT(*) FROM information_schema.tables WHERE table_name = 'news'").Scan(&tableCount)
	if tableCount > 0 {
		fmt.Println("✅ news 表已创建")
	} else {
		fmt.Println("❌ news 表不存在")
		return
	}

	// 2. 统计新闻总数
	var newsCount int64
	db.Model(&News{}).Count(&newsCount)
	fmt.Printf("📊 新闻总数: %d 条\n", newsCount)

	// 3. 按分类统计
	var categoryStats []CategoryStats
	db.Model(&News{}).Select("category, COUNT(*) as count").Group("category").Find(&categoryStats)

	fmt.Println("\n📈 分类统计:")
	for _, stat := range categoryStats {
		fmt.Printf("   %s: %d 条\n", stat.Category, stat.Count)
	}

	// 4. 显示最新的5条新闻
	var latestNews []News
	db.Select("id, title, category, view_count, like_count, published_at").
		Order("published_at DESC").
		Limit(5).
		Find(&latestNews)

	fmt.Println("\n🗞️ 最新新闻 (前5条):")
	for i, news := range latestNews {
		fmt.Printf("   %d. [%s] %s (浏览:%d, 点赞:%d)\n",
			i+1, news.Category, news.Title, news.ViewCount, news.LikeCount)
	}

	// 5. 热度最高的新闻
	var hotNews []News
	db.Select("id, title, category, view_count, like_count").
		Order("(view_count * 0.3 + like_count * 0.7) DESC").
		Limit(3).
		Find(&hotNews)

	fmt.Println("\n🔥 热度最高新闻 (前3条):")
	for i, news := range hotNews {
		hotScore := float64(news.ViewCount)*0.3 + float64(news.LikeCount)*0.7
		fmt.Printf("   %d. [%s] %s (热度:%.1f)\n",
			i+1, news.Category, news.Title, hotScore)
	}

	// 6. 检查热度计算函数
	var functionExists int64
	db.Raw("SELECT COUNT(*) FROM pg_proc WHERE proname = 'calculate_news_hotness'").Scan(&functionExists)
	if functionExists > 0 {
		fmt.Println("\n✅ 热度计算函数已创建")

		// 测试热度函数（修正时区问题）
		var testHotness float64
		err := db.Raw("SELECT calculate_news_hotness(1000::BIGINT, 500::BIGINT, 100::BIGINT, 50::BIGINT, (NOW() - INTERVAL '2 hours')::TIMESTAMP)").Scan(&testHotness).Error
		if err != nil {
			fmt.Printf("⚠️ 热度函数测试失败: %v\n", err)
		} else {
			fmt.Printf("🧮 热度函数测试: calculate_news_hotness(1000,500,100,50,2小时前) = %.2f\n", testHotness)
		}
	} else {
		fmt.Println("\n❌ 热度计算函数不存在")
	}

	// 7. 检查视图
	var viewExists int64
	db.Raw("SELECT COUNT(*) FROM information_schema.views WHERE table_name IN ('news_stats_summary', 'news_with_stats', 'trending_news')").Scan(&viewExists)
	if viewExists > 0 {
		fmt.Printf("✅ 新闻统计视图已创建 (%d个视图)\n", viewExists)

		// 测试视图查询
		var summaryCount int64
		db.Raw("SELECT COUNT(*) FROM news_stats_summary").Scan(&summaryCount)
		fmt.Printf("📊 统计汇总视图包含 %d 个分类\n", summaryCount)
	} else {
		fmt.Println("❌ 新闻统计视图不存在")
	}

	fmt.Println("\n" + strings.Repeat("=", 50))
	fmt.Println("🎉 数据库验证完成!")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
