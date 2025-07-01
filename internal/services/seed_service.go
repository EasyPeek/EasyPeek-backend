package services

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"gorm.io/gorm"
)

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

	// 检查是否已经有新闻数据，避免重复导入
	var count int64
	if err := s.db.Model(&models.News{}).Count(&count).Error; err != nil {
		return fmt.Errorf("failed to check existing news count: %w", err)
	}

	if count > 0 {
		log.Printf("数据库中已存在 %d 条新闻记录，跳过数据导入", count)
		return nil
	}

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

	// 在这里可以添加其他类型的数据导入，例如：
	// - 用户数据
	// - RSS源数据
	// - 事件数据等

	log.Println("所有种子数据初始化完成！")
	return nil
}
