-- EasyPeek 新闻系统数据库初始化脚本
-- 适配容器名称: postgres_easypeak

-- 创建新闻表
CREATE TABLE IF NOT EXISTS news (
    id SERIAL PRIMARY KEY,
    title VARCHAR(500) NOT NULL,
    content TEXT NOT NULL,
    summary TEXT,
    description TEXT,
    source VARCHAR(100),
    category VARCHAR(100),
    published_at TIMESTAMP NOT NULL,
    created_by INTEGER NULL,
    is_active BOOLEAN DEFAULT true,
    author VARCHAR(100),
    image_url VARCHAR(1000),
    tags TEXT,
    language VARCHAR(10) DEFAULT 'zh',
    view_count BIGINT DEFAULT 0,
    like_count BIGINT DEFAULT 0,
    comment_count BIGINT DEFAULT 0,
    share_count BIGINT DEFAULT 0,
    hotness_score DECIMAL(10,2) DEFAULT 0,
    status VARCHAR(20) DEFAULT 'published',
    is_processed BOOLEAN DEFAULT false,
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP NULL
);

-- 创建索引
CREATE INDEX IF NOT EXISTS idx_news_title ON news(title);

CREATE INDEX IF NOT EXISTS idx_news_category ON news(category);

CREATE INDEX IF NOT EXISTS idx_news_published_at ON news(published_at);

CREATE INDEX IF NOT EXISTS idx_news_created_by ON news(created_by);

CREATE INDEX IF NOT EXISTS idx_news_hotness_score ON news(hotness_score);

CREATE INDEX IF NOT EXISTS idx_news_status ON news(status);

CREATE INDEX IF NOT EXISTS idx_news_language ON news(language);

CREATE INDEX IF NOT EXISTS idx_news_is_active ON news(is_active);

-- 插入示例数据
INSERT INTO news (title, content, summary, category, published_at, author, source, view_count, like_count, hotness_score) VALUES
('EasyPeek 新闻系统启动', '系统成功初始化，数据库迁移完成', '新闻系统已就绪', '系统', CURRENT_TIMESTAMP, 'System', 'EasyPeek', 100, 10, 50.0);

INSERT INTO news (title, content, summary, category, published_at, author, source, view_count, like_count, hotness_score) VALUES
('数据库连接测试', 'postgres_easypeak 容器连接正常', '数据库运行正常', '技术', CURRENT_TIMESTAMP, 'Admin', 'EasyPeek', 50, 5, 25.0);
