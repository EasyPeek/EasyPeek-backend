package main

import (
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/api"
	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"github.com/EasyPeek/EasyPeek-backend/internal/database"
	"github.com/EasyPeek/EasyPeek-backend/internal/models"
	"github.com/EasyPeek/EasyPeek-backend/internal/scheduler"
	"github.com/EasyPeek/EasyPeek-backend/internal/services"
)

func main() {
	// load config
	cfg, err := config.LoadConfig("internal/config/config.yaml")
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	// 调试：检查API Key加载情况
	if len(cfg.AI.APIKey) > 0 {
		previewLen := 15
		if len(cfg.AI.APIKey) < previewLen {
			previewLen = len(cfg.AI.APIKey)
		}
		preview := cfg.AI.APIKey[:previewLen] + "..."
		log.Printf("🔍 Loaded API Key = %s (length: %d)", preview, len(cfg.AI.APIKey))
	} else {
		log.Printf("❌ API Key not loaded or empty")
	}
	log.Printf("🔍 Full AI config: Provider=%s, Model=%s, BaseURL=%s", cfg.AI.Provider, cfg.AI.Model, cfg.AI.BaseURL)

	// initialize database
	if err := database.Initialize(cfg); err != nil {
		log.Fatalf("Failed to initialize database: %v", err)
	}
	defer database.CloseDatabase()

	// execute database migration
	if err := database.Migrate(
		&models.User{},
		&models.News{},
		&models.Event{},
		&models.RSSSource{},
		&models.Comment{},
		&models.CommentLike{},
		&models.Message{},
		&models.Follow{},
		&models.NewsLike{},
		&models.AIAnalysis{}, // 添加AI分析表
	); err != nil {
		log.Fatalf("Failed to migrate database: %v", err)
	}

	// initialize seed data
	seedService := services.NewSeedService()

	if err := seedService.SeedCompleteData(); err != nil {
		log.Printf("Warning: Failed to seed initial data: %v", err)
	}

	// initialize RSS scheduler
	rssScheduler := scheduler.NewRSSScheduler()
	if err := rssScheduler.Start(); err != nil {
		log.Fatalf("Failed to start RSS scheduler: %v", err)
	}
	defer rssScheduler.Stop()

	// initialize AI event generation service with config from yaml
	aiEventConfig := &services.AIEventConfig{
		Provider:    cfg.AI.Provider,
		APIKey:      cfg.AI.APIKey,
		APIEndpoint: cfg.AI.BaseURL + "/chat/completions",
		Model:       cfg.AI.Model,
		MaxTokens:   cfg.AI.MaxTokens,
		Timeout:     cfg.AI.Timeout,
		Enabled:     true,
	}
	aiEventConfig.EventGeneration.Enabled = true
	aiEventConfig.EventGeneration.ConfidenceThreshold = 0.0
	aiEventConfig.EventGeneration.MinNewsCount = 2
	aiEventConfig.EventGeneration.TimeWindowHours = 24
	aiEventConfig.EventGeneration.MaxNewsLimit = 0 // 0表示不限制，处理所有新闻

	eventScheduler := scheduler.NewEventScheduler()
	if err := eventScheduler.Start(); err != nil {
		log.Fatalf("Failed to start Event scheduler: %v", err)
	}
	defer eventScheduler.Stop()

	aiEventService := services.NewAIEventServiceWithConfig(aiEventConfig)
	log.Println("AI事件服务配置：使用config.yaml中的AI配置，处理所有未关联的新闻")

	// 初始化新闻AI分析调度器
	newsAnalysisScheduler := scheduler.NewNewsAnalysisScheduler()
	if err := newsAnalysisScheduler.Start(); err != nil {
		log.Fatalf("Failed to start News Analysis scheduler: %v", err)
	}
	defer newsAnalysisScheduler.Stop()

	// 启动AI事件生成定时器
	aiEventTicker := time.NewTicker(30 * time.Minute) // 每30分钟执行一次
	defer aiEventTicker.Stop()

	// 启动AI事件生成的goroutine
	go func() {
		log.Println("AI事件生成服务已启动，每30分钟执行一次")

		// 立即执行一次事件生成（可选）
		if aiEventService.IsEnabled() {
			log.Println("执行初始AI事件生成...")
			if err := aiEventService.GenerateEventsFromNews(); err != nil {
				log.Printf("初始AI事件生成失败: %v", err)
			}
		}

		for range aiEventTicker.C {
			if aiEventService.IsEnabled() {
				log.Println("开始定时AI事件生成...")
				if err := aiEventService.GenerateEventsFromNews(); err != nil {
					log.Printf("定时AI事件生成失败: %v", err)
				} else {
					log.Println("定时AI事件生成完成")
				}
			} else {
				log.Println("AI事件生成服务未启用，跳过本次执行")
			}
		}
	}()

	// set up routes
	router := api.SetupRoutes()

	// create http server
	server := &http.Server{
		Addr:         ":8080",
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	go func() {
		sigChan := make(chan os.Signal, 1)
		signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
		<-sigChan

		log.Println("Shutting down server...")

		// 停止AI事件生成定时器
		aiEventTicker.Stop()
		log.Println("AI事件生成定时器已停止")

		// 停止新闻AI分析调度器
		newsAnalysisScheduler.Stop()
		log.Println("新闻AI分析调度器已停止")

		// 停止RSS调度器
		rssScheduler.Stop()

		// 停止事件调度器
		eventScheduler.Stop()

		if err := server.Close(); err != nil {
			log.Printf("Server shutdown error: %v", err)
		}
	}()

	log.Println("Server is starting on :8080")
	log.Println("RSS scheduler is running")
	log.Println("Event scheduler is running (stats update every 2 hours, hotness refresh every 4 hours)")
	log.Println("AI Event generation service is running (every 30 minutes)")
	log.Println("News AI Analysis scheduler is running (every 15 minutes)")

	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("Failed to start server: %v", err)
	}

	log.Println("Server stopped")
}
