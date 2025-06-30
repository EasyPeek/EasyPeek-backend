-- 创建事件表
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

-- 创建事件-新闻关联表
CREATE TABLE IF NOT EXISTS event_news_relations (
    id SERIAL PRIMARY KEY,
    event_id INTEGER NOT NULL,
    news_id INTEGER NOT NULL,
    relation_type VARCHAR(50) NOT NULL, -- 'primary', 'related', 'background' 等
    created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (event_id) REFERENCES events(id),
    FOREIGN KEY (news_id) REFERENCES news(id)
);

-- 创建索引以提高查询性能
CREATE INDEX idx_events_category ON events(category);
CREATE INDEX idx_events_start_date ON events(start_date);
CREATE INDEX idx_events_importance ON events(importance);
CREATE INDEX idx_events_is_active ON events(is_active);
CREATE INDEX idx_event_news_event_id ON event_news_relations(event_id);
CREATE INDEX idx_event_news_news_id ON event_news_relations(news_id);
CREATE INDEX idx_event_news_relation_type ON event_news_relations(relation_type);

-- 创建触发器自动更新更新时间
CREATE OR REPLACE FUNCTION update_modified_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = now();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_events_modtime
BEFORE UPDATE ON events
FOR EACH ROW
EXECUTE PROCEDURE update_modified_column();
