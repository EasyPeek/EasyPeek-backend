-- ================================================
-- EasyPeek 新闻系统数据库迁移脚本 (简化版)
-- 版本: v1.1 (移除外键依赖)
-- 创建时间: 2025-06-30
-- 说明: 创建新闻相关表结构，无外键依赖
-- ================================================

-- 1. 创建新闻表（移除外键依赖）
CREATE TABLE IF NOT EXISTS news (
    id SERIAL PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    content TEXT NOT NULL,
    summary TEXT,
    description TEXT,
    source VARCHAR(100),
    category VARCHAR(100),
    published_at TIMESTAMP NOT NULL,
    created_by INTEGER NULL, -- 移除外键约束
    is_active BOOLEAN DEFAULT true,
    
    -- 新闻媒体字段
    author VARCHAR(100),
    image_url VARCHAR(1000),
    tags TEXT, -- JSON字符串格式
    language VARCHAR(10) DEFAULT 'zh',
    
    -- 统计字段
    view_count BIGINT DEFAULT 0,
    like_count BIGINT DEFAULT 0,
    comment_count BIGINT DEFAULT 0,
    share_count BIGINT DEFAULT 0,
    hotness_score DECIMAL(10,2) DEFAULT 0,
    
    -- 状态字段
    status VARCHAR(20) DEFAULT 'published',
    is_processed BOOLEAN DEFAULT false,
    
    -- 时间戳
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);

-- 为新闻表创建索引
CREATE INDEX IF NOT EXISTS idx_news_title ON news(title);
CREATE INDEX IF NOT EXISTS idx_news_category ON news(category);
CREATE INDEX IF NOT EXISTS idx_news_published_at ON news(published_at);
CREATE INDEX IF NOT EXISTS idx_news_created_by ON news(created_by);
CREATE INDEX IF NOT EXISTS idx_news_hotness_score ON news(hotness_score);
CREATE INDEX IF NOT EXISTS idx_news_status ON news(status);
CREATE INDEX IF NOT EXISTS idx_news_deleted_at ON news(deleted_at);
CREATE INDEX IF NOT EXISTS idx_news_is_active ON news(is_active);

-- 复合索引优化查询性能
CREATE INDEX IF NOT EXISTS idx_news_category_published ON news(category, published_at DESC);
CREATE INDEX IF NOT EXISTS idx_news_hotness_published ON news(hotness_score DESC, published_at DESC);
CREATE INDEX IF NOT EXISTS idx_news_active_published ON news(is_active, published_at DESC);

-- 2. 创建新闻查询视图（简化版，无用户表依赖）
CREATE OR REPLACE VIEW news_with_stats AS
SELECT 
    n.*,
    -- 计算相对热度排名
    RANK() OVER (PARTITION BY n.category ORDER BY n.hotness_score DESC) as category_rank,
    RANK() OVER (ORDER BY n.hotness_score DESC) as global_rank
FROM news n
WHERE n.deleted_at IS NULL AND n.is_active = true;

-- 3. 创建热度计算函数
CREATE OR REPLACE FUNCTION calculate_news_hotness(
    p_view_count BIGINT,
    p_like_count BIGINT, 
    p_comment_count BIGINT,
    p_share_count BIGINT,
    p_published_at TIMESTAMP
) RETURNS DECIMAL(10,2) AS $$
DECLARE
    time_decay DECIMAL(10,2);
    engagement_score DECIMAL(10,2);
    final_score DECIMAL(10,2);
    hours_since_published DECIMAL(10,2);
BEGIN
    -- 计算发布后经过的小时数
    hours_since_published := EXTRACT(EPOCH FROM (NOW() - p_published_at)) / 3600.0;
    
    -- 时间衰减因子 (24小时内权重为1，之后每24小时衰减10%)
    time_decay := CASE 
        WHEN hours_since_published <= 24 THEN 1.0
        WHEN hours_since_published <= 72 THEN 0.8
        WHEN hours_since_published <= 168 THEN 0.6  -- 一周内
        WHEN hours_since_published <= 720 THEN 0.4  -- 一个月内
        ELSE 0.2
    END;
    
    -- 交互得分计算 (浏览:1, 点赞:3, 评论:5, 分享:8)
    engagement_score := (p_view_count * 1.0) + (p_like_count * 3.0) + 
                       (p_comment_count * 5.0) + (p_share_count * 8.0);
    
    -- 计算最终热度得分
    final_score := engagement_score * time_decay;
    
    RETURN final_score;
END;
$$ LANGUAGE plpgsql;

-- 4. 创建更新热度得分的触发器函数
CREATE OR REPLACE FUNCTION update_news_hotness() RETURNS TRIGGER AS $$
BEGIN
    NEW.hotness_score := calculate_news_hotness(
        NEW.view_count,
        NEW.like_count,
        NEW.comment_count,
        NEW.share_count,
        NEW.published_at
    );
    NEW.updated_at := CURRENT_TIMESTAMP;
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- 5. 创建热度更新触发器
DROP TRIGGER IF EXISTS trigger_update_news_hotness ON news;
CREATE TRIGGER trigger_update_news_hotness
    BEFORE UPDATE OF view_count, like_count, comment_count, share_count
    ON news
    FOR EACH ROW
    EXECUTE FUNCTION update_news_hotness();

-- 6. 创建新闻统计汇总视图
CREATE OR REPLACE VIEW news_stats_summary AS
SELECT 
    category,
    COUNT(*) as total_news,
    AVG(hotness_score) as avg_hotness,
    SUM(view_count) as total_views,
    SUM(like_count) as total_likes,
    SUM(comment_count) as total_comments,
    SUM(share_count) as total_shares,
    MIN(published_at) as earliest_news,
    MAX(published_at) as latest_news
FROM news
WHERE deleted_at IS NULL AND is_active = true
GROUP BY category
ORDER BY avg_hotness DESC;

-- 7. 创建热门新闻视图
CREATE OR REPLACE VIEW trending_news AS
SELECT 
    n.*,
    CASE 
        WHEN n.published_at >= NOW() - INTERVAL '1 day' THEN 'today'
        WHEN n.published_at >= NOW() - INTERVAL '3 days' THEN 'recent'
        WHEN n.published_at >= NOW() - INTERVAL '7 days' THEN 'week'
        ELSE 'older'
    END as time_category
FROM news n
WHERE n.deleted_at IS NULL 
  AND n.is_active = true
  AND n.hotness_score > 0
ORDER BY n.hotness_score DESC
LIMIT 100;

-- 完成提示
-- 执行完成！新闻系统数据库结构已创建
-- 现在可以开始插入新闻数据了
