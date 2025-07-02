-- 清理导入的新闻数据
-- 保留初始化时的系统数据

-- 删除所有导入的新闻数据（保留系统初始化的2条记录）
DELETE FROM news 
WHERE title NOT IN (
    'EasyPeek 新闻系统启动',
    '数据库连接测试'
);

-- 显示剩余记录数
SELECT 'Remaining records:' as message, COUNT(*) as count FROM news;
