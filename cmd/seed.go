package main

import (
	"log"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/services"
)

func main() {
	// 初始化配置
	cfg, err := config.LoadConfig("internal/config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}
	// 初始化数据库连接
	database.Initialize(cfg)
	db := database.GetDB()

	// 创建种子数据服务
	seedService := services.NewSeedService()

	// 初始化默认数据（管理员账户等）
	if err := seedService.SeedDefaultData(); err != nil {
		log.Fatalf("Failed to seed default data: %v", err)
	}

	// 创建测试用户
	testUsers := []models.User{
		{
			Username: "testuser1",
			Email:    "test1@example.com",
			Password: "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", // password
			Role:     "user",
			Status:   "active",
		},
		{
			Username: "testuser2",
			Email:    "test2@example.com",
			Password: "$2a$10$92IXUNpkjO0rOQ5byMi.Ye4oKoEa3Ro9llC/.og/at2.uheWG/igi", // password
			Role:     "user",
			Status:   "active",
		},
	}

	for _, user := range testUsers {
		var existingUser models.User
		if err := db.Where("email = ?", user.Email).First(&existingUser).Error; err != nil {
			if err := db.Create(&user).Error; err != nil {
				log.Printf("Failed to create user %s: %v", user.Username, err)
			} else {
				log.Printf("Created test user: %s", user.Username)
			}
		} else {
			log.Printf("User %s already exists", user.Username)
		}
	}

	// RSS源不需要初始化，跳过创建

	// 创建测试关注数据
	var user1, user2 models.User
	db.Where("username = ?", "testuser1").First(&user1)
	db.Where("username = ?", "testuser2").First(&user2)

	if user1.ID > 0 && user2.ID > 0 {
		// 获取一些事件用于创建关注关系
		var events []models.Event
		db.Limit(3).Find(&events)

		if len(events) > 0 {
			// 创建事件关注数据
			testFollows := []models.Follow{
				{
					UserID:  user1.ID,
					EventID: events[0].ID,
				},
			}

			if len(events) > 1 {
				testFollows = append(testFollows, models.Follow{
					UserID:  user1.ID,
					EventID: events[1].ID,
				})
			}

			if len(events) > 2 {
				testFollows = append(testFollows, models.Follow{
					UserID:  user2.ID,
					EventID: events[2].ID,
				})
			}

			for _, follow := range testFollows {
				var existingFollow models.Follow
				if err := db.Where("user_id = ? AND event_id = ?", follow.UserID, follow.EventID).First(&existingFollow).Error; err != nil {
					if err := db.Create(&follow).Error; err != nil {
						log.Printf("Failed to create follow: %v", err)
					} else {
						log.Printf("Created follow: user %d follows event %d", follow.UserID, follow.EventID)
					}
				} else {
					log.Printf("Follow already exists: user %d follows event %d", follow.UserID, follow.EventID)
				}
			}
		} else {
			log.Printf("No events found, skipping follow creation")
		}

		// 创建测试消息数据
		testMessages := []models.Message{
			{
				UserID:    user1.ID,
				Type:      "news_update",
				Title:     "新闻更新通知",
				Content:   "您关注的科技分类有新的新闻更新",
				IsRead:    false,
				CreatedAt: time.Now(),
			},
			{
				UserID:    user2.ID,
				Type:      "event_like",
				Title:     "事件点赞通知",
				Content:   "有人点赞了您关注的事件",
				IsRead:    false,
				CreatedAt: time.Now(),
			},
			{
				UserID:    user1.ID,
				Type:      "system",
				Title:     "系统通知",
				Content:   "欢迎使用EasyPeek系统",
				IsRead:    true,
				CreatedAt: time.Now().Add(-24 * time.Hour),
			},
		}

		for _, message := range testMessages {
			if err := db.Create(&message).Error; err != nil {
				log.Printf("Failed to create message: %v", err)
			} else {
				log.Printf("Created message: %s for user %d", message.Title, message.UserID)
			}
		}
	}

	log.Println("Test data seeding completed!")
}
