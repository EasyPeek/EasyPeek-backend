-- ================================================
-- EasyPeek 数据库补充迁移脚本
-- 版本: v1.2 (添加RSS相关字段以匹配Go模型)
-- 创建时间: 2025-06-30
-- 说明: 为news表添加缺失的字段以匹配Go模型
-- ================================================

-- 1. 添加RSS相关字段
ALTER TABLE news ADD COLUMN IF NOT EXISTS source_type VARCHAR(20) DEFAULT 'manual';
ALTER TABLE news ADD COLUMN IF NOT EXISTS rss_source_id INTEGER;
ALTER TABLE news ADD COLUMN IF NOT EXISTS link VARCHAR(1000);
ALTER TABLE news ADD COLUMN IF NOT EXISTS guid VARCHAR(500);

-- 2. 为新字段创建索引
CREATE INDEX IF NOT EXISTS idx_news_source_type ON news(source_type);
CREATE INDEX IF NOT EXISTS idx_news_rss_source_id ON news(rss_source_id);
CREATE INDEX IF NOT EXISTS idx_news_link ON news(link);
CREATE INDEX IF NOT EXISTS idx_news_guid ON news(guid);

-- 3. 更新现有数据，设置默认值
UPDATE news SET source_type = 'manual' WHERE source_type IS NULL;
UPDATE news SET link = '' WHERE link IS NULL;
UPDATE news SET guid = '' WHERE guid IS NULL;

-- 完成提示
-- 执行完成！RSS相关字段已添加到news表
