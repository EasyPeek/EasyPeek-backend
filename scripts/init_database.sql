-- EasyPeek 数据库初始化脚本
-- 请以 PostgreSQL 超级用户（如 postgres）身份运行此脚本

-- 创建数据库
CREATE DATABASE easypeekdb
    WITH 
    OWNER = postgres
    ENCODING = 'UTF8'
    LC_COLLATE = 'en_US.UTF-8'
    LC_CTYPE = 'en_US.UTF-8'
    TABLESPACE = pg_default
    CONNECTION LIMIT = -1
    IS_TEMPLATE = False;

-- 创建专用用户（可选）
-- CREATE USER easypeek WITH PASSWORD 'PostgresPassword';

-- 授权给用户（如果创建了专用用户）
-- GRANT ALL PRIVILEGES ON DATABASE easypeekdb TO easypeek;

-- 连接到新创建的数据库
\c easypeekdb;

-- 创建扩展（如果需要）
-- CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- 显示数据库信息
SELECT 'Database easypeekdb created successfully!' as message;
\l easypeekdb
