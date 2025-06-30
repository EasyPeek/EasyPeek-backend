package main

import (
	"log"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"github.com/EasyPeek/EasyPeek-backend/internal/database"
)

// 事件与新闻关联表结构
type EventNewsRelation struct {
	ID           uint   `gorm:"primaryKey"`
	EventID      uint   `gorm:"not null"`
	NewsID       uint   `gorm:"not null"`
	RelationType string `gorm:"type:varchar(20);default:'related'"` // primary, related, background
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func main() {
	// 加载配置
	cfg, err := config.LoadConfig("internal/config/config.yaml")
	if err != nil {
		log.Fatalf("无法加载配置: %v", err)
	}

	// 初始化数据库
	if err := database.Initialize(cfg); err != nil {
		log.Fatalf("无法初始化数据库: %v", err)
	}
	defer database.CloseDatabase()

	db := database.GetDB()

	// 先检查是否有新闻数据
	var newsCount int64
	db.Table("news").Count(&newsCount)
	if newsCount == 0 {
		log.Println("警告: 新闻表中没有数据，请先导入新闻数据!")
		return
	}

	// 创建事件新闻关联表
	if err := db.AutoMigrate(&EventNewsRelation{}); err != nil {
		log.Fatalf("创建事件新闻关联表失败: %v", err)
	}

	// 查询所有新闻ID（限制10条，用于测试）
	var newsIDs []uint
	db.Table("news").Select("id").Order("hotness_score DESC").Limit(10).Pluck("id", &newsIDs)

	if len(newsIDs) == 0 {
		log.Println("警告: 未找到有效的新闻ID!")
		return
	}

	log.Printf("找到 %d 条新闻，准备创建事件关联", len(newsIDs))

	// 为每个事件关联3条新闻
	// 事件1: "2025年全国两会" - 关联政治类新闻
	relations := []EventNewsRelation{
		{EventID: 1, NewsID: newsIDs[0], RelationType: "primary"},
		{EventID: 1, NewsID: newsIDs[1], RelationType: "related"},
		{EventID: 1, NewsID: newsIDs[2], RelationType: "background"},

		{EventID: 2, NewsID: newsIDs[3], RelationType: "primary"},
		{EventID: 2, NewsID: newsIDs[4], RelationType: "related"},

		{EventID: 3, NewsID: newsIDs[5], RelationType: "primary"},
		{EventID: 3, NewsID: newsIDs[6], RelationType: "related"},

		{EventID: 4, NewsID: newsIDs[7], RelationType: "primary"},

		{EventID: 5, NewsID: newsIDs[8], RelationType: "primary"},
		{EventID: 5, NewsID: newsIDs[9], RelationType: "related"},
	}

	for i, relation := range relations {
		result := db.Create(&relation)
		if result.Error != nil {
			log.Printf("警告: 创建关联 #%d 失败: %v", i+1, result.Error)
			continue
		}
		log.Printf("创建关联成功: 事件ID %d - 新闻ID %d - 关系类型 %s", relation.EventID, relation.NewsID, relation.RelationType)
	}

	log.Println("事件新闻关联数据导入完成！")
}
