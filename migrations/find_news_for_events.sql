-- 查找合适的政治类新闻，可以与"两会"事件关联
SELECT id, title 
FROM news 
WHERE category = '政治' 
AND (title LIKE '%两会%' OR content LIKE '%两会%' OR summary LIKE '%两会%')
LIMIT 5;

-- 查找合适的科技类新闻，可以与"AI大会"事件关联
SELECT id, title 
FROM news 
WHERE category = '科技' 
OR (title LIKE '%AI%' OR title LIKE '%人工智能%' OR content LIKE '%人工智能%')
LIMIT 5;

-- 查找合适的经济类新闻，可以与"进口博览会"事件关联
SELECT id, title 
FROM news 
WHERE category = '经济' 
OR (title LIKE '%博览%' OR title LIKE '%进口%' OR content LIKE '%博览会%')
LIMIT 5;

-- 查找合适的体育类新闻，可以与"冬奥会"事件关联
SELECT id, title 
FROM news 
WHERE category = '体育' 
OR (title LIKE '%奥运%' OR title LIKE '%冬奥%' OR content LIKE '%奥运会%')
LIMIT 5;
