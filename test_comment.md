# 评论功能API测试文档

## 数据库表结构

评论表 `comments` 包含以下字段：
- `id`: 主键ID
- `news_id`: 新闻ID（外键）
- `user_id`: 用户ID（外键）
- `content`: 评论内容
- `created_at`: 评论时间
- `updated_at`: 更新时间
- `deleted_at`: 软删除时间

## API接口

### 1. 创建评论
**POST** `/api/v1/comments`
```json
{
  "news_id": 1,
  "content": "这是一条测试评论"
}
```
需要用户认证

### 2. 获取单条评论
**GET** `/api/v1/comments/:id`
公开接口，无需认证

### 3. 获取新闻的评论列表
**GET** `/api/v1/comments/news/:news_id?page=1&size=10`
公开接口，支持分页

### 4. 获取用户的评论列表
**GET** `/api/v1/comments/user/:user_id?page=1&size=10`
公开接口，支持分页

### 5. 更新评论
**PUT** `/api/v1/comments/:id`
```json
{
  "content": "更新后的评论内容"
}
```
需要用户认证，只能更新自己的评论

### 6. 删除评论
**DELETE** `/api/v1/comments/:id`
需要用户认证，只能删除自己的评论（软删除）

### 7. 管理员获取所有评论
**GET** `/api/v1/admin/comments?page=1&size=10`
需要管理员权限

### 8. 管理员删除评论（硬删除）
**DELETE** `/api/v1/admin/comments/:id`
需要管理员权限

## 功能特点

1. **权限控制**: 用户只能修改/删除自己的评论
2. **软删除**: 用户删除评论时使用软删除，管理员可以硬删除
3. **关联更新**: 创建/删除评论时自动更新新闻的评论数
4. **分页支持**: 所有列表接口都支持分页
5. **数据验证**: 评论内容长度限制（1-1000字符）
6. **关联查询**: 支持预加载用户和新闻信息

## 数据库关系

- `comments.news_id` -> `news.id`
- `comments.user_id` -> `users.id`
- 评论创建/删除时自动更新 `news.comment_count` 