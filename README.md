# EasyPeek-backend
EasyPeek 数据库操作完整流程指南
根据您的代码和项目结构，以下是从数据库建表到迁移再到数据导入的完整流程和相应命令：

1️⃣ 启动数据库容器
首先，需要确保PostgreSQL容器正在运行：
# 检查容器状态
docker ps | findstr postgres_easypeak

# 如果容器不存在或未运行，创建并启动
docker run -d --name postgres_easypeak `
  -e POSTGRES_USER=postgres `
  -e POSTGRES_PASSWORD=PostgresPassword `
  -e POSTGRES_DB=easypeekdb `
  -p 5432:5432 `
  postgres:15

# 或者使用Makefile中的命令
make launch_postgres

2️⃣ 创建数据库表结构
使用迁移脚本创建初始表结构：
# 执行初始表创建脚本
go run scripts/migrate.go migrations/001_create_news_tables.sql

# 或使用Windows批处理
.\migrate.bat migrations\001_create_news_tables.sql

3️⃣ 应用数据库迁移
随着开发进展，应用额外的数据库迁移：
# 添加缺失的字段
go run scripts/migrate.go migrations/add_missing_fields.sql

# 添加事件表
go run scripts/migrate.go migrations/create_events_table.sql

# 添加belonged_event_id字段到news表
go run scripts/migrate.go migrations/add_belonged_event_id.sql

4️⃣ 导入示例数据
导入示例新闻和事件数据：
# 导入新闻数据
go run cmd/import-news/main.go

# 导入事件数据
go run cmd/import-events/main.go

# 关联新闻到事件
go run cmd/link-news-to-events/main.go

5️⃣ 验证数据导入结果
验证数据是否正确导入：
# 验证数据导入
go run cmd/verify/main.go
