package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run scripts/migrate.go <sql_file>")
		fmt.Println("Example: go run scripts/migrate.go migrations/001_create_news_tables.sql")
		os.Exit(1)
	}

	sqlFile := os.Args[1]

	// 使用环境变量或默认配置连接数据库
	dsn := "host=localhost user=postgres password=password dbname=easypeek port=5432 sslmode=disable TimeZone=Asia/Shanghai"

	// 如果有环境变量，使用环境变量
	if dbHost := os.Getenv("DB_HOST"); dbHost != "" {
		dsn = fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Shanghai",
			getEnv("DB_HOST", "localhost"),
			getEnv("DB_USER", "postgres"),
			getEnv("DB_PASSWORD", "password"),
			getEnv("DB_NAME", "easypeek"),
			getEnv("DB_PORT", "5432"),
			getEnv("DB_SSLMODE", "disable"),
		)
	}

	// 连接数据库
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// 获取原始SQL连接
	sqlDB, err := db.DB()
	if err != nil {
		log.Fatal("Failed to get SQL DB:", err)
	}
	defer sqlDB.Close()

	// 执行SQL文件
	err = executeSQLFile(db, sqlFile)
	if err != nil {
		log.Fatal("Failed to execute SQL file:", err)
	}

	fmt.Printf("Successfully executed %s\n", sqlFile)
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func executeSQLFile(db *gorm.DB, filename string) error {
	// 读取SQL文件
	content, err := os.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read SQL file: %v", err)
	}

	sqlContent := string(content)

	// 分割SQL语句（基于分号和换行）
	statements := strings.Split(sqlContent, ";")

	for i, statement := range statements {
		statement = strings.TrimSpace(statement)
		if statement == "" || strings.HasPrefix(statement, "--") {
			continue
		}

		fmt.Printf("Executing statement %d...\n", i+1)

		// 使用GORM执行原始SQL
		result := db.Exec(statement)
		if result.Error != nil {
			return fmt.Errorf("failed to execute statement %d: %v\nStatement: %s", i+1, result.Error, statement)
		}

		fmt.Printf("✓ Statement %d executed successfully (affected rows: %d)\n", i+1, result.RowsAffected)
	}

	return nil
}
