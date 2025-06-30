-- 安全地创建事件-新闻关联，不依赖于特定的新闻ID
-- 为两会事件关联政治新闻
DO $$
DECLARE
    event_id_val INT;
    news_id_val INT;
    counter INT := 0;
BEGIN
    -- 获取两会事件ID
    SELECT id INTO event_id_val FROM events WHERE title = '2025年全国两会';
    
    -- 为事件关联最多3个政治类新闻
    FOR news_id_val IN 
        SELECT id FROM news 
        WHERE category = '政治' 
        AND is_active = true
        ORDER BY published_at DESC
        LIMIT 3
    LOOP
        counter := counter + 1;
        
        -- 根据顺序设置关联类型
        IF counter = 1 THEN
            INSERT INTO event_news_relations (event_id, news_id, relation_type)
            VALUES (event_id_val, news_id_val, 'primary');
        ELSIF counter = 2 THEN
            INSERT INTO event_news_relations (event_id, news_id, relation_type)
            VALUES (event_id_val, news_id_val, 'related');
        ELSE
            INSERT INTO event_news_relations (event_id, news_id, relation_type)
            VALUES (event_id_val, news_id_val, 'background');
        END IF;
    END LOOP;
    
    -- 为AI大会关联科技新闻
    SELECT id INTO event_id_val FROM events WHERE title = '世界人工智能大会2025';
    counter := 0;
    
    FOR news_id_val IN 
        SELECT id FROM news 
        WHERE category = '科技' OR category = '技术'
        AND is_active = true
        ORDER BY published_at DESC
        LIMIT 2
    LOOP
        counter := counter + 1;
        
        IF counter = 1 THEN
            INSERT INTO event_news_relations (event_id, news_id, relation_type)
            VALUES (event_id_val, news_id_val, 'primary');
        ELSE
            INSERT INTO event_news_relations (event_id, news_id, relation_type)
            VALUES (event_id_val, news_id_val, 'related');
        END IF;
    END LOOP;
    
    -- 为进口博览会关联经济新闻
    SELECT id INTO event_id_val FROM events WHERE title = '中国国际进口博览会2025';
    
    -- 由于可能没有经济类新闻，我们关联一些政治新闻作为示例
    FOR news_id_val IN 
        SELECT id FROM news 
        WHERE is_active = true
        ORDER BY hotness_score DESC
        LIMIT 2
    LOOP
        INSERT INTO event_news_relations (event_id, news_id, relation_type)
        VALUES (event_id_val, news_id_val, 'related');
    END LOOP;
END $$;
