-- ================================================
-- EasyPeek 新闻系统数据库迁移脚本
-- 版本: v1.1
-- 创建时间: 2025-06-30
-- 说明: 为 news 表添加 belonged_event_id 字段，支持新闻与事件的直接关联
-- ================================================

-- 1. 为新闻表添加事件关联字段
ALTER TABLE news 
ADD COLUMN belonged_event_id INTEGER NULL;

-- 2. 为新字段创建索引
CREATE INDEX IF NOT EXISTS idx_news_belonged_event_id ON news(belonged_event_id);

-- 3. 添加外键约束
ALTER TABLE news
ADD CONSTRAINT fk_news_belonged_event
FOREIGN KEY (belonged_event_id) 
REFERENCES events(id) 
ON DELETE SET NULL;

-- 4. 更新现有的 news_with_stats 视图以包含事件信息
DROP VIEW IF EXISTS news_with_stats;

CREATE OR REPLACE VIEW news_with_stats AS
SELECT 
    n.*,
    u.username as creator_name,
    e.title as event_title,
    -- 计算相对热度排名
    RANK() OVER (PARTITION BY n.category ORDER BY n.hotness_score DESC) as category_rank,
    RANK() OVER (ORDER BY n.hotness_score DESC) as global_rank
FROM news n
LEFT JOIN users u ON n.created_by = u.id
LEFT JOIN events e ON n.belonged_event_id = e.id
WHERE n.deleted_at IS NULL AND n.is_active = true;

-- 注释：此迁移脚本将添加 belonged_event_id 字段到 news 表中，
-- 使得每个新闻可以直接关联到一个事件，无需使用 event_news_relations 中间表。
-- 通过这种方式，我们可以简化查询并提高性能。
-- 同时，我们保留了 event_news_relations 表，以支持未来可能的多对多关系需求。
