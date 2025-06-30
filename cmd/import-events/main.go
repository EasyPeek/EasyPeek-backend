package main

import (
	"fmt"
	"log"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"gorm.io/gorm"
)

// 测试数据
var testEvents = []models.Event{
	{
		Title:        "2025年全国两会",
		Description:  "中国人民政治协商会议第十四届全国委员会第三次会议和中华人民共和国第十四届全国人民代表大会第三次会议",
		Category:     "政治",
		StartTime:    time.Date(2025, 3, 4, 0, 0, 0, 0, time.Local),
		EndTime:      time.Date(2025, 3, 15, 0, 0, 0, 0, time.Local),
		Location:     "北京",
		Status:       "已结束",
		CreatedBy:    1, // 系统默认管理员ID，需确保存在
		Tags:         `["政治", "两会", "国家事务"]`,
		Source:       "央视新闻",
		ViewCount:    10000,
		LikeCount:    500,
		CommentCount: 300,
		ShareCount:   800,
		HotnessScore: 3000,
	},
	{
		Title:        "2025世界人工智能大会",
		Description:  "全球AI技术盛会，展示最新AI成果与技术发展趋势",
		Category:     "科技",
		StartTime:    time.Date(2025, 7, 10, 9, 0, 0, 0, time.Local),
		EndTime:      time.Date(2025, 7, 13, 18, 0, 0, 0, time.Local),
		Location:     "上海",
		Status:       "筹备中",
		CreatedBy:    1,
		Tags:         `["科技", "AI", "创新"]`,
		Source:       "上海日报",
		ViewCount:    5000,
		LikeCount:    300,
		CommentCount: 120,
		ShareCount:   450,
		HotnessScore: 1500,
	},
	{
		Title:        "第八届中国国际进口博览会",
		Description:  "全球贸易盛会，促进国际贸易与经济合作",
		Category:     "经济",
		StartTime:    time.Date(2025, 11, 5, 0, 0, 0, 0, time.Local),
		EndTime:      time.Date(2025, 11, 10, 0, 0, 0, 0, time.Local),
		Location:     "上海",
		Status:       "计划中",
		CreatedBy:    1,
		Tags:         `["经济", "贸易", "国际"]`,
		Source:       "新华社",
		ViewCount:    3000,
		LikeCount:    150,
		CommentCount: 80,
		ShareCount:   200,
		HotnessScore: 900,
	},
	{
		Title:        "2026年冬季奥运会倒计时一周年活动",
		Description:  "迎接米兰-科尔蒂纳丹佩佐冬奥会倒计时一周年系列活动",
		Category:     "体育",
		StartTime:    time.Date(2025, 2, 6, 0, 0, 0, 0, time.Local),
		EndTime:      time.Date(2025, 2, 6, 23, 59, 59, 0, time.Local),
		Location:     "北京",
		Status:       "已结束",
		CreatedBy:    1,
		Tags:         `["体育", "奥运", "国际"]`,
		Source:       "人民日报",
		ViewCount:    7000,
		LikeCount:    400,
		CommentCount: 250,
		ShareCount:   600,
		HotnessScore: 2000,
	},
	{
		Title:        "世界经济论坛2025年年会",
		Description:  "全球政商学界领袖共商世界经济与社会发展大计",
		Category:     "经济",
		StartTime:    time.Date(2025, 1, 15, 0, 0, 0, 0, time.Local),
		EndTime:      time.Date(2025, 1, 19, 0, 0, 0, 0, time.Local),
		Location:     "瑞士达沃斯",
		Status:       "已结束",
		CreatedBy:    1,
		Tags:         `["经济", "政治", "国际"]`,
		Source:       "财经日报",
		ViewCount:    8000,
		LikeCount:    350,
		CommentCount: 420,
		ShareCount:   550,
		HotnessScore: 2300,
	},
}

// 创建默认管理员用户
func createDefaultAdminIfNotExists(db *gorm.DB) error {
	var count int64
	db.Model(&models.User{}).Count(&count)

	if count == 0 {
		// 创建默认管理员用户
		admin := models.User{
			Username: "admin",
			Email:    "admin@easypeek.com",
			Password: "$2a$10$Rk9K4BsXxQG0E6klHOHAG.k16Vd7ZFUvF0K1Z1ZVGfTkd2rCVUeEG", // 加密后的 "admin123"
			Role:     "admin",
			Status:   "active",
		}

		result := db.Create(&admin)
		if result.Error != nil {
			return fmt.Errorf("创建管理员用户失败: %w", result.Error)
		}
		log.Println("创建默认管理员用户成功，ID:", admin.ID)
	}

	return nil
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

	// 创建默认管理员用户
	if err := createDefaultAdminIfNotExists(db); err != nil {
		log.Fatalf("创建管理员用户失败: %v", err)
	}

	// 导入事件数据
	for i, event := range testEvents {
		result := db.Create(&event)
		if result.Error != nil {
			log.Fatalf("导入事件 #%d 失败: %v", i+1, result.Error)
		}
		log.Printf("导入事件 #%d 成功: %s (ID: %d)", i+1, event.Title, event.ID)
	}

	log.Println("事件数据导入完成！")
}
