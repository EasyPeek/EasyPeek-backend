package main

import (
	"fmt"
	"log"
	"os"
	"os/exec"
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

	fmt.Println("🔧 EasyPeek 数据库迁移工具")
	fmt.Println("============================")

	// 检查Docker容器状态
	fmt.Println("📋 检查Docker容器状态...")
	checkDockerContainer()

	// 构建DSN
	dsn := buildDSN()
	fmt.Printf("🔌 连接字符串: %s\n", maskDSN(dsn))

	// 连接数据库
	fmt.Println("🔌 连接数据库...")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("❌ 数据库连接失败: %v\n", err)
		showTroubleshootingTips()
		log.Fatal("Failed to connect to database:", err)
	}
	fmt.Println("✅ 数据库连接成功")

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

// 检查Docker容器状态
func checkDockerContainer() {
	containerName := "postgres_easypeak"

	// 检查容器是否运行
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.Names}}")
	output, err := cmd.Output()

	if err != nil || strings.TrimSpace(string(output)) == "" {
		fmt.Printf("⚠️ 容器 %s 未运行\n", containerName)
		fmt.Println("启动建议:")
		fmt.Printf("  docker start %s\n", containerName)
		fmt.Println("  或检查容器是否存在: docker ps -a | grep postgres")
	} else {
		fmt.Printf("✅ 容器 %s 正在运行\n", containerName)
	}
}

// 构建数据库连接字符串
func buildDSN() string {
	// 默认配置
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

	return dsn
}

// 隐藏密码的连接字符串
func maskDSN(dsn string) string {
	// 隐藏密码部分
	parts := strings.Split(dsn, " ")
	for i, part := range parts {
		if strings.HasPrefix(part, "password=") {
			parts[i] = "password=***"
		}
	}
	return strings.Join(parts, " ")
}

// 显示故障排除提示
func showTroubleshootingTips() {
	fmt.Println("\n🛠️ 故障排除建议:")
	fmt.Println("1. 检查Docker容器状态:")
	fmt.Println("   docker ps | grep postgres_easypeak")
	fmt.Println("2. 启动容器:")
	fmt.Println("   docker start postgres_easypeak")
	fmt.Println("3. 查看容器日志:")
	fmt.Println("   docker logs postgres_easypeak")
	fmt.Println("4. 检查端口映射:")
	fmt.Println("   docker port postgres_easypeak")
	fmt.Println("5. 如果容器不存在，创建新容器:")
	fmt.Println("   docker run -d --name postgres_easypeak -e POSTGRES_PASSWORD=password -p 5432:5432 postgres:15")
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
	fmt.Printf("📄 文件大小: %d bytes\n", len(sqlContent))

	// 改进的SQL语句分割逻辑
	statements := splitSQLStatements(sqlContent)

	fmt.Printf("📊 找到 %d 个SQL语句\n", len(statements))

	for i, statement := range statements {
		statement = strings.TrimSpace(statement)
		if statement == "" {
			continue
		}

		fmt.Printf("📝 执行语句 %d:\n", i+1)
		// 显示语句的前50个字符
		preview := statement
		if len(preview) > 50 {
			preview = preview[:50] + "..."
		}
		fmt.Printf("   %s\n", preview)

		// 使用GORM执行原始SQL
		result := db.Exec(statement)
		if result.Error != nil {
			return fmt.Errorf("failed to execute statement %d: %v\nStatement preview: %s", i+1, result.Error, preview)
		}

		fmt.Printf("   ✅ 成功 (影响行数: %d)\n", result.RowsAffected)
	}

	return nil
}

// 改进的SQL语句分割函数
func splitSQLStatements(content string) []string {
	var statements []string
	var current strings.Builder

	lines := strings.Split(content, "\n")

	for _, line := range lines {
		line = strings.TrimSpace(line)

		// 跳过空行和注释
		if line == "" || strings.HasPrefix(line, "--") {
			continue
		}

		current.WriteString(line)
		current.WriteString(" ")

		// 如果行以分号结束，这是一个完整的语句
		if strings.HasSuffix(line, ";") {
			stmt := strings.TrimSpace(current.String())
			stmt = strings.TrimSuffix(stmt, ";") // 移除最后的分号
			if stmt != "" {
				statements = append(statements, stmt)
			}
			current.Reset()
		}
	}

	// 处理最后一个语句（如果没有以分号结束）
	if current.Len() > 0 {
		stmt := strings.TrimSpace(current.String())
		if stmt != "" {
			statements = append(statements, stmt)
		}
	}

	return statements
}
