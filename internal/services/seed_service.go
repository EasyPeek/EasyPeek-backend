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

main functions:
1. 从JSON文件中保存基础新闻
2. 创建管理员账户
3. 创建实例用户
4. 创建RSS源

use example:
  seedService := NewSeedService()
  seedService.SeedAllData()
*/

type SeedService struct {
	db *gorm.DB
}

// create a new seed service instance
func NewSeedService() *SeedService {
	return &SeedService{
		db: database.GetDB(),
	}
}

// import all initial data
func (s *SeedService) SeedAllData() error {
	log.Println("start to seed all data...")

	// import news data
	if err := s.SeedNewsFromJSON("data/new.json"); err != nil {
		return fmt.Errorf("failed to seed news data: %w", err)
	}

	if err := s.SeedInitialAdmin(); err != nil {
		return fmt.Errorf("failed to seed initial admin: %w", err)
	}

	// 导入示例用户
	if err := s.SeedExampleUsers(); err != nil {
		return fmt.Errorf("failed to seed example users: %w", err)
	}

	// 导入默认RSS源
	if err := s.SeedRSSources(); err != nil {
		return fmt.Errorf("failed to seed RSS sources: %w", err)
	}

	log.Println("all seed data initialized!")
	return nil
}

// NewsJSONData define the news data structure in the JSON file
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

// SeedNewsFromJSON import news data from a JSON file
func (s *SeedService) SeedNewsFromJSON(jsonFilePath string) error {
	log.Printf("start to import news data from file %s...", jsonFilePath)

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

	// 创建一些默认的RSS源
	defaultSources := []models.RSSSource{
		{
			Name:        "澎湃新闻",
			URL:         "https://feedx.net/rss/thepaper.xml",
			Category:    "综合新闻",
			Language:    "zh",
			IsActive:    true,
			Description: "澎湃新闻全文RSS源，提供综合新闻资讯",
			Priority:    1,
			UpdateFreq:  60,
		},
		{

			Name:        "光明日报",
			URL:         "https://feedx.net/rss/guangmingribao.xml",
			Category:    "时政新闻",
			Language:    "zh",
			IsActive:    true,
			Description: "光明日报全文RSS源，提供权威时政和文化新闻",

			Priority:   1,
			UpdateFreq: 60,
		},
		{

			Name:        "新华每日电讯",
			URL:         "https://feedx.net/rss/mrdx.xml",
			Category:    "时政新闻",
			Language:    "zh",
			IsActive:    true,
			Description: "新华每日电讯全文RSS源，提供权威时政和社会新闻",
			Priority:    1,
			UpdateFreq:  60,
		},
		{

			Name:        "经济日报",
			URL:         "https://feedx.net/rss/jingjiribao.xml",
			Category:    "财经新闻",
			Language:    "zh",
			IsActive:    true,
			Description: "经济日报全文RSS源，提供权威财经和经济政策新闻",

			Priority:   1,
			UpdateFreq: 60,
		},
		{

			Name:        "南方周末",
			URL:         "https://feedx.net/rss/infzm.xml",
			Category:    "时政新闻",
			Language:    "zh",
			IsActive:    true,
			Description: "南方周末全文RSS源，提供权威时政和社会新闻",
			Priority:    1,
			UpdateFreq:  60,
		},
		{
			Name:        "凤凰军事",
			URL:         "https://feedx.net/rss/ifengmil.xml",
			Category:    "军事新闻",
			Language:    "zh",
			IsActive:    true,
			Description: "凤凰军事全文RSS源，提供权威时政和军事新闻",
			Priority:    1,
			UpdateFreq:  60,
		},
		{
			Name:        "3dmgame",
			URL:         "https://feedx.net/rss/3dmgame.xml",
			Category:    "游戏新闻",
			Language:    "zh",
			IsActive:    true,
			Description: "3dmgame全文RSS源，提供权威游戏新闻",
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

// SeedExampleUsers 创建示例用户
func (s *SeedService) SeedExampleUsers() error {
	if s.db == nil {
		return errors.New("database connection not initialized")
	}

	log.Println("开始创建示例用户...")

	// 检查是否已经存在示例用户
	var userCount int64
	if err := s.db.Model(&models.User{}).Where("role = ?", "user").Count(&userCount).Error; err != nil {
		return fmt.Errorf("failed to check existing users count: %w", err)
	}

	log.Printf("数据库中当前有 %d 个普通用户", userCount)

	// 定义示例用户数据
	exampleUsers := []models.User{
		{
			Username:  "zhangsan",
			Email:     "zhangsan@example.com",
			Password:  "password123",
			Avatar:    "https://api.dicebear.com/7.x/avataaars/svg?seed=zhangsan",
			Phone:     "13800138001",
			Location:  "北京市",
			Bio:       "热爱科技新闻的用户，关注AI和互联网发展趋势",
			Interests: `["科技", "人工智能", "互联网", "创业"]`,
			Role:      "user",
			Status:    "active",
		},
		{
			Username:  "lisi",
			Email:     "lisi@example.com",
			Password:  "password123",
			Avatar:    "https://api.dicebear.com/7.x/avataaars/svg?seed=lisi",
			Phone:     "13800138002",
			Location:  "上海市",
			Bio:       "财经分析师，专注投资理财和市场动态",
			Interests: `["财经", "投资", "股票", "基金"]`,
			Role:      "user",
			Status:    "active",
		},
		{
			Username:  "wangwu",
			Email:     "wangwu@example.com",
			Password:  "password123",
			Avatar:    "https://api.dicebear.com/7.x/avataaars/svg?seed=wangwu",
			Phone:     "13800138003",
			Location:  "广州市",
			Bio:       "体育爱好者，关注国内外体育赛事和健身资讯",
			Interests: `["体育", "足球", "篮球", "健身"]`,
			Role:      "user",
			Status:    "active",
		},
		{
			Username:  "zhaoliu",
			Email:     "zhaoliu@example.com",
			Password:  "password123",
			Avatar:    "https://api.dicebear.com/7.x/avataaars/svg?seed=zhaoliu",
			Phone:     "13800138004",
			Location:  "深圳市",
			Bio:       "娱乐达人，热衷明星动态和影视资讯",
			Interests: `["娱乐", "电影", "音乐", "明星"]`,
			Role:      "user",
			Status:    "active",
		},
		{
			Username:  "qianqi",
			Email:     "qianqi@example.com",
			Password:  "password123",
			Avatar:    "https://api.dicebear.com/7.x/avataaars/svg?seed=qianqi",
			Phone:     "13800138005",
			Location:  "杭州市",
			Bio:       "教育工作者，关注教育政策和学术研究",
			Interests: `["教育", "学术", "政策", "科研"]`,
			Role:      "user",
			Status:    "active",
		},
		{
			Username:  "sunba",
			Email:     "sunba@example.com",
			Password:  "password123",
			Avatar:    "https://api.dicebear.com/7.x/avataaars/svg?seed=sunba",
			Phone:     "13800138006",
			Location:  "成都市",
			Bio:       "旅游博主，分享各地美食和旅行攻略",
			Interests: `["旅游", "美食", "摄影", "文化"]`,
			Role:      "user",
			Status:    "active",
		},
		{
			Username:  "zhoujiu",
			Email:     "zhoujiu@example.com",
			Password:  "password123",
			Avatar:    "https://api.dicebear.com/7.x/avataaars/svg?seed=zhoujiu",
			Phone:     "13800138007",
			Location:  "西安市",
			Bio:       "游戏玩家，关注游戏行业动态和电竞赛事",
			Interests: `["游戏", "电竞", "动漫", "科技"]`,
			Role:      "user",
			Status:    "active",
		},
		{
			Username:  "wuzshi",
			Email:     "wushi@example.com",
			Password:  "password123",
			Avatar:    "https://api.dicebear.com/7.x/avataaars/svg?seed=wushi",
			Phone:     "13800138008",
			Location:  "南京市",
			Bio:       "健康生活倡导者，关注医疗健康和养生资讯",
			Interests: `["健康", "医疗", "养生", "运动"]`,
			Role:      "user",
			Status:    "active",
		},
	}

	// 批量创建示例用户
	createdCount := 0
	skippedCount := 0

	for _, user := range exampleUsers {
		// 检查用户名和邮箱是否已存在
		var existingUser models.User
		err := s.db.Where("username = ? OR email = ?", user.Username, user.Email).First(&existingUser).Error
		if err == nil {
			skippedCount++
			log.Printf("跳过重复用户：%s", user.Username)
			continue
		} else if err != gorm.ErrRecordNotFound {
			log.Printf("检查用户时出错：%v", err)
			continue
		}

		// 验证用户数据
		if !utils.IsValidEmail(user.Email) {
			log.Printf("用户 %s 邮箱格式无效，跳过", user.Username)
			continue
		}

		if !utils.IsValidPassword(user.Password) {
			log.Printf("用户 %s 密码格式无效，跳过", user.Username)
			continue
		}

		if !utils.IsValidUsername(user.Username) {
			log.Printf("用户 %s 用户名格式无效，跳过", user.Username)
			continue
		}

		// 创建用户
		if err := s.db.Create(&user).Error; err != nil {
			log.Printf("创建用户 %s 失败：%v", user.Username, err)
			continue
		}

		createdCount++
		log.Printf("成功创建示例用户：%s (%s)", user.Username, user.Email)
	}

	log.Printf("示例用户创建完成！成功创建 %d 个用户，跳过 %d 个重复用户", createdCount, skippedCount)
	return nil
}

/*
使用示例：

// 1. 基本使用
seedService := NewSeedService()
err := seedService.SeedAllData() // 导入新闻、管理员和示例用户

// 2. 仅导入新闻数据
seedService := NewSeedService()
err := seedService.SeedNewsFromJSON("data/news.json")

// 3. 仅创建示例用户
seedService := NewSeedService()
err := seedService.SeedExampleUsers()

// 4. 完整初始化
seedService := NewSeedService()
err := seedService.SeedCompleteData()

注意事项：
1. 所有数据导入都会进行去重检查
2. 管理员账户信息可以通过环境变量配置
3. 示例用户包含多种兴趣类型和地理分布
4. 支持批量插入提高性能
5. 所有操作都有详细的日志记录
*/
