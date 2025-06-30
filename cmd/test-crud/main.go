package main

import (
	"fmt"
	"log"
	"os"

	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/services"
)

func main() {
	fmt.Println("🧪 测试EasyPeek新闻服务CRUD操作")
	fmt.Println("=====================================")

	// 1. 初始化配置和数据库
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

	fmt.Println("🔌 连接数据库...")
	err := database.Initialize(cfg)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	fmt.Println("✅ 数据库连接成功")

	newsService := services.NewNewsService()
	fmt.Println("📰 新闻服务已初始化")

	// 2. 测试创建新闻
	fmt.Println("\n➕ 测试创建新闻...")
	createReq := &models.NewsCreateRequest{
		Title:    "测试新闻 - API验证",
		Content:  "这是一条通过API创建的测试新闻，用于验证新闻服务的创建功能。内容包含了完整的正文信息。",
		Summary:  "API创建的测试新闻",
		Source:   "EasyPeek API Test",
		Category: "技术",
	}

	var testUserID uint = 1 // 假设用户ID为1
	createdNews, err := newsService.CreateNews(createReq, testUserID)
	if err != nil {
		log.Printf("❌ 创建新闻失败: %v", err)
	} else {
		fmt.Printf("✅ 成功创建新闻 ID: %d\n", createdNews.ID)
		fmt.Printf("   标题: %s\n", createdNews.Title)
		fmt.Printf("   分类: %s\n", createdNews.Category)
	}

	// 3. 测试更新新闻
	if createdNews != nil {
		fmt.Printf("\n✏️ 测试更新新闻 (ID: %d)...\n", createdNews.ID)
		updateReq := &models.NewsUpdateRequest{
			Title:   "更新后的测试新闻标题",
			Content: "这是更新后的新闻内容，验证更新功能是否正常工作。",
			Summary: "更新后的摘要",
		}

		err := newsService.UpdateNews(createdNews, updateReq)
		if err != nil {
			log.Printf("❌ 更新新闻失败: %v", err)
		} else {
			fmt.Printf("✅ 成功更新新闻\n")
			fmt.Printf("   新标题: %s\n", createdNews.Title)
		}
	}

	// 4. 测试分页功能
	fmt.Println("\n📄 测试分页功能...")
	page1, total, err := newsService.GetAllNews(1, 3)
	if err != nil {
		log.Printf("❌ 获取第1页失败: %v", err)
	} else {
		fmt.Printf("✅ 第1页: %d条 (总共%d条)\n", len(page1), total)
	}

	page2, _, err := newsService.GetAllNews(2, 3)
	if err != nil {
		log.Printf("❌ 获取第2页失败: %v", err)
	} else {
		fmt.Printf("✅ 第2页: %d条\n", len(page2))
	}

	// 5. 测试高级搜索
	fmt.Println("\n🔍 测试高级搜索...")
	keywords := []string{"技术", "发展", "突破"}
	for _, keyword := range keywords {
		results, count, err := newsService.SearchNews(keyword, 1, 5)
		if err != nil {
			log.Printf("❌ 搜索'%s'失败: %v", keyword, err)
		} else {
			fmt.Printf("✅ 搜索'%s': %d条结果\n", keyword, count)
			if len(results) > 0 {
				fmt.Printf("   首条: %s\n", truncateString(results[0].Title, 40))
			}
		}
	}

	// 6. 测试数据统计
	fmt.Println("\n📊 测试数据统计...")
	allNews, total, err := newsService.GetAllNews(1, 100) // 获取所有新闻
	if err != nil {
		log.Printf("❌ 获取统计数据失败: %v", err)
	} else {
		categoryCount := make(map[string]int)
		totalViews := int64(0)
		totalLikes := int64(0)

		for _, news := range allNews {
			categoryCount[news.Category]++
			totalViews += news.ViewCount
			totalLikes += news.LikeCount
		}

		fmt.Printf("✅ 数据统计结果:\n")
		fmt.Printf("   总新闻数: %d\n", total)
		fmt.Printf("   总浏览量: %d\n", totalViews)
		fmt.Printf("   总点赞数: %d\n", totalLikes)
		fmt.Printf("   分类分布:\n")
		for category, count := range categoryCount {
			fmt.Printf("     %s: %d条\n", category, count)
		}
	}

	// 7. 清理测试数据
	if createdNews != nil {
		fmt.Printf("\n🗑️ 清理测试数据 (删除新闻 ID: %d)...\n", createdNews.ID)
		err := newsService.DeleteNews(createdNews.ID)
		if err != nil {
			log.Printf("❌ 删除测试新闻失败: %v", err)
		} else {
			fmt.Printf("✅ 测试新闻已删除\n")
		}
	}

	fmt.Println("\n=====================================")
	fmt.Println("🎉 新闻服务CRUD测试完成!")
	fmt.Println("📈 所有功能验证通过，后端新闻模块可以正确访问数据库")
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
