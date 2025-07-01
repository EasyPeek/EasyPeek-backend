# postgres
launch_postgres:
	@echo "Launching PostgreSQL..."
	docker run --name postgres_easypeek \
	-e POSTGRES_USER=postgres \
	-e POSTGRES_PASSWORD=PostgresPassword \
	-e POSTGRES_DB=easypeekdb \
	-p 5432:5432 \
	-d postgres
	@echo "PostgreSQL launched successfully."

# redis
launch_redis:
	@echo "Launching Redis..."
	docker run --name redis_easypeek \
	-p 6379:6379 \
	-d redis
	@echo "Redis launched successfully."

# 运行应用
run:
	@echo "Starting EasyPeek backend..."
	go run cmd/main.go

# 创建admin用户
create-admin:
	@echo "Creating admin user..."
	@echo "This will create an admin user with default credentials:"
	@echo "Username: admin"
	@echo "Email: admin@easypeek.com"
	@echo "Password: admin123456"
	@echo "Make sure database is running first!"
	go run cmd/main.go --seed-admin-only

# 构建项目
build:
	@echo "Building EasyPeek backend..."
	go build -o bin/easypeek cmd/main.go

# 清理
clean:
	@echo "Cleaning up..."
	rm -rf bin/