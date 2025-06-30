-- ================================================
-- EasyPeek 新闻系统数据库迁移脚本
-- 版本: v1.0 (简化版，无RSS功能)
-- 创建时间: 2025-06-30
-- 说明: 创建新闻相关表结构和基础数据
-- ================================================

-- 1. 创建新闻表
CREATE TABLE IF NOT EXISTS news (
    id SERIAL PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    content TEXT NOT NULL,
    summary TEXT,
    description TEXT,
    source VARCHAR(100),
    category VARCHAR(100),
    published_at TIMESTAMP NOT NULL,
    created_by INTEGER NULL, -- 移除外键引用，避免循环依赖
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

-- 为新闻表创建索引（在表创建之后）
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

-- 2. 创建事件新闻关联表（为未来事件匹配功能预留）
CREATE TABLE IF NOT EXISTS event_news_relations (
    id SERIAL PRIMARY KEY,
    event_id INTEGER NOT NULL REFERENCES events(id),
    news_id INTEGER NOT NULL REFERENCES news(id),
    match_score DECIMAL(10,2) DEFAULT 0,
    match_method VARCHAR(100),
    is_confirmed BOOLEAN DEFAULT false,
    created_by VARCHAR(50) DEFAULT 'system',
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL,
    
    -- 确保同一事件和新闻只能关联一次
    UNIQUE(event_id, news_id)
);

-- 为关联表创建索引
CREATE INDEX IF NOT EXISTS idx_event_news_event_id ON event_news_relations(event_id);
CREATE INDEX IF NOT EXISTS idx_event_news_news_id ON event_news_relations(news_id);
CREATE INDEX IF NOT EXISTS idx_event_news_match_score ON event_news_relations(match_score);
CREATE INDEX IF NOT EXISTS idx_event_news_deleted_at ON event_news_relations(deleted_at);

-- 3. 创建新闻查询视图（优化查询性能）
CREATE OR REPLACE VIEW news_with_stats AS
SELECT 
    n.*,
    u.username as creator_name,
    -- 计算相对热度排名
    RANK() OVER (PARTITION BY n.category ORDER BY n.hotness_score DESC) as category_rank,
    RANK() OVER (ORDER BY n.hotness_score DESC) as global_rank
FROM news n
LEFT JOIN users u ON n.created_by = u.id
WHERE n.deleted_at IS NULL AND n.is_active = true;

-- 4. 创建热度计算函数
CREATE OR REPLACE FUNCTION calculate_news_hotness(
    p_view_count BIGINT,
    p_like_count BIGINT, 
    p_comment_count BIGINT,
    p_share_count BIGINT,
    p_published_at TIMESTAMP
) RETURNS DECIMAL(10,2) AS $$
DECLARE
    view_score DECIMAL(10,2);
    like_score DECIMAL(10,2);
    comment_score DECIMAL(10,2);
    share_score DECIMAL(10,2);
    time_score DECIMAL(10,2);
    total_score DECIMAL(10,2);
    hours_diff INTEGER;
BEGIN
    -- 浏览量分值：每1000次浏览=8分，最高10分
    view_score := LEAST((p_view_count / 1000.0) * 8.0, 10.0);
    
    -- 点赞分值：每100个点赞=1分，最高10分
    like_score := LEAST(p_like_count / 100.0, 10.0);
    
    -- 评论分值：每10个评论=1分，最高10分
    comment_score := LEAST(p_comment_count / 10.0, 10.0);
    
    -- 分享分值：每5个分享=1分，最高10分
    share_score := LEAST(p_share_count / 5.0, 10.0);
    
    -- 时间分值：越新的新闻分值越高
    hours_diff := EXTRACT(EPOCH FROM (CURRENT_TIMESTAMP - p_published_at)) / 3600;
    
    time_score := CASE 
        WHEN hours_diff <= 1 THEN 10.0
        WHEN hours_diff <= 6 THEN 8.0
        WHEN hours_diff <= 24 THEN 5.0
        WHEN hours_diff <= 168 THEN 2.0 -- 7天
        ELSE 0.5
    END;
    
    -- 加权计算总分
    total_score := (view_score * 0.2) + 
                   (like_score * 0.3) + 
                   (comment_score * 0.25) + 
                   (share_score * 0.15) + 
                   (time_score * 0.1);
    
    RETURN ROUND(total_score, 2);
END;
$$ LANGUAGE plpgsql;

-- 5. 创建自动更新热度的触发器
CREATE OR REPLACE FUNCTION update_news_hotness()
RETURNS TRIGGER AS $$
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

-- 创建触发器
DROP TRIGGER IF EXISTS trigger_update_news_hotness ON news;
CREATE TRIGGER trigger_update_news_hotness
    BEFORE UPDATE OF view_count, like_count, comment_count, share_count
    ON news
    FOR EACH ROW
    EXECUTE FUNCTION update_news_hotness();

-- 6. 插入示例新闻数据
-- 6. 插入示例新闻数据
INSERT INTO news (
    title, content, summary, source, category, published_at, 
    author, tags, view_count, like_count, comment_count, share_count, status
) VALUES 
(
    '2025年全国两会圆满闭幕',
    '2025年全国两会在北京圆满闭幕，大会审议通过了多项重要法案和决议，为新一年的发展指明了方向。会议期间，来自全国各地的人大代表和政协委员围绕经济发展、民生改善、环境保护等重要议题进行了深入讨论。',
    '全国两会圆满闭幕，审议通过重要法案，为新一年发展指明方向',
    '新华网',
    '政治',
    '2025-06-30 10:00:00',
    '新华社记者',
    '["政治", "两会", "法案", "发展"]',
    1250, 89, 23, 12, 'published'
),
(
    'AI技术助推医疗诊断精准化',
    '最新研究表明，人工智能技术在医疗诊断领域取得重大突破，准确率提升到95%以上。该技术已在多家三甲医院开始试点应用，为患者提供更加精准的诊断服务。专家表示，AI技术将极大提高医疗效率，降低误诊率。',
    'AI技术在医疗诊断领域实现重大突破，准确率超95%',
    '科技日报',
    '科技',
    '2025-06-30 14:30:00',
    '科技记者',
    '["AI", "医疗", "诊断", "技术"]',
    856, 67, 18, 8, 'published'
),
(
    '新能源汽车销量创历史新高',
    '2025年上半年，新能源汽车销量达到500万辆，同比增长40%，市场占有率突破30%。随着充电基础设施的不断完善和电池技术的持续进步，消费者对新能源汽车的接受度不断提高。',
    '新能源汽车市场表现强劲，上半年销量创新高达500万辆',
    '财经网',
    '经济',
    '2025-06-30 16:45:00',
    '财经记者',
    '["新能源", "汽车", "销量", "增长"]',
    623, 45, 12, 6, 'published'
),
(
    '夏季奥运会筹备工作有序推进',
    '2025年夏季奥运会各项筹备工作正在有序推进中，场馆建设已完成90%，志愿者招募工作即将启动。组委会表示，将全力以赴为世界呈现一届精彩、难忘的奥运盛会。',
    '2025年夏季奥运会筹备工作进展顺利，场馆建设接近完工',
    '体育时报',
    '体育',
    '2025-06-30 18:20:00',
    '体育记者',
    '["奥运会", "体育", "筹备", "场馆"]',
    445, 32, 8, 4, 'published'
),
(
    '国产电影《星辰大海》票房突破10亿',
    '国产科幻电影《星辰大海》上映两周票房突破10亿元，创下国产科幻片新纪录。该片以其精良的制作和深刻的主题获得了观众和评论家的一致好评，标志着国产科幻电影迈入新时代。',
    '国产科幻电影《星辰大海》票房破10亿，创造新纪录',
    '娱乐周刊',
    '娱乐',
    '2025-06-30 20:15:00',
    '娱乐记者',
    '["电影", "科幻", "票房", "国产"]',
    892, 156, 45, 23, 'published'
);

-- 7. 更新所有新闻的热度分数
UPDATE news 
SET hotness_score = calculate_news_hotness(
    view_count, like_count, comment_count, share_count, published_at
),
updated_at = CURRENT_TIMESTAMP
WHERE hotness_score = 0;

-- 8. 创建数据库视图用于统计分析
CREATE OR REPLACE VIEW news_stats_summary AS
SELECT 
    category,
    COUNT(*) as total_news,
    AVG(hotness_score) as avg_hotness,
    SUM(view_count) as total_views,
    SUM(like_count) as total_likes,
    MAX(published_at) as latest_published
FROM news 
WHERE deleted_at IS NULL AND is_active = true
GROUP BY category
ORDER BY total_news DESC;

-- 完成提示
SELECT 'EasyPeek 新闻系统数据库迁移完成!' as status,
       'Tables: news, event_news_relations' as created_tables,
       'Functions: calculate_news_hotness, update_news_hotness' as created_functions,
       'Views: news_with_stats, news_stats_summary' as created_views,
       'Triggers: trigger_update_news_hotness' as created_triggers;
