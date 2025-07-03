-- 添加用户个人信息字段
ALTER TABLE users ADD COLUMN phone VARCHAR(20) DEFAULT '';
ALTER TABLE users ADD COLUMN location VARCHAR(100) DEFAULT '';
ALTER TABLE users ADD COLUMN bio TEXT DEFAULT '';
ALTER TABLE users ADD COLUMN interests TEXT DEFAULT '';

-- 添加索引以提高查询性能
CREATE INDEX idx_users_location ON users(location);