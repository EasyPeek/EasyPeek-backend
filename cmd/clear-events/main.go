package main

import (
	"fmt"
	"log"
	"os"
	"time"

	"github.com/EasyPeek/EasyPeek-backend/internal/config"
	"github.com/EasyPeek/EasyPeek-backend/internal/database"
)

func main() {
	startTime := time.Now()
	fmt.Println("🔄 EasyPeek 事件数据清空工具")
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

	// 确认操作
	fmt.Println("⚠️ 警告: 此操作将清空所有事件数据及关联关系")
	fmt.Println("您确定要继续吗? (y/n):")

	var confirm string
	fmt.Scanln(&confirm)

	if confirm != "y" && confirm != "Y" {
		fmt.Println("❌ 操作已取消")
		os.Exit(0)
	}

	fmt.Println("🔄 正在清空事件数据...")

	// 开始事务
	tx := db.Begin()
	if tx.Error != nil {
		log.Fatalf("❌ 开始事务失败: %v", tx.Error)
	}

	// 1. 先清除news表中的事件关联 (belonged_event_id)
	if err := tx.Exec("UPDATE news SET belonged_event_id = NULL").Error; err != nil {
		tx.Rollback()
		log.Fatalf("❌ 清除新闻事件关联失败: %v", err)
	}
	fmt.Println("✅ 已清除所有新闻的事件关联")

	// 2. 清空events表
	if err := tx.Exec("DELETE FROM events").Error; err != nil {
		tx.Rollback()
		log.Fatalf("❌ 清空事件表失败: %v", err)
	}
	fmt.Println("✅ 已清空events表")

	// 3. 重置events表的自增ID
	if err := tx.Exec("ALTER SEQUENCE events_id_seq RESTART WITH 1").Error; err != nil {
		tx.Rollback()
		log.Printf("⚠️ 重置事件表ID序列失败 (非致命错误): %v", err)
	} else {
		fmt.Println("✅ 已重置events表ID序列")
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		log.Fatalf("❌ 提交事务失败: %v", err)
	}

	elapsed := time.Since(startTime)
	fmt.Println("====================================")
	fmt.Printf("✅ 操作完成! 所有事件数据已清空 (耗时: %.2f秒)\n", elapsed.Seconds())
	fmt.Println("现在您可以运行 generate-events-from-news 脚本生成新的事件数据")
}
