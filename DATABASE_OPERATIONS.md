# 📰 EasyPeek 新闻系统数据库操作指南

## 🎯 概述

本指南专注于数据库操作，避免与Docker PostgreSQL版本冲突。已提供多种无需本地PostgreSQL安装的解决方案。

## 📊 数据库结构

### 主要表结构
- **`news`** - 新闻主表（简化版，无RSS相关字段）
- **`events`** - 事件主表，存储重要事件信息
- **`event_news_relations`** - 事件新闻关联表
- **视图和函数** - 热度计算、统计分析

## 🚀 使用方法（无冲突方案）

### 方案一：VS Code扩展（推荐）

已安装扩展：
- PostgreSQL Client (cweijan.vscode-postgresql-client2)
- SQLTools PostgreSQL Driver (mtxr.sqltools-driver-pg)

#### 使用步骤：
1. 在VS Code中按 `Ctrl+Shift+P` 打开命令面板
2. 搜索 "PostgreSQL: New Connection"
3. 输入数据库连接信息
4. 连接后可直接执行SQL文件

### 方案二：Go脚本执行器（推荐）

#### Windows批处理方式：
```bash
# 执行数据库迁移
migrate.bat migrations/001_create_news_tables.sql

# 插入示例数据
migrate.bat migrations/insert_sample_news.sql
```

#### PowerShell方式：
```powershell
# 执行数据库迁移
.\scripts\migrate.ps1 migrations/001_create_news_tables.sql

# 插入示例数据
.\scripts\migrate.ps1 migrations/insert_sample_news.sql
```

#### 直接Go命令：
```bash
# 执行数据库迁移
go run scripts/migrate.go migrations/001_create_news_tables.sql

# 插入示例数据
go run scripts/migrate.go migrations/insert_sample_news.sql
```

### 方案三：便携式工具（无需安装）

#### DBeaver便携版
1. 下载：https://dbeaver.io/download/
2. 选择"Portable version"
3. 解压即用，无需安装
4. 支持直接执行SQL脚本文件

## 📝 直接插入数据示例

### 基础插入语法
```sql
INSERT INTO news (
    title, content, summary, source, category, published_at, 
    author, tags, view_count, like_count, status
) VALUES (
    '新闻标题',
    '新闻正文内容...',
    '新闻摘要',
    '新闻来源',
    '分类',
    '2025-06-30 10:00:00',
    '作者姓名',
    '["标签1", "标签2"]',
    100,  -- 浏览量
    10,   -- 点赞数
    'published'
);
```

### 事件表操作示例
```sql
-- 创建事件表 (如果使用迁移脚本，此步骤可省略)
CREATE TABLE IF NOT EXISTS events (
    id SERIAL PRIMARY KEY,
    title VARCHAR(255) NOT NULL,
    description TEXT,
    category VARCHAR(50) NOT NULL,
    start_date TIMESTAMP NOT NULL,
    end_date TIMESTAMP,
    location VARCHAR(255),
    importance INTEGER DEFAULT 0,
    tags JSONB,
    is_active BOOLEAN DEFAULT TRUE,
    status VARCHAR(50) DEFAULT 'published',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP
);

-- 插入事件数据
INSERT INTO events (
    title, description, category, start_date, end_date, 
    location, importance, tags, status
) VALUES (
    '2025年全国两会',
    '中国人民政治协商会议第十四届全国委员会第三次会议和中华人民共和国第十四届全国人民代表大会第三次会议',
    '政治',
    '2025-03-04 00:00:00',
    '2025-03-15 00:00:00',
    '北京',
    5,  -- 重要性级别 (1-5)
    '["政治", "两会", "国家事务"]',
    'published'
);

-- 批量插入事件
INSERT INTO events (title, description, category, start_date, importance, tags) VALUES 
('世界人工智能大会', '全球AI技术盛会，展示最新AI成果', '科技', '2025-07-10 09:00:00', 4, '["科技", "AI", "创新"]'),
('中国国际进口博览会', '第八届中国国际进口博览会', '经济', '2025-11-05 00:00:00', 5, '["经济", "贸易", "国际"]'),
('冬季奥运会', '2026年冬季奥林匹克运动会', '体育', '2026-02-06 00:00:00', 5, '["体育", "奥运", "国际"]');

-- 创建事件与新闻关联
INSERT INTO event_news_relations (event_id, news_id, relation_type) VALUES 
(1, 15, 'primary'),  -- 主要相关新闻
(1, 16, 'related'),  -- 相关新闻
(1, 17, 'background');  -- 背景新闻
```

## 🔍 数据查询示例

### 基础查询
```sql
-- 查看所有新闻
SELECT id, title, source, category, hotness_score, published_at 
FROM news 
ORDER BY published_at DESC 
LIMIT 10;

-- 查看所有事件
SELECT id, title, category, start_date, end_date, importance 
FROM events 
WHERE is_active = true
ORDER BY start_date DESC 
LIMIT 10;

-- 按分类查询事件
SELECT * FROM events 
WHERE category = '政治' AND is_active = true
ORDER BY start_date DESC;
```

### 高级查询
```sql
-- 使用统计视图
SELECT * FROM news_stats_summary;

-- 查询事件及其关联新闻
SELECT e.id as event_id, e.title as event_title, 
       n.id as news_id, n.title as news_title, 
       enr.relation_type
FROM events e
JOIN event_news_relations enr ON e.id = enr.event_id
JOIN news n ON enr.news_id = n.id
WHERE e.id = 1;

-- 查询正在进行的事件
SELECT * FROM events
WHERE start_date <= NOW() 
AND (end_date IS NULL OR end_date >= NOW())
AND is_active = true
ORDER BY importance DESC;

-- 搜索事件
SELECT id, title, description 
FROM events 
WHERE title ILIKE '%大会%' OR description ILIKE '%大会%';
```

## 📈 热度管理

### 手动更新热度分数
```sql
-- 更新单条新闻热度
UPDATE news 
SET hotness_score = calculate_news_hotness(view_count, like_count, comment_count, share_count, published_at)
WHERE id = 1;

-- 批量更新所有新闻热度
UPDATE news 
SET hotness_score = calculate_news_hotness(view_count, like_count, comment_count, share_count, published_at);
```

### 模拟用户交互
```sql
-- 增加浏览量
UPDATE news SET view_count = view_count + 1 WHERE id = 1;

-- 增加点赞数
UPDATE news SET like_count = like_count + 1 WHERE id = 1;

-- 增加评论数
UPDATE news SET comment_count = comment_count + 1 WHERE id = 1;

-- 增加分享数
UPDATE news SET share_count = share_count + 1 WHERE id = 1;
```

## 🛠️ 维护操作

### 数据清理
```sql
-- 删除测试数据
DELETE FROM news WHERE source = '测试来源';
DELETE FROM events WHERE title LIKE '%测试%';

-- 软删除（推荐）
UPDATE news SET deleted_at = NOW() WHERE id = 1;
UPDATE events SET deleted_at = NOW() WHERE id = 1;

-- 清理事件与新闻关联
DELETE FROM event_news_relations 
WHERE event_id IN (SELECT id FROM events WHERE deleted_at IS NOT NULL)
OR news_id IN (SELECT id FROM news WHERE deleted_at IS NOT NULL);
```

### 数据备份
```sql
-- 导出新闻数据
COPY (SELECT * FROM news WHERE is_active = true) 
TO '/path/to/backup/news_backup.csv' 
WITH CSV HEADER;

-- 导出特定分类数据
COPY (SELECT * FROM news WHERE category = '科技' AND is_active = true) 
TO '/path/to/backup/tech_news.csv' 
WITH CSV HEADER;
```

## 📊 性能优化

### 查看索引使用情况
```sql
-- 查看查询计划
EXPLAIN ANALYZE 
SELECT * FROM news 
WHERE category = '科技' 
ORDER BY hotness_score DESC 
LIMIT 10;

-- 查看表统计信息
SELECT 
    schemaname,
    tablename,
    n_tup_ins as inserts,
    n_tup_upd as updates,
    n_tup_del as deletes
FROM pg_stat_user_tables 
WHERE tablename = 'news';
```

### 定期维护
```sql
-- 更新表统计信息
ANALYZE news;

-- 重建索引（如有必要）
REINDEX TABLE news;
```

## ⚡ 快速验证

### 检查数据完整性
```sql
-- 验证数据是否正常
SELECT 
    COUNT(*) as total_news,
    COUNT(DISTINCT category) as categories,
    AVG(hotness_score) as avg_hotness,
    MAX(published_at) as latest_news
FROM news 
WHERE is_active = true;
```

### 检查热度计算
```sql
-- 测试热度计算函数
SELECT 
    id,
    title,
    view_count,
    like_count,
    hotness_score,
    calculate_news_hotness(view_count, like_count, comment_count, share_count, published_at) as recalculated_score
FROM news 
LIMIT 5;
```

## 🎉 总结

现在您可以：
- ✅ 直接向数据库插入新闻数据
- ✅ 使用简化的表结构（无RSS复杂性）
- ✅ 享受自动热度计算功能
- ✅ 使用强大的查询和统计视图
- ✅ 保持代码结构不变（RSS字段在代码中保留）

这种设计既简化了数据库操作，又保持了代码的完整性和扩展性！

docker exec postgres_easypeak psql -U postgres -c "ALTER USER postgres PASSWORD 'password';"
docker exec postgres_easypeak psql -U postgres -c "CREATE DATABASE easypeek;"
.\migrate.bat migrations\simple_init.sql 
.\migrate.bat migrations\add_missing_fields.sql