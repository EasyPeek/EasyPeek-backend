-- 添加导入新闻脚本所需的缺失字段
-- 适配 postgres_easypeak 容器

-- 添加RSS相关字段
ALTER TABLE news ADD COLUMN IF NOT EXISTS source_type VARCHAR(20) DEFAULT 'manual';
ALTER TABLE news ADD COLUMN IF NOT EXISTS rss_source_id INTEGER;
ALTER TABLE news ADD COLUMN IF NOT EXISTS link TEXT DEFAULT '';
ALTER TABLE news ADD COLUMN IF NOT EXISTS guid VARCHAR(255) DEFAULT '';

-- 为新字段创建索引
CREATE INDEX IF NOT EXISTS idx_news_source_type ON news(source_type);
CREATE INDEX IF NOT EXISTS idx_news_rss_source_id ON news(rss_source_id);
CREATE INDEX IF NOT EXISTS idx_news_link ON news(link);
CREATE INDEX IF NOT EXISTS idx_news_guid ON news(guid);

-- 验证字段添加结果
SELECT column_name, data_type, is_nullable, column_default 
FROM information_schema.columns 
WHERE table_name = 'news' 
AND column_name IN ('source_type', 'rss_source_id', 'link', 'guid')
ORDER BY column_name;
