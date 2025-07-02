package main

import (
	"log"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	// 数据库连接
	dsn := "host=localhost user=postgres password=PostgresPassword dbname=easypeekdb port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// 清理follows表
	if err := db.Exec("DELETE FROM follows").Error; err != nil {
		log.Fatal("Failed to clean follows table:", err)
	}

	log.Println("Successfully cleaned follows table")
}