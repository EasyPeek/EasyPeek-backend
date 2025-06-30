-- ================================================
-- EasyPeek 新闻系统数据库回滚脚本
-- 版本: v1.0 (简化版，无RSS功能)
-- 创建时间: 2025-06-30
-- 说明: 回滚新闻相关表结构（谨慎使用！）
-- ================================================

-- 警告：此脚本将删除所有新闻相关数据，请确保已备份重要数据！

-- 1. 删除触发器
DROP TRIGGER IF EXISTS trigger_update_news_hotness ON news;

-- 2. 删除函数
DROP FUNCTION IF EXISTS update_news_hotness();
DROP FUNCTION IF EXISTS calculate_news_hotness(BIGINT, BIGINT, BIGINT, BIGINT, TIMESTAMP);

-- 3. 删除视图
DROP VIEW IF EXISTS news_stats_summary;
DROP VIEW IF EXISTS news_with_stats;

-- 4. 删除表（注意顺序，先删除有外键依赖的表）
DROP TABLE IF EXISTS event_news_relations;
DROP TABLE IF EXISTS news;

-- 完成提示
SELECT 'EasyPeek 新闻系统数据库回滚完成!' as status,
       '所有新闻相关表、函数、视图、触发器已删除' as message,
       '请注意：所有新闻数据已丢失！' as warning;
