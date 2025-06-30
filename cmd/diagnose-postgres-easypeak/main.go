package main

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func main() {
	fmt.Println("🔍 EasyPeek 数据库诊断工具")
	fmt.Println("容器名称: postgres_easypeak")
	fmt.Println("==============================")
	fmt.Println()

	// 1. 检查Docker状态
	fmt.Println("📋 1. 检查Docker状态...")
	checkDockerStatus()

	// 2. 检查容器状态
	fmt.Println("\n🐳 2. 检查容器状态...")
	checkContainerStatus()

	// 3. 检查端口映射
	fmt.Println("\n🔌 3. 检查端口映射...")
	checkPortMapping()

	// 4. 测试数据库连接
	fmt.Println("\n💾 4. 测试数据库连接...")
	testDatabaseConnection()

	fmt.Println("\n🎉 诊断完成!")
}

func checkDockerStatus() {
	cmd := exec.Command("docker", "version")
	err := cmd.Run()
	if err != nil {
		fmt.Println("❌ Docker未运行或未安装")
		fmt.Println("请启动Docker Desktop或安装Docker")
		os.Exit(1)
	}
	fmt.Println("✅ Docker运行正常")
}

func checkContainerStatus() {
	containerName := "postgres_easypeak"

	// 检查容器是否运行
	cmd := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.Names}}")
	output, err := cmd.Output()

	if err != nil {
		fmt.Printf("❌ 检查容器状态失败: %v\n", err)
		return
	}

	if strings.TrimSpace(string(output)) != "" {
		fmt.Printf("✅ 容器 %s 正在运行\n", containerName)

		// 获取容器详细信息
		cmd = exec.Command("docker", "inspect", containerName, "--format", "{{.State.Status}}")
		status, err := cmd.Output()
		if err == nil {
			fmt.Printf("   状态: %s\n", strings.TrimSpace(string(status)))
		}
	} else {
		fmt.Printf("❌ 容器 %s 未运行\n", containerName)

		// 检查容器是否存在但停止了
		cmd = exec.Command("docker", "ps", "-a", "--filter", fmt.Sprintf("name=%s", containerName), "--format", "{{.Names}}")
		output, err = cmd.Output()
		if err == nil && strings.TrimSpace(string(output)) != "" {
			fmt.Println("⚠️ 容器存在但已停止")
			fmt.Printf("启动命令: docker start %s\n", containerName)
		} else {
			fmt.Println("⚠️ 容器不存在")
			fmt.Println("创建命令: docker run -d --name postgres_easypeak -e POSTGRES_PASSWORD=password -p 5432:5432 postgres:15")
		}
	}
}

func checkPortMapping() {
	containerName := "postgres_easypeak"

	cmd := exec.Command("docker", "port", containerName)
	output, err := cmd.Output()

	if err != nil {
		fmt.Printf("❌ 无法获取端口信息: %v\n", err)
		return
	}

	portInfo := strings.TrimSpace(string(output))
	if portInfo != "" {
		fmt.Println("✅ 端口映射:")
		fmt.Printf("   %s\n", portInfo)
	} else {
		fmt.Println("❌ 没有端口映射")
	}
}

func testDatabaseConnection() {
	// 构建连接字符串
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=%s TimeZone=Asia/Shanghai",
		getEnv("DB_HOST", "localhost"),
		getEnv("DB_USER", "postgres"),
		getEnv("DB_PASSWORD", "password"),
		getEnv("DB_NAME", "easypeek"),
		getEnv("DB_PORT", "5432"),
		getEnv("DB_SSLMODE", "disable"),
	)

	fmt.Printf("🔗 连接配置:\n")
	fmt.Printf("   主机: %s\n", getEnv("DB_HOST", "localhost"))
	fmt.Printf("   端口: %s\n", getEnv("DB_PORT", "5432"))
	fmt.Printf("   用户: %s\n", getEnv("DB_USER", "postgres"))
	fmt.Printf("   数据库: %s\n", getEnv("DB_NAME", "easypeek"))

	// 尝试连接
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		fmt.Printf("❌ 数据库连接失败: %v\n", err)
		showConnectionTroubleshooting()
		return
	}

	// 测试ping
	sqlDB, err := db.DB()
	if err != nil {
		fmt.Printf("❌ 获取数据库实例失败: %v\n", err)
		return
	}
	defer sqlDB.Close()

	if err := sqlDB.Ping(); err != nil {
		fmt.Printf("❌ 数据库ping失败: %v\n", err)
		return
	}

	fmt.Println("✅ 数据库连接成功!")

	// 获取数据库信息
	var version string
	if err := db.Raw("SELECT version()").Scan(&version).Error; err == nil {
		fmt.Printf("   PostgreSQL版本: %s\n", version[:50]+"...")
	}

	var currentDB string
	if err := db.Raw("SELECT current_database()").Scan(&currentDB).Error; err == nil {
		fmt.Printf("   当前数据库: %s\n", currentDB)
	}

	// 检查news表
	var tableExists bool
	err = db.Raw("SELECT EXISTS (SELECT 1 FROM information_schema.tables WHERE table_name = 'news')").Scan(&tableExists).Error
	if err == nil {
		if tableExists {
			var count int64
			db.Raw("SELECT COUNT(*) FROM news").Scan(&count)
			fmt.Printf("   news表: 存在 (%d条记录)\n", count)
		} else {
			fmt.Println("   news表: 不存在 (需要运行迁移)")
		}
	}
}

func showConnectionTroubleshooting() {
	fmt.Println("\n🛠️ 连接故障排除:")
	fmt.Println("1. 确保容器运行:")
	fmt.Println("   docker start postgres_easypeak")
	fmt.Println("2. 检查容器日志:")
	fmt.Println("   docker logs postgres_easypeak")
	fmt.Println("3. 进入容器测试:")
	fmt.Println("   docker exec postgres_easypeak pg_isready -U postgres")
	fmt.Println("4. 重置密码:")
	fmt.Println("   docker exec postgres_easypeak psql -U postgres -c \"ALTER USER postgres PASSWORD 'password';\"")
	fmt.Println("5. 如果问题持续，重建容器:")
	fmt.Println("   docker rm -f postgres_easypeak")
	fmt.Println("   docker run -d --name postgres_easypeak -e POSTGRES_PASSWORD=password -p 5432:5432 postgres:15")
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
