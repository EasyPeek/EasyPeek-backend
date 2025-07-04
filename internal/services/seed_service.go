package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/utils"
	"gorm.io/gorm"
)

/*
SeedService 种子数据服务

主要功能：
1. 导入新闻数据从JSON文件
2. 创建默认管理员账户
3. 创建默认RSS源

使用方式：
  seedService := NewSeedService()
  seedService.SeedAllData()  // 导入数据
*/

type SeedService struct {
	db *gorm.DB
}

// NewSeedService 创建新的种子数据服务实例
func NewSeedService() *SeedService {
	return &SeedService{
		db: database.GetDB(),
	}
}

// NewsJSONData 定义JSON文件中的新闻数据结构
type NewsJSONData struct {
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

// SeedNewsFromJSON 从JSON文件导入新闻数据
func (s *SeedService) SeedNewsFromJSON(jsonFilePath string) error {
	log.Printf("开始从文件 %s 导入新闻数据...", jsonFilePath)

	// 检查数据库连接
	if s.db == nil {
		return fmt.Errorf("database connection not initialized")
	}

	// 检查现有新闻数据数量
	var count int64
	if err := s.db.Model(&models.News{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check existing news count: %w", err)
	}

	log.Printf("数据库中当前有 %d 条新闻记录，准备进行增量导入", count)

	// 读取JSON文件
	jsonData, err := os.ReadFile(jsonFilePath)
	if err != nil {
		return fmt.Errorf("failed to read JSON file: %w", err)
	}

	// 解析JSON数据
	var newsDataList []NewsJSONData
	if err := json.Unmarshal(jsonData, &newsDataList); err != nil {
		return fmt.Errorf("failed to parse JSON data: %w", err)
	}

	log.Printf("成功解析JSON文件，找到 %d 条新闻记录", len(newsDataList))

	// 批量插入数据
	var newsList []models.News
	importedCount := 0
	skippedCount := 0

	for i, newsData := range newsDataList {
		// 解析发布时间
		publishedAt, err := time.Parse("2006-01-02 15:04:05", newsData.PublishedAt)
		if err != nil {
			log.Printf("警告：解析第 %d 条记录的发布时间失败，使用当前时间: %v", i+1, err)
			publishedAt = time.Now()
		}

		// 检查是否已存在相同GUID或链接的记录
		var existingNews models.News
		err = s.db.Where("guid = ? OR link = ?", newsData.GUID, newsData.Link).First(&existingNews).Error
		if err == nil {
			skippedCount++
			log.Printf("跳过重复记录：%s", newsData.Title)
			continue
		} else if err != gorm.ErrRecordNotFound {
			log.Printf("检查重复记录时出错：%v", err)
			continue
		}

		// 转换SourceType
		var sourceType models.NewsType = models.NewsTypeManual
		if newsData.SourceType == "rss" {
			sourceType = models.NewsTypeRSS
		}

		// 创建新闻记录
		news := models.News{
			Title:        newsData.Title,
			Content:      newsData.Content,
			Summary:      newsData.Summary,
			Description:  newsData.Description,
			Source:       newsData.Source,
			Category:     newsData.Category,
			PublishedAt:  publishedAt,
			CreatedBy:    newsData.CreatedBy,
			IsActive:     newsData.IsActive,
			SourceType:   sourceType,
			RSSSourceID:  newsData.RSSSourceID,
			Link:         newsData.Link,
			GUID:         newsData.GUID,
			Author:       newsData.Author,
			ImageURL:     newsData.ImageURL,
			Tags:         newsData.Tags,
			Language:     newsData.Language,
			ViewCount:    newsData.ViewCount,
			LikeCount:    newsData.LikeCount,
			CommentCount: newsData.CommentCount,
			ShareCount:   newsData.ShareCount,
			HotnessScore: newsData.HotnessScore,
			Status:       newsData.Status,
			IsProcessed:  newsData.IsProcessed,
		}

		newsList = append(newsList, news)
		importedCount++

		// 每100条记录批量插入一次，避免单次事务过大
		if len(newsList) >= 100 {
			if err := s.batchInsertNews(newsList); err != nil {
				return fmt.Errorf("failed to batch insert news: %w", err)
			}
			newsList = []models.News{} // 清空切片
		}
	}

	// 插入剩余的记录
	if len(newsList) > 0 {
		if err := s.batchInsertNews(newsList); err != nil {
			return fmt.Errorf("failed to insert remaining news: %w", err)
		}
	}

	log.Printf("新闻数据导入完成！成功导入 %d 条记录，跳过 %d 条重复记录", importedCount, skippedCount)
	return nil
}

// batchInsertNews 批量插入新闻记录
func (s *SeedService) batchInsertNews(newsList []models.News) error {
	if len(newsList) == 0 {
		return nil
	}

	// 使用事务进行批量插入
	return s.db.Transaction(func(tx *gorm.DB) error {
		// CreateInBatches 可以进行分批插入，避免单次插入过多数据
		if err := tx.CreateInBatches(newsList, 50).Error; err != nil {
			return err
		}
		return nil
	})
}

// SeedAllData 导入所有初始化数据
func (s *SeedService) SeedAllData() error {
	log.Println("开始初始化种子数据...")

	// 导入新闻数据
	if err := s.SeedNewsFromJSON("data/new.json"); err != nil {
		return fmt.Errorf("failed to seed news data: %w", err)
	}

	if err := s.SeedInitialAdmin(); err != nil {
		return fmt.Errorf("failed to seed initial admin: %w", err)
	}

	// 可以在这里添加其他类型的数据导入，例如：
	// - 用户数据
	// - RSS源数据
	// - 其他初始化数据等

	log.Println("所有种子数据初始化完成！")
	return nil
}

// SeedInitialAdmin 创建初始管理员账户
func (s *SeedService) SeedInitialAdmin() error {
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	// 检查是否已经存在管理员账户
	var adminCount int64
	if err := s.db.Model(&models.User{}).Where("role = ?", "admin").Count(&adminCount).Error; err != nil {
		return err
	}

	// 如果已经存在管理员，不需要创建
	if adminCount > 0 {
		log.Println("Admin account already exists, skipping seed")
		return nil
	}

	// 从环境变量或默认值获取管理员信息
	adminEmail := os.Getenv("ADMIN_EMAIL")
	if adminEmail == "" {
		adminEmail = "admin@easypeek.com"
	}

	adminPassword := os.Getenv("ADMIN_PASSWORD")
	if adminPassword == "" {
		adminPassword = "admin123456"
	}

	adminUsername := os.Getenv("ADMIN_USERNAME")
	if adminUsername == "" {
		adminUsername = "admin"
	}

	// 验证输入
	if !utils.IsValidEmail(adminEmail) {
		return errors.New("invalid admin email format")
	}

	if !utils.IsValidPassword(adminPassword) {
		return errors.New("admin password must contain at least one letter and one number")
	}

	if !utils.IsValidUsername(adminUsername) {
		return errors.New("invalid admin username format")
	}

	// 检查邮箱和用户名是否已存在
	var existingUser models.User
	if err := s.db.Where("email = ? OR username = ?", adminEmail, adminUsername).First(&existingUser).Error; err == nil {
		return errors.New("admin email or username already exists")
	}

	// 创建管理员账户
	adminUser := &models.User{
		Username: adminUsername,
		Email:    adminEmail,
		Password: adminPassword, // 会被 BeforeCreate hook 自动加密
		Role:     "admin",
		Status:   "active",
	}

	if err := s.db.Create(adminUser).Error; err != nil {
		return err
	}

	log.Printf("Initial admin account created successfully:")
	log.Printf("- Username: %s", adminUsername)
	log.Printf("- Email: %s", adminEmail)
	log.Printf("- Password: %s", adminPassword)
	log.Println("Please change the default password after first login!")

	return nil
}

// SeedDefaultData 种子数据初始化
func (s *SeedService) SeedDefaultData() error {
	// 创建初始管理员
	if err := s.SeedInitialAdmin(); err != nil {
		return err
	}

	// 可以在这里添加其他默认数据的初始化
	// 例如：默认分类、默认RSS源等

	return nil
}

// SeedRSSources 创建默认RSS源（可选）
func (s *SeedService) SeedRSSources() error {
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	// 检查是否已经存在RSS源
	var rssCount int64
	if err := s.db.Model(&models.RSSSource{}).Count(&rssCount).Error; err != nil {
		return err
	}

	// 如果已经存在RSS源，不需要创建
	if rssCount > 0 {
		log.Println("RSS sources already exist, skipping seed")
		return nil
	}

	// 创建一些默认的RSS源
	defaultSources := []models.RSSSource{
		{
			Name:        "BBC News",
			URL:         "http://feeds.bbci.co.uk/news/rss.xml",
			Category:    "国际新闻",
			Language:    "en",
			IsActive:    true,
			Description: "BBC 新闻RSS源",
			Priority:    1,
			UpdateFreq:  60,
		},
		{
			Name:        "CNN Top Stories",
			URL:         "http://rss.cnn.com/rss/edition.rss",
			Category:    "国际新闻",
			Language:    "en",
			IsActive:    true,
			Description: "CNN 头条新闻RSS源",
			Priority:    1,
			UpdateFreq:  60,
		},
		{
			Name:        "TechCrunch",
			URL:         "https://techcrunch.com/feed/",
			Category:    "科技",
			Language:    "en",
			IsActive:    true,
			Description: "TechCrunch 科技新闻RSS源",
			Priority:    1,
			UpdateFreq:  60,
		},
		{
			Name:        "Hacker News",
			URL:         "https://hnrss.org/frontpage",
			Category:    "科技",
			Language:    "en",
			IsActive:    true,
			Description: "Hacker News 前端页面RSS源",
			Priority:    1,
			UpdateFreq:  60,
		},
		{
			Name:        "Reuters World News",
			URL:         "https://feeds.reuters.com/reuters/worldNews",
			Category:    "国际新闻",
			Language:    "en",
			IsActive:    true,
			Description: "路透社世界新闻RSS源",
			Priority:    1,
			UpdateFreq:  60,
		},
	}

	for _, source := range defaultSources {
		if err := s.db.Create(&source).Error; err != nil {
			log.Printf("Failed to create RSS source %s: %v", source.Name, err)
		} else {
			log.Printf("Created default RSS source: %s", source.Name)
		}
	}

	return nil
}

// SeedCompleteData 完整的数据种子化
func (s *SeedService) SeedCompleteData() error {
	log.Println("开始完整的数据种子化...")

	// 1. 导入基础数据
	if err := s.SeedAllData(); err != nil {
		return fmt.Errorf("基础数据导入失败: %w", err)
	}

	// 2. 创建默认数据
	if err := s.SeedDefaultData(); err != nil {
		return fmt.Errorf("默认数据创建失败: %w", err)
	}

	// 3. 创建RSS源
	if err := s.SeedRSSources(); err != nil {
		return fmt.Errorf("RSS源创建失败: %w", err)
	}

	log.Println("完整的数据种子化完成！")
	return nil
}

/*
使用示例：

// 1. 基本使用
seedService := NewSeedService()
err := seedService.SeedAllData() // 导入新闻和基础数据

// 2. 仅导入新闻数据
seedService := NewSeedService()
err := seedService.SeedNewsFromJSON("data/news.json")

// 3. 完整初始化
seedService := NewSeedService()
err := seedService.SeedCompleteData()

注意事项：
1. 所有数据导入都会进行去重检查
2. 管理员账户信息可以通过环境变量配置
3. 支持批量插入提高性能
4. 所有操作都有详细的日志记录
*/
