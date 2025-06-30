-- 查看新闻表结构
\d news;

-- 查看插入的新闻数量
SELECT COUNT(*) as total_news FROM news;

-- 查看新闻分类统计
SELECT category, COUNT(*) as count 
FROM news 
GROUP BY category 
ORDER BY count DESC;

-- 查看热度最高的前5条新闻
SELECT title, category, hotness_score, view_count, like_count 
FROM news 
ORDER BY hotness_score DESC 
LIMIT 5;

-- 查看新闻统计汇总
SELECT * FROM news_stats_summary;

-- 查看热门新闻
SELECT title, time_category, hotness_score 
FROM trending_news 
LIMIT 10;
