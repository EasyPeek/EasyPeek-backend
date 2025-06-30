package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
)

// ConvertedNewsData 对应转换后的JSON数据结构
type ConvertedNewsData struct {
	NewsItems      []NewsItem `json:"news_items"`
	TotalCount     int        `json:"total_count"`
	ConversionTime string     `json:"conversion_time"`
	SourceFile     string     `json:"source_file"`
}

// NewsItem 对应JSON中的新闻项
type NewsItem struct {
	Title        string  `json:"title"`
	Content      string  `json:"content"`
	Summary      string  `json:"summary"`
	Description  string  `json:"description"`
	Source       string  `json:"source"`
	Category     string  `json:"category"`
	PublishedAt  string  `json:"published_at"`
	CreatedBy    *uint   `json:"created_by"`
	IsActive     bool    `json:"is_active"`
	SourceType   string  `json:"source_type"`
	RSSSourceID  *uint   `json:"rss_source_id"`
	Link         string  `json:"link"`
	GUID         string  `json:"guid"`
	Author       string  `json:"author"`
	ImageURL     string  `json:"image_url"`
	Tags         string  `json:"tags"`
	Language     string  `json:"language"`
	ViewCount    int64   `json:"view_count"`
	LikeCount    int64   `json:"like_count"`
	CommentCount int64   `json:"comment_count"`
	ShareCount   int64   `json:"share_count"`
	HotnessScore float64 `json:"hotness_score"`
	Status       string  `json:"status"`
	IsProcessed  bool    `json:"is_processed"`
}

func main() {
	fmt.Println("🔄 EasyPeek 新闻数据导入工具")
	fmt.Println("====================================")

	// 1. 初始化配置 (使用与test-news相同的方式)
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
		log.Fatal("❌ 数据库连接失败:", err)
	}
	fmt.Println("✅ 数据库连接成功")

	// 3. 检查转换后的数据文件是否存在
	jsonFile := "converted_news_data.json"
	if _, err := os.Stat(jsonFile); os.IsNotExist(err) {
		log.Fatalf("❌ 找不到数据文件 %s\n请先运行: python scripts\\convert_localization_to_news.py", jsonFile)
	}

	// 4. 读取并解析JSON数据
	fmt.Printf("📖 读取数据文件: %s\n", jsonFile)
	data, err := ioutil.ReadFile(jsonFile)
	if err != nil {
		log.Fatalf("❌ 读取文件失败: %v", err)
	}

	var convertedData ConvertedNewsData
	err = json.Unmarshal(data, &convertedData)
	if err != nil {
		log.Fatalf("❌ 解析JSON失败: %v", err)
	}

	fmt.Printf("✅ 成功解析数据，共 %d 条新闻\n", len(convertedData.NewsItems))

	// 5. 确认导入
	fmt.Printf("\n准备导入 %d 条新闻数据到数据库\n", len(convertedData.NewsItems))
	fmt.Print("确认导入吗？(y/N): ")
	var confirm string
	fmt.Scanln(&confirm)
	if confirm != "y" && confirm != "yes" && confirm != "Y" && confirm != "YES" {
		fmt.Println("❌ 取消导入")
		return
	}

	// 6. 开始批量导入
	fmt.Println("\n🚀 开始导入数据...")
	startTime := time.Now()
	successCount := 0
	errorCount := 0

	for i, item := range convertedData.NewsItems {
		// 转换为models.News结构
		news, err := convertToNewsModel(item)
		if err != nil {
			log.Printf("❌ 转换第 %d 条数据失败: %v", i+1, err)
			errorCount++
			continue
		}

		// 使用GORM保存到数据库
		result := database.DB.Create(&news)
		if result.Error != nil {
			log.Printf("❌ 插入第 %d 条数据失败: %v", i+1, result.Error)
			errorCount++
		} else {
			successCount++
		}

		// 显示进度
		if (i+1)%10 == 0 || i+1 == len(convertedData.NewsItems) {
			fmt.Printf("📊 进度: %d/%d (%.1f%%)\n",
				i+1, len(convertedData.NewsItems),
				float64(i+1)/float64(len(convertedData.NewsItems))*100)
		}
	}

	endTime := time.Now()
	duration := endTime.Sub(startTime)

	// 7. 显示导入结果
	fmt.Println("\n====================================")
	fmt.Printf("🎉 导入完成！用时: %.2f 秒\n", duration.Seconds())
	fmt.Printf("✅ 成功导入: %d 条\n", successCount)
	fmt.Printf("❌ 失败: %d 条\n", errorCount)
	fmt.Printf("📊 总计: %d 条\n", len(convertedData.NewsItems))

	if successCount > 0 {
		// 8. 验证导入结果
		fmt.Println("\n🔍 验证导入结果...")
		var totalNews int64
		database.DB.Model(&models.News{}).Count(&totalNews)
		fmt.Printf("📰 数据库中总新闻数量: %d\n", totalNews)

		// 显示最新导入的几条新闻
		var recentNews []models.News
		database.DB.Order("created_at DESC").Limit(3).Find(&recentNews)
		fmt.Println("\n📋 最新导入的新闻:")
		for i, news := range recentNews {
			fmt.Printf("   %d. [%s] %s\n", i+1, news.Category, truncateString(news.Title, 50))
		}

		fmt.Println("\n✅ 数据导入成功！可以启动后端服务查看新闻数据。")
	} else {
		fmt.Println("\n❌ 没有成功导入任何数据，请检查错误信息。")
	}
}

// convertToNewsModel 将JSON数据转换为models.News结构
func convertToNewsModel(item NewsItem) (models.News, error) {
	// 解析发布时间
	publishedAt, err := time.Parse("2006-01-02 15:04:05", item.PublishedAt)
	if err != nil {
		return models.News{}, fmt.Errorf("解析发布时间失败: %v", err)
	}

	// 转换SourceType
	var sourceType models.NewsType
	if item.SourceType == "rss" {
		sourceType = models.NewsTypeRSS
	} else {
		sourceType = models.NewsTypeManual
	}

	// 创建News对象
	news := models.News{
		Title:        item.Title,
		Content:      item.Content,
		Summary:      item.Summary,
		Description:  item.Description,
		Source:       item.Source,
		Category:     item.Category,
		PublishedAt:  publishedAt,
		CreatedBy:    item.CreatedBy,
		IsActive:     item.IsActive,
		SourceType:   sourceType,
		RSSSourceID:  item.RSSSourceID,
		Link:         item.Link,
		GUID:         item.GUID,
		Author:       item.Author,
		ImageURL:     item.ImageURL,
		Tags:         item.Tags,
		Language:     item.Language,
		ViewCount:    item.ViewCount,
		LikeCount:    item.LikeCount,
		CommentCount: item.CommentCount,
		ShareCount:   item.ShareCount,
		HotnessScore: item.HotnessScore,
		Status:       item.Status,
		IsProcessed:  item.IsProcessed,
	}

	return news, nil
}

// getEnv 获取环境变量，如果不存在则返回默认值
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// truncateString 截断字符串
func truncateString(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
