package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strings"
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
	fmt.Println("容器: postgres_easypeak")
	fmt.Println("====================================")

	// 1. 检查Docker容器状态
	fmt.Println("📋 检查Docker容器状态...")
	containerName := "postgres_easypeak"
	checkDockerContainer(containerName)

	// 2. 初始化配置
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     5432,
			User:     getEnv("DB_USER", "postgres"),
			Password: getEnv("DB_PASSWORD", "PostgresPassword"),
			DBName:   getEnv("DB_NAME", "easypeekdb"),
			SSLMode:  getEnv("DB_SSLMODE", "disable"),
		},
	}

	fmt.Printf("🔗 数据库连接: %s@%s:%d/%s\n",
		cfg.Database.User, cfg.Database.Host, cfg.Database.Port, cfg.Database.DBName)

	// 3. 初始化数据库连接
	fmt.Println("🔌 连接数据库...")
	err := database.Initialize(cfg)
	if err != nil {
		fmt.Printf("❌ 数据库连接失败: %v\n", err)
		fmt.Println("\n故障排除建议:")
		fmt.Println("1. 确保容器运行: docker start postgres_easypeak")
		fmt.Println("2. 确保数据库存在: docker exec postgres_easypeak psql -U postgres -c \"CREATE DATABASE easypeek;\"")
		fmt.Println("3. 运行迁移: migrate.bat migrations/simple_init.sql")
		log.Fatal("❌ 数据库连接失败:", err)
	}
	fmt.Println("✅ 数据库连接成功")

	// 4. 检查news表是否存在
	fmt.Println("📊 检查数据库表结构...")
	checkTableExists()

	// 5. 检查转换后的数据文件
	jsonFile := findDataFile()
	if jsonFile == "" {
		fmt.Println("\n❌ 找不到数据文件")
		fmt.Println("可用数据文件:")
		fmt.Println("1. converted_news_data.json (推荐)")
		fmt.Println("2. news_converted.json")
		fmt.Println("3. 请确保文件位于项目根目录下")
		os.Exit(1)
	}

	// 4. 读取并解析JSON数据
	fmt.Printf("📖 读取数据文件: %s\n", jsonFile)
	data, err := os.ReadFile(jsonFile)
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
	if errorCount > 0 {
		fmt.Printf("❌ 失败: %d 条\n", errorCount)
	}
	fmt.Printf("📊 总计: %d 条\n", len(convertedData.NewsItems))

	if successCount > 0 {
		// 8. 验证导入结果
		fmt.Println("\n🔍 验证导入结果...")
		var totalNews int64
		database.DB.Model(&models.News{}).Count(&totalNews)
		fmt.Printf("📰 数据库中总新闻数量: %d\n", totalNews)

		// 显示最新导入的新闻
		var latestNews []models.News
		database.DB.Order("created_at DESC").Limit(3).Find(&latestNews)

		fmt.Println("\n📰 最新导入的新闻:")
		for i, news := range latestNews {
			fmt.Printf("  %d. [%s] %s\n", i+1, news.Category, news.Title)
		}

		// 按分类统计
		fmt.Println("\n📊 分类统计:")
		var categoryStats []struct {
			Category string
			Count    int64
		}
		database.DB.Model(&models.News{}).
			Select("category, COUNT(*) as count").
			Group("category").
			Order("count DESC").
			Scan(&categoryStats)

		for _, stat := range categoryStats {
			fmt.Printf("  %s: %d条\n", stat.Category, stat.Count)
		}

		fmt.Println("\n✨ 导入成功！现在可以:")
		fmt.Println("1. 启动服务: go run cmd/main.go")
		fmt.Println("2. 查看数据: docker exec -it postgres_easypeak psql -U postgres -d easypeekdb")
	} else {
		fmt.Println("\n❌ 没有成功导入任何数据，请检查:")
		fmt.Println("1. 数据文件格式是否正确")
		fmt.Println("2. 数据库连接是否正常")
		fmt.Println("3. 表结构是否已创建")
	}
}

// convertToNewsModel 将JSON数据转换为models.News结构
func convertToNewsModel(item NewsItem) (models.News, error) {
	// 解析发布时间
	var publishedAt time.Time
	var err error

	// 尝试多种时间格式
	timeFormats := []string{
		"2006-01-02T15:04:05Z",
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05+08:00",
	}

	for _, format := range timeFormats {
		publishedAt, err = time.Parse(format, item.PublishedAt)
		if err == nil {
			break
		}
	}

	if err != nil {
		// 如果都失败了，使用当前时间
		fmt.Printf("⚠️ 解析时间 '%s' 失败，使用当前时间: %v\n", item.PublishedAt, err)
		publishedAt = time.Now()
	}

	// 转换SourceType - 直接使用字符串，让GORM处理类型转换
	sourceType := models.NewsTypeManual
	if item.SourceType == "rss" {
		sourceType = models.NewsTypeRSS
	}

	// 创建News对象
	news := models.News{
		Title:        truncateString(item.Title, 500),
		Content:      item.Content,
		Summary:      item.Summary,
		Description:  item.Description,
		Source:       truncateString(item.Source, 100),
		Category:     truncateString(item.Category, 100),
		PublishedAt:  publishedAt,
		CreatedBy:    item.CreatedBy,
		IsActive:     item.IsActive,
		SourceType:   sourceType,
		RSSSourceID:  item.RSSSourceID,
		Link:         truncateString(item.Link, 1000),
		GUID:         truncateString(item.GUID, 500),
		Author:       truncateString(item.Author, 100),
		ImageURL:     truncateString(item.ImageURL, 1000),
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

	// 设置默认值
	if news.Language == "" {
		news.Language = "zh"
	}
	if news.Status == "" {
		news.Status = "published"
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

// checkDockerContainer 检查指定容器状态
func checkDockerContainer(containerName string) {
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.Names}}")
	output, err := cmd.Output()

	if err != nil || strings.TrimSpace(string(output)) == "" {
		fmt.Printf("❌ 容器 %s 未运行\n", containerName)
		fmt.Println("启动建议:")
		fmt.Printf("  docker start %s\n", containerName)
		fmt.Println("  或运行: start-postgres-easypeak.bat")
		os.Exit(1)
	} else {
		fmt.Printf("✅ 容器 %s 正在运行\n", containerName)
	}
}

// checkTableExists 检查news表是否存在
func checkTableExists() {
	var tableExists bool
	err := database.DB.Raw("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'news')").Scan(&tableExists).Error
	if err != nil {
		fmt.Printf("❌ 检查表失败: %v\n", err)
		os.Exit(1)
	}

	if !tableExists {
		fmt.Println("❌ news表不存在")
		fmt.Println("请先运行数据库迁移:")
		fmt.Println("  migrate.bat migrations/simple_init.sql")
		os.Exit(1)
	}

	// 检查现有数据
	var count int64
	database.DB.Raw("SELECT COUNT(*) FROM news").Scan(&count)
	fmt.Printf("✅ news表存在，当前有 %d 条记录\n", count)
}

// findDataFile 查找可用的数据文件
func findDataFile() string {
	possibleFiles := []string{
		"converted_news_data.json",
		"news_converted.json",
		"localization_converted.json",
	}

	for _, file := range possibleFiles {
		if _, err := os.Stat(file); err == nil {
			fmt.Printf("📖 找到数据文件: %s\n", file)
			return file
		}
	}

	return ""
}
