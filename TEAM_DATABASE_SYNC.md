# 🤝 团队协作数据库同步指南

## 📋 问题说明

当我们在Go代码中修改数据库模型时，这些改动**不会自动同步**到其他协作者的数据库中。每个人需要手动执行数据库迁移来保持数据库结构一致。

## 🎯 解决方案

我们创建了标准化的SQL迁移脚本，确保所有团队成员的数据库结构完全一致。

## 📂 文件结构

```
EasyPeek-backend/
├── migrations/
│   ├── 001_create_news_tables.sql      # 创建新闻表结构
│   ├── 001_rollback_news_tables.sql    # 回滚脚本
│   └── README.md                       # 详细使用说明
├── cmd/migrate/main.go                 # Go迁移工具
├── migrate.ps1                        # PowerShell迁移脚本
└── cmd/main.go                        # 主程序（已包含自动迁移）
```

## 🚀 协作者同步步骤

### 步骤1: 拉取最新代码
```bash
git pull origin main
```

### 步骤2: 选择一种迁移方式

#### 方式A: 自动迁移（推荐新手）
```bash
# 直接运行主程序，GORM会自动创建基础表结构
go run cmd/main.go
```
**优点**: 简单快捷，适合开发环境  
**缺点**: 无法创建复杂的索引、函数、触发器

#### 方式B: 使用Go迁移工具（推荐）
```bash
# 执行迁移
go run cmd/migrate/main.go up

# 如需回滚
go run cmd/migrate/main.go down
```

#### 方式C: 使用PowerShell脚本（Windows用户）
```powershell
# 执行迁移
.\migrate.ps1 up

# 如需回滚
.\migrate.ps1 down
```

#### 方式D: 手动执行SQL（高级用户）
```bash
# 连接数据库并执行迁移脚本
psql -h localhost -U postgres -d easypeek_db -f migrations/001_create_news_tables.sql
```

### 步骤3: 验证迁移成功
```bash
# 编译项目
go build ./cmd/

# 运行项目，确保没有错误
go run cmd/main.go
```

## 🔍 验证同步结果

### 检查数据库表
```sql
-- 连接数据库后执行
\dt  -- 查看所有表

-- 应该看到以下表：
-- users
-- events  
-- rss_sources
-- news
-- event_news_relations
```

### 检查示例数据
```sql
-- 查看示例RSS源
SELECT name, url, category FROM rss_sources;

-- 查看示例新闻
SELECT title, source, category, hotness_score FROM news LIMIT 5;
```

### 检查高级功能
```sql
-- 测试热度计算函数
SELECT calculate_news_hotness(1000, 50, 10, 5, NOW());

-- 查看统计视图
SELECT * FROM news_stats_summary;
```

## ⚠️ 常见问题及解决方案

### 问题1: 权限不足
```
错误: permission denied for relation xxx
```
**解决方案**: 确保数据库用户有创建表的权限
```sql
GRANT ALL PRIVILEGES ON DATABASE easypeek_db TO your_username;
```

### 问题2: 表已存在
```
错误: relation "news" already exists
```
**解决方案**: 脚本使用了`IF NOT EXISTS`，可以安全重复执行。如果仍有问题，检查表结构是否匹配。

### 问题3: 依赖表缺失
```
错误: relation "users" does not exist
```
**解决方案**: 确保先创建了users和events表
```bash
# 先运行主程序创建基础表
go run cmd/main.go
# 然后执行新闻表迁移
go run cmd/migrate/main.go up
```

### 问题4: 连接数据库失败
**解决方案**: 检查配置文件`internal/config/config.yaml`中的数据库连接信息

## 📋 最佳实践

### 1. 团队约定
- ✅ 所有数据库结构变更必须通过迁移脚本
- ✅ 迁移脚本纳入Git版本控制
- ✅ 在合并代码前，确保迁移脚本已测试
- ✅ 生产环境部署前，先在测试环境验证迁移

### 2. 开发流程
```mermaid
graph LR
    A[修改Go模型] --> B[创建迁移脚本] 
    B --> C[本地测试] 
    C --> D[提交代码]
    D --> E[团队成员拉取] 
    E --> F[执行迁移]
    F --> G[验证同步]
```

### 3. 备份策略
- 生产环境执行迁移前必须备份数据库
- 测试环境定期备份，便于快速恢复
- 重要迁移操作前，导出关键数据

## 🆘 需要帮助？

1. **查看详细文档**: `migrations/README.md`
2. **检查迁移脚本**: `migrations/001_create_news_tables.sql`
3. **查看错误日志**: 运行时的错误信息通常很详细
4. **联系项目维护者**: 遇到复杂问题时及时沟通

## 🎉 总结

通过使用标准化的迁移脚本，我们确保了：
- ✅ 所有团队成员的数据库结构完全一致
- ✅ 新功能（热度计算、统计视图）正常工作
- ✅ 数据库性能优化（索引、触发器）生效
- ✅ 示例数据帮助快速开始开发

记住：**数据库同步是团队协作的关键环节，请认真对待每一次迁移操作！**
