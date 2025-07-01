package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
)

// ImportedEventData 导入的事件数据结构
type ImportedEventData struct {
	Events        []EventImport `json:"events"`
	TotalCount    int           `json:"total_count"`
	GeneratedTime string        `json:"generated_time"`
}

// EventImport 要导入的事件结构
type EventImport struct {
	Title        string    `json:"title"`
	Description  string    `json:"description"`
	Content      string    `json:"content"`
	StartTime    time.Time `json:"start_time"`
	EndTime      time.Time `json:"end_time"`
	Location     string    `json:"location"`
	Status       string    `json:"status"`
	Category     string    `json:"category"`
	Tags         string    `json:"tags"`
	Source       string    `json:"source"`
	RelatedLinks string    `json:"related_links"`
	ViewCount    int64     `json:"view_count"`
	LikeCount    int64     `json:"like_count"`
	CommentCount int64     `json:"comment_count"`
	ShareCount   int64     `json:"share_count"`
	HotnessScore float64   `json:"hotness_score"`
	CreatedBy    uint      `json:"created_by"`
	NewsIDs      []uint    `json:"news_ids"`
}

func main() {
	startTime := time.Now()
	fmt.Println("🔄 EasyPeek 自动生成事件导入工具")
	fmt.Println("====================================")

	// 加载配置
	cfg, err := config.LoadConfig("internal/config/config.yaml")
	if err != nil {
		log.Fatalf("❌ 无法加载配置: %v", err)
	}

	// 初始化数据库
	if err := database.Initialize(cfg); err != nil {
		log.Fatalf("❌ 无法初始化数据库: %v", err)
	}
	defer database.CloseDatabase()

	db := database.GetDB()

	// 1. 查找最新的事件JSON文件
	eventFile := findLatestEventFile()
	if eventFile == "" {
		log.Fatalf("❌ 找不到事件数据文件，请先运行 generate-events-from-news 脚本")
	}
	fmt.Printf("✅ 找到事件数据文件: %s\n", eventFile)

	// 2. 读取JSON文件
	jsonData, err := os.ReadFile(eventFile)
	if err != nil {
		log.Fatalf("❌ 读取文件失败: %v", err)
	}

	// 3. 解析JSON数据
	var importData ImportedEventData
	if err := json.Unmarshal(jsonData, &importData); err != nil {
		log.Fatalf("❌ 解析JSON失败: %v", err)
	}
	fmt.Printf("✅ 成功解析 %d 个事件\n", len(importData.Events))

	// 4. 导入事件数据
	fmt.Println("🔄 开始导入事件数据...")
	importCount := 0
	failCount := 0

	// 开始事务
	tx := db.Begin()
	if tx.Error != nil {
		log.Fatalf("❌ 开始事务失败: %v", tx.Error)
	}

	for i, eventImport := range importData.Events {
		// 创建事件对象
		event := models.Event{
			Title:        eventImport.Title,
			Description:  eventImport.Description,
			Content:      eventImport.Content,
			StartTime:    eventImport.StartTime,
			EndTime:      eventImport.EndTime,
			Location:     eventImport.Location,
			Status:       eventImport.Status,
			Category:     eventImport.Category,
			Tags:         eventImport.Tags,
			Source:       eventImport.Source,
			RelatedLinks: eventImport.RelatedLinks,
			ViewCount:    eventImport.ViewCount,
			LikeCount:    eventImport.LikeCount,
			CommentCount: eventImport.CommentCount,
			ShareCount:   eventImport.ShareCount,
			HotnessScore: eventImport.HotnessScore,
			CreatedBy:    eventImport.CreatedBy,
		}

		// 保存事件到数据库
		if err := tx.Create(&event).Error; err != nil {
			log.Printf("❌ 保存事件 #%d [%s] 失败: %v", i+1, event.Title, err)
			failCount++
			continue
		}

		// 更新相关新闻的事件关联
		if len(eventImport.NewsIDs) > 0 {
			if err := tx.Model(&models.News{}).Where("id IN ?", eventImport.NewsIDs).
				Update("belonged_event_id", event.ID).Error; err != nil {
				log.Printf("⚠️ 更新关联新闻失败 (事件 #%d): %v", event.ID, err)
			} else {
				log.Printf("✅ 为事件 #%d [%s] 关联了 %d 条新闻",
					event.ID, event.Title, len(eventImport.NewsIDs))
			}
		}

		importCount++

		// 每10个事件报告一次进度
		if importCount%10 == 0 || importCount == len(importData.Events) {
			fmt.Printf("🔄 进度: %d/%d (%.1f%%)\n",
				importCount, len(importData.Events),
				float64(importCount)/float64(len(importData.Events))*100)
		}
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		log.Fatalf("❌ 提交事务失败: %v", err)
	}

	elapsed := time.Since(startTime)
	fmt.Println("====================================")
	fmt.Printf("✅ 操作完成! 成功导入 %d/%d 个事件 (耗时: %.2f秒)\n",
		importCount, len(importData.Events), elapsed.Seconds())

	if failCount > 0 {
		fmt.Printf("⚠️ %d 个事件导入失败\n", failCount)
	}
}

// findLatestEventFile 查找最新的事件JSON文件
func findLatestEventFile() string {
	exportDir := "exports"
	pattern := filepath.Join(exportDir, "generated_events_*.json")

	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return ""
	}

	// 按修改时间排序，找出最新的文件
	var latestFile string
	var latestTime time.Time

	for _, file := range matches {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if latestFile == "" || info.ModTime().After(latestTime) {
			latestFile = file
			latestTime = info.ModTime()
		}
	}

	return latestFile
}
