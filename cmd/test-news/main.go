package main

import (
	"fmt"
	"log"
	"os"

	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/services"
)

func main() {
	fmt.Println("🧪 测试EasyPeek新闻服务模块")
	fmt.Println("=====================================")

	// 1. 初始化配置
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     5432,
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "password"),
			DBName:   getEnv("DB_NAME", "easypeek"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
	}

	// 2. 初始化数据库连接
	fmt.Println("🔌 连接数据库...")
	err := database.Initialize(cfg)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	fmt.Println("✅ 数据库连接成功")

	// 3. 创建新闻服务实例
	newsService := services.NewNewsService()
	fmt.Println("📰 新闻服务已初始化")

	// 4. 测试获取所有新闻
	fmt.Println("\n📊 测试获取所有新闻 (第1页，每页5条)...")
	newsList, total, err := newsService.GetAllNews(1, 5)
	if err != nil {
		log.Printf("❌ 获取新闻失败: %v", err)
	} else {
		fmt.Printf("✅ 成功获取新闻，总数: %d, 当前页: %d条\n", total, len(newsList))
		for i, news := range newsList {
			fmt.Printf("   %d. [%s] %s (浏览:%d, 点赞:%d)\n",
				i+1, news.Category, news.Title, news.ViewCount, news.LikeCount)
		}
	}

	// 5. 测试根据ID获取新闻
	if len(newsList) > 0 {
		fmt.Printf("\n🔍 测试根据ID获取新闻 (ID: %d)...\n", newsList[0].ID)
		news, err := newsService.GetNewsByID(newsList[0].ID)
		if err != nil {
			log.Printf("❌ 根据ID获取新闻失败: %v", err)
		} else {
			fmt.Printf("✅ 成功获取新闻: %s\n", news.Title)
			fmt.Printf("   内容预览: %s...\n", truncateString(news.Content, 100))
		}
	}

	// 6. 测试搜索新闻
	fmt.Println("\n🔎 测试搜索新闻 (关键词: '新')...")
	searchResults, searchTotal, err := newsService.SearchNews("新", 1, 3)
	if err != nil {
		log.Printf("❌ 搜索新闻失败: %v", err)
	} else {
		fmt.Printf("✅ 搜索结果: %d条, 显示前3条\n", searchTotal)
		for i, news := range searchResults {
			fmt.Printf("   %d. [%s] %s\n", i+1, news.Category, news.Title)
		}
	}

	// 7. 测试根据标题获取新闻
	if len(newsList) > 0 {
		testTitle := newsList[0].Title
		fmt.Printf("\n📝 测试根据标题获取新闻 (标题: '%s')...\n", truncateString(testTitle, 30))
		titleResults, err := newsService.GetNewsByTitle(testTitle)
		if err != nil {
			log.Printf("❌ 根据标题获取新闻失败: %v", err)
		} else {
			fmt.Printf("✅ 找到 %d 条匹配的新闻\n", len(titleResults))
		}
	}

	fmt.Println("\n=====================================")
	fmt.Println("🎉 新闻服务模块测试完成!")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
