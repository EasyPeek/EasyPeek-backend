# 📚 EasyPeek 新闻系统数据库迁移指南

## 🎯 迁移脚本说明

### 文件结构
```
migrations/
├── 001_create_news_tables.sql    # 创建新闻表结构
├── 001_rollback_news_tables.sql  # 回滚脚本（删除所有新闻表）
└── README.md                     # 本使用说明
```

## 🚀 使用方法

### 方法一：通过psql命令行执行
```bash
# 进入项目目录
cd d:\EasyPeek\EasyPeek-backend

# 执行迁移脚本
psql -h localhost -U your_username -d easypeek_db -f migrations/001_create_news_tables.sql

# 如需回滚（谨慎！）
psql -h localhost -U your_username -d easypeek_db -f migrations/001_rollback_news_tables.sql
```

### 方法二：通过数据库管理工具
1. 打开pgAdmin、DBeaver或其他数据库工具
2. 连接到EasyPeek数据库
3. 打开并执行 `001_create_news_tables.sql` 文件内容

### 方法三：通过Go代码执行（推荐协作）
```go
// 在项目根目录创建 cmd/migrate/main.go
package main

import (
    "io/ioutil"
    "log"
    "github.com/EasyPeek/EasyPeek-backend/internal/config"
    "github.com/EasyPeek/EasyPeek-backend/internal/database"
)

func main() {
    // 加载配置
    cfg, err := config.LoadConfig("internal/config/config.yaml")
    if err != nil {
        log.Fatal(err)
    }

    // 初始化数据库
    if err := database.Initialize(cfg); err != nil {
        log.Fatal(err)
    }
    defer database.CloseDatabase()

    // 读取并执行迁移脚本
    sqlContent, err := ioutil.ReadFile("migrations/001_create_news_tables.sql")
    if err != nil {
        log.Fatal(err)
    }

    db := database.GetDB()
    if err := db.Exec(string(sqlContent)).Error; err != nil {
        log.Fatal("Migration failed:", err)
    }

    log.Println("Migration completed successfully!")
}
```

## 📊 迁移内容详解

### 1. 表结构
- **rss_sources**: RSS源管理表
- **news**: 新闻主表
- **event_news_relations**: 事件-新闻关联表

### 2. 索引优化
- 单字段索引：提升常用字段查询性能
- 复合索引：优化排序和多条件查询
- 唯一索引：确保数据唯一性

### 3. 自动化功能
- **热度计算函数**: 自动计算新闻热度分值
- **更新触发器**: 统计数据变更时自动更新热度
- **统计视图**: 便于数据分析和报表生成

### 4. 示例数据
- 5个常用RSS源
- 3条示例新闻
- 自动计算的热度分值

## ⚠️ 注意事项

### 协作者同步步骤
1. **拉取最新代码**: `git pull origin main`
2. **执行迁移脚本**: 运行上述任一方法
3. **验证结果**: 检查表是否创建成功
4. **启动项目**: 确保程序正常运行

### 数据安全
- ✅ 迁移前请备份现有数据
- ✅ 在测试环境先验证脚本
- ⚠️ 回滚脚本会删除所有数据
- ⚠️ 生产环境执行前请三思

### 常见问题
1. **权限问题**: 确保数据库用户有创建表的权限
2. **依赖表缺失**: 确保users和events表已存在
3. **重复执行**: 脚本使用IF NOT EXISTS，可安全重复执行

## 🔍 验证迁移成功

### 检查表结构
```sql
-- 查看创建的表
\dt

-- 查看表结构
\d rss_sources
\d news
\d event_news_relations
```

### 检查功能
```sql
-- 查看示例数据
SELECT * FROM news LIMIT 5;

-- 测试热度计算函数
SELECT calculate_news_hotness(1000, 50, 10, 5, NOW() - INTERVAL '2 hours');

-- 查看统计视图
SELECT * FROM news_stats_summary;
```

### 检查Go模型同步
```bash
# 重新编译项目
go build ./cmd/

# 运行项目确保模型匹配
go run ./cmd/main.go
```

## 🤝 团队协作建议

1. **统一迁移**: 所有团队成员应使用相同的迁移脚本
2. **版本控制**: 迁移脚本纳入Git版本控制
3. **文档更新**: 及时更新API文档和数据字典
4. **测试覆盖**: 为新功能编写单元测试和集成测试

## 📞 遇到问题？

如果在迁移过程中遇到问题：
1. 检查数据库连接和权限
2. 查看错误日志确定具体问题
3. 参考本文档的常见问题部分
4. 联系项目维护者获得帮助
