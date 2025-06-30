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
	fmt.Println("🧪 测试EasyPeek新闻创建功能")
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

	err := database.Initialize(cfg)
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	fmt.Println("✅ 数据库连接成功")

	newsService := services.NewNewsService()

	// 2. 测试创建新闻
	fmt.Println("➕ 测试创建新闻...")
	createReq := &models.NewsCreateRequest{
		Title:    "测试新闻 - 数据库字段验证",
		Content:  "这是一条通过API创建的测试新闻，用于验证数据库字段匹配问题是否已解决。",
		Summary:  "测试新闻摘要",
		Source:   "EasyPeek测试",
		Category: "技术测试",
	}

	var testUserID uint = 1
	createdNews, err := newsService.CreateNews(createReq, testUserID)
	if err != nil {
		log.Printf("❌ 创建新闻失败: %v", err)
		return
	}

	fmt.Printf("✅ 成功创建新闻!\n")
	fmt.Printf("   ID: %d\n", createdNews.ID)
	fmt.Printf("   标题: %s\n", createdNews.Title)
	fmt.Printf("   分类: %s\n", createdNews.Category)
	fmt.Printf("   来源类型: %s\n", createdNews.SourceType)

	// 3. 验证创建的新闻
	fmt.Printf("\n🔍 验证创建的新闻 (ID: %d)...\n", createdNews.ID)
	retrievedNews, err := newsService.GetNewsByID(createdNews.ID)
	if err != nil {
		log.Printf("❌ 获取新闻失败: %v", err)
	} else {
		fmt.Printf("✅ 成功获取创建的新闻: %s\n", retrievedNews.Title)
	}

	// 4. 清理测试数据
	fmt.Printf("\n🗑️ 清理测试数据...\n")
	err = newsService.DeleteNews(createdNews.ID)
	if err != nil {
		log.Printf("❌ 删除测试新闻失败: %v", err)
	} else {
		fmt.Printf("✅ 测试新闻已删除\n")
	}

	fmt.Println("\n=====================================")
	fmt.Println("🎉 新闻创建功能测试完成!")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
