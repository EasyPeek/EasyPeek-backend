# EasyPeek API 文档

## 📋 目录

- [基本信息](#基本信息)
- [认证机制](#认证机制)
- [通用响应格式](#通用响应格式)
- [健康检查](#健康检查)
- [用户认证](#用户认证)
- [用户管理](#用户管理)
- [事件管理](#事件管理)
- [管理员接口](#管理员接口)
- [错误码说明](#错误码说明)
- [使用示例](#使用示例)

---

## 📝 基本信息

- **Base URL**: `http://localhost:8080`
- **API Version**: v1
- **Content-Type**: `application/json`
- **编码**: `UTF-8`
- **支持的HTTP方法**: GET, POST, PUT, DELETE

---

## 🔐 认证机制

需要认证的接口需要在请求头中包含 JWT Token：

```http
Authorization: Bearer <your_jwt_token>
```

### Token 获取方式
通过 `/api/v1/auth/login` 接口获取 JWT Token。

---

## 📊 通用响应格式

所有 API 接口都遵循统一的响应格式：

```json
{
  "code": 200,
  "message": "success",
  "data": {}
}
```

**字段说明:**
- `code`: HTTP状态码
- `message`: 响应消息
- `data`: 响应数据（可为对象、数组或null）

---

## ❤️ 健康检查

### GET /health

检查服务器运行状态

**描述**: 用于监控服务器是否正常运行

**请求示例:**
```bash
curl -X GET http://localhost:8080/health
```

**响应示例:**
```json
{
  "status": "ok",
  "message": "EasyPeek backend is running"
}
```

---

## 👤 用户认证

### POST /api/v1/auth/register

用户注册

**请求体:**
```json
{
  "username": "string (required)",
  "email": "string (required, valid email)",
  "password": "string (required, min 6 chars)"
}
```

**请求示例:**
```bash
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "testuser",
    "email": "test@example.com",
    "password": "password123"
  }'
```

**响应示例:**
```json
{
  "code": 201,
  "message": "User registered successfully",
  "data": {
    "id": 1,
    "username": "testuser",
    "email": "test@example.com",
    "created_at": "2025-06-27T10:00:00Z"
  }
}
```

### POST /api/v1/auth/login

用户登录

**请求体:**
```json
{
  "email": "string (required)",
  "password": "string (required)"
}
```

**请求示例:**
```bash
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "test@example.com",
    "password": "password123"
  }'
```

**响应示例:**
```json
{
  "code": 200,
  "message": "Login successful",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
    "user": {
      "id": 1,
      "username": "testuser",
      "email": "test@example.com",
      "role": "user"
    }
  }
}
```

---

## 👥 用户管理

> 🔒 **所有用户管理接口都需要认证**

### GET /api/v1/user/profile

获取当前用户资料

**请求示例:**
```bash
curl -X GET http://localhost:8080/api/v1/user/profile \
  -H "Authorization: Bearer <token>"
```

**响应示例:**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": 1,
    "username": "testuser",
    "email": "test@example.com",
    "role": "user",
    "created_at": "2025-06-27T10:00:00Z",
    "updated_at": "2025-06-27T10:00:00Z"
  }
}
```

### PUT /api/v1/user/profile

更新用户资料

**请求体:**
```json
{
  "username": "string (optional)",
  "email": "string (optional, valid email)"
}
```

**请求示例:**
```bash
curl -X PUT http://localhost:8080/api/v1/user/profile \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "newusername"
  }'
```

### POST /api/v1/user/change-password

修改密码

**请求体:**
```json
{
  "old_password": "string (required)",
  "new_password": "string (required, min 6 chars)"
}
```

**请求示例:**
```bash
curl -X POST http://localhost:8080/api/v1/user/change-password \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "old_password": "oldpassword123",
    "new_password": "newpassword123"
  }'
```

---

## 📅 事件管理

### GET /api/v1/events

获取事件列表 (公开接口)

**查询参数:**
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `status` | string | 否 | - | 事件状态 (`进行中`, `已结束`) |
| `category` | string | 否 | - | 事件分类 |
| `search` | string | 否 | - | 搜索关键词（标题、描述、内容、地点） |
| `sort_by` | string | 否 | `time` | 排序方式 (`time`, `hotness`, `views`) |
| `page` | int | 否 | 1 | 页码 |
| `limit` | int | 否 | 10 | 每页数量 |

**请求示例:**
```bash
# 获取第一页事件
curl -X GET "http://localhost:8080/api/v1/events?page=1&limit=10"

# 按分类和热度排序
curl -X GET "http://localhost:8080/api/v1/events?category=科技&sort_by=hotness"

# 搜索事件
curl -X GET "http://localhost:8080/api/v1/events?search=AI技术&page=1"
```

**响应示例:**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "total": 25,
    "events": [
      {
        "id": 1,
        "title": "AI技术峰会2025",
        "description": "最新人工智能技术发展趋势探讨",
        "content": "详细的会议内容介绍...",
        "start_time": "2025-07-01T09:00:00Z",
        "end_time": "2025-07-01T18:00:00Z",
        "location": "北京国际会议中心",
        "status": "进行中",
        "category": "科技",
        "tags": "[\"AI\", \"技术峰会\", \"人工智能\"]",
        "source": "科技日报",
        "author": "张三",
        "related_links": "[\"https://ai-summit.com\"]",
        "view_count": 1250,
        "hotness_score": 8.5,
        "created_at": "2025-06-27T10:00:00Z",
        "updated_at": "2025-06-27T10:00:00Z"
      }
    ]
  }
}
```

### GET /api/v1/events/hot

获取热点事件 (公开接口)

**查询参数:**
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `limit` | int | 否 | 10 | 限制数量 |

**请求示例:**
```bash
curl -X GET "http://localhost:8080/api/v1/events/hot?limit=5"
```

**响应示例:**
```json
{
  "code": 200,
  "message": "success",
  "data": [
    {
      "id": 1,
      "title": "AI技术峰会2025",
      "hotness_score": 9.2,
      "view_count": 5432,
      "category": "科技",
      "status": "进行中"
    }
  ]
}
```

### GET /api/v1/events/categories

获取事件分类列表 (公开接口)

**请求示例:**
```bash
curl -X GET http://localhost:8080/api/v1/events/categories
```

**响应示例:**
```json
{
  "code": 200,
  "message": "success",
  "data": ["政治", "经济", "科技", "体育", "娱乐", "社会", "教育", "健康"]
}
```

### GET /api/v1/events/{id}

获取事件详情 (公开接口)

**路径参数:**
- `id` (int, required): 事件ID

**请求示例:**
```bash
curl -X GET http://localhost:8080/api/v1/events/1
```

**响应示例:**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "id": 1,
    "title": "AI技术峰会2025",
    "description": "最新人工智能技术发展趋势探讨",
    "content": "详细的会议内容...",
    "start_time": "2025-07-01T09:00:00Z",
    "end_time": "2025-07-01T18:00:00Z",
    "location": "北京国际会议中心",
    "status": "进行中",
    "category": "科技",
    "tags": "[\"AI\", \"技术峰会\"]",
    "source": "科技日报",
    "author": "张三",
    "related_links": "[\"https://ai-summit.com\"]",
    "view_count": 1251,
    "hotness_score": 8.5,
    "created_at": "2025-06-27T10:00:00Z",
    "updated_at": "2025-06-27T10:00:00Z"
  }
}
```

### GET /api/v1/events/status/{status}

按状态获取事件 (公开接口)

**路径参数:**
- `status` (string, required): 事件状态 (`进行中`, `已结束`)

**请求示例:**
```bash
curl -X GET "http://localhost:8080/api/v1/events/status/进行中"
```

### POST /api/v1/events/{id}/view

记录事件浏览次数 (公开接口)

**描述**: 用户查看事件详情时调用，自动增加浏览计数

**路径参数:**
- `id` (int, required): 事件ID

**请求示例:**
```bash
curl -X POST http://localhost:8080/api/v1/events/1/view
```

**响应示例:**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "message": "View count incremented"
  }
}
```

### POST /api/v1/events

创建事件 (需要认证)

**请求体:**
```json
{
  "title": "string (required, max 200)",
  "description": "string (optional, max 1000)",
  "content": "string (required)",
  "start_time": "datetime (required, ISO 8601)",
  "end_time": "datetime (required, ISO 8601)",
  "location": "string (required, max 255)",
  "category": "string (required, max 50)",
  "tags": ["string"] (optional),
  "source": "string (optional, max 100)",
  "author": "string (optional, max 100)",
  "related_links": ["string"] (optional),
  "image": "string (optional, URL)"
}
```

**请求示例:**
```bash
curl -X POST http://localhost:8080/api/v1/events \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Vue.js 3.0 技术分享会",
    "description": "深入了解Vue.js 3.0的新特性和最佳实践",
    "content": "本次分享将详细介绍Vue.js 3.0的Composition API、性能优化、TypeScript支持等核心特性，以及在实际项目中的应用经验...",
    "start_time": "2025-07-15T19:00:00Z",
    "end_time": "2025-07-15T21:00:00Z",
    "location": "深圳南山科技园腾讯大厦",
    "category": "科技",
    "tags": ["Vue.js", "前端", "JavaScript", "技术分享"],
    "source": "前端开发者社区",
    "author": "王五",
    "related_links": ["https://vuejs.org", "https://github.com/vuejs/vue-next"],
    "image": "https://example.com/vue-event.jpg"
  }'
```

**响应示例:**
```json
{
  "code": 201,
  "message": "Event created successfully",
  "data": {
    "id": 10,
    "title": "Vue.js 3.0 技术分享会",
    "status": "进行中",
    "view_count": 0,
    "hotness_score": 0,
    "created_at": "2025-06-27T12:00:00Z"
  }
}
```

### PUT /api/v1/events/{id}

更新事件 (需要认证)

**路径参数:**
- `id` (int, required): 事件ID

**请求体:** (所有字段都是可选的)
```json
{
  "title": "string (optional, max 200)",
  "description": "string (optional, max 1000)",
  "content": "string (optional)",
  "start_time": "datetime (optional, ISO 8601)",
  "end_time": "datetime (optional, ISO 8601)",
  "location": "string (optional, max 255)",
  "status": "string (optional, enum: 进行中|已结束)",
  "category": "string (optional, max 50)",
  "tags": ["string"] (optional),
  "source": "string (optional, max 100)",
  "author": "string (optional, max 100)",
  "related_links": ["string"] (optional),
  "image": "string (optional, URL)"
}
```

**请求示例:**
```bash
curl -X PUT http://localhost:8080/api/v1/events/10 \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "title": "Vue.js 3.0 高级技术分享会",
    "status": "已结束",
    "tags": ["Vue.js", "前端", "JavaScript", "高级技术"]
  }'
```

### DELETE /api/v1/events/{id}

删除事件 (需要认证)

**描述**: 软删除事件，事件不会真正删除，仅标记为删除状态

**路径参数:**
- `id` (int, required): 事件ID

**请求示例:**
```bash
curl -X DELETE http://localhost:8080/api/v1/events/10 \
  -H "Authorization: Bearer <token>"
```

**响应示例:**
```json
{
  "code": 200,
  "message": "success",
  "data": {
    "message": "Event deleted successfully"
  }
}
```

---

## 👑 管理员接口

> 🔒 **所有管理员接口都需要管理员权限**

### GET /api/v1/admin/users

获取用户列表

**查询参数:**
| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| `page` | int | 否 | 1 | 页码 |
| `limit` | int | 否 | 10 | 每页数量 |

**请求示例:**
```bash
curl -X GET "http://localhost:8080/api/v1/admin/users?page=1&limit=20" \
  -H "Authorization: Bearer <admin_token>"
```

### GET /api/v1/admin/users/{id}

获取用户详情

**路径参数:**
- `id` (int, required): 用户ID

**请求示例:**
```bash
curl -X GET http://localhost:8080/api/v1/admin/users/1 \
  -H "Authorization: Bearer <admin_token>"
```

### DELETE /api/v1/admin/users/{id}

删除用户

**路径参数:**
- `id` (int, required): 用户ID

**请求示例:**
```bash
curl -X DELETE http://localhost:8080/api/v1/admin/users/1 \
  -H "Authorization: Bearer <admin_token>"
```

---

## ⚠️ 错误码说明

| HTTP状态码 | 说明 | 常见场景 |
|------------|------|----------|
| 200 | 请求成功 | 正常的GET、PUT、DELETE请求 |
| 201 | 创建成功 | POST请求成功创建资源 |
| 400 | 请求参数错误 | 参数格式错误、必填参数缺失 |
| 401 | 未授权 | Token无效、过期或缺失 |
| 403 | 权限不足 | 普通用户访问管理员接口 |
| 404 | 资源不存在 | 请求的事件、用户不存在 |
| 409 | 资源冲突 | 邮箱已注册、用户名已存在 |
| 500 | 服务器内部错误 | 数据库连接失败、系统异常 |

### 错误响应格式

```json
{
  "code": 400,
  "message": "Invalid request parameters: title is required",
  "data": null
}
```

### 常见错误示例

**401 Unauthorized:**
```json
{
  "code": 401,
  "message": "Token is invalid or expired",
  "data": null
}
```

**403 Forbidden:**
```json
{
  "code": 403,
  "message": "Insufficient permissions",
  "data": null
}
```

**404 Not Found:**
```json
{
  "code": 404,
  "message": "Event not found",
  "data": null
}
```

---

## 🚀 使用示例

### 完整的事件管理流程

#### 1. 用户注册和登录
```bash
# 注册新用户
curl -X POST http://localhost:8080/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "username": "developer",
    "email": "dev@example.com",
    "password": "securepass123"
  }'

# 用户登录获取Token
curl -X POST http://localhost:8080/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "dev@example.com",
    "password": "securepass123"
  }'
```

#### 2. 浏览和搜索事件
```bash
# 获取热点事件
curl -X GET "http://localhost:8080/api/v1/events/hot?limit=5"

# 获取科技类事件
curl -X GET "http://localhost:8080/api/v1/events?category=科技&sort_by=hotness"

# 搜索包含"AI"的事件
curl -X GET "http://localhost:8080/api/v1/events?search=AI&page=1&limit=5"

# 查看具体事件详情
curl -X GET http://localhost:8080/api/v1/events/1

# 记录浏览次数
curl -X POST http://localhost:8080/api/v1/events/1/view
```

#### 3. 创建和管理事件
```bash
# 获取可用分类
curl -X GET http://localhost:8080/api/v1/events/categories

# 创建新事件
curl -X POST http://localhost:8080/api/v1/events \
  -H "Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..." \
  -H "Content-Type: application/json" \
  -d '{
    "title": "React 18 新特性深度解析",
    "description": "探索React 18中的并发特性、Suspense和自动批处理",
    "content": "React 18带来了许多激动人心的新特性...",
    "start_time": "2025-08-10T14:00:00Z",
    "end_time": "2025-08-10T16:00:00Z",
    "location": "上海浦东软件园",
    "category": "科技",
    "tags": ["React", "前端", "JavaScript", "Web开发"],
    "source": "React开发者社区",
    "author": "李明"
  }'

# 更新事件状态
curl -X PUT http://localhost:8080/api/v1/events/10 \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "status": "已结束"
  }'
```

#### 4. 用户资料管理
```bash
# 查看个人资料
curl -X GET http://localhost:8080/api/v1/user/profile \
  -H "Authorization: Bearer <token>"

# 更新个人信息
curl -X PUT http://localhost:8080/api/v1/user/profile \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "username": "new_developer_name"
  }'

# 修改密码
curl -X POST http://localhost:8080/api/v1/user/change-password \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "old_password": "securepass123",
    "new_password": "newsecurepass456"
  }'
```

### JavaScript 前端调用示例

```javascript
// API 客户端封装
class EasyPeekAPI {
  constructor(baseURL = 'http://localhost:8080') {
    this.baseURL = baseURL;
    this.token = localStorage.getItem('token');
  }

  async request(endpoint, options = {}) {
    const url = `${this.baseURL}${endpoint}`;
    const config = {
      headers: {
        'Content-Type': 'application/json',
        ...(this.token && { Authorization: `Bearer ${this.token}` }),
        ...options.headers,
      },
      ...options,
    };

    const response = await fetch(url, config);
    return response.json();
  }

  // 事件相关API
  async getEvents(params = {}) {
    const query = new URLSearchParams(params).toString();
    return this.request(`/api/v1/events?${query}`);
  }

  async getHotEvents(limit = 10) {
    return this.request(`/api/v1/events/hot?limit=${limit}`);
  }

  async getEventById(id) {
    return this.request(`/api/v1/events/${id}`);
  }

  async createEvent(eventData) {
    return this.request('/api/v1/events', {
      method: 'POST',
      body: JSON.stringify(eventData),
    });
  }

  async incrementViewCount(id) {
    return this.request(`/api/v1/events/${id}/view`, {
      method: 'POST',
    });
  }

  // 用户认证API
  async login(email, password) {
    const response = await this.request('/api/v1/auth/login', {
      method: 'POST',
      body: JSON.stringify({ email, password }),
    });
    
    if (response.code === 200) {
      this.token = response.data.token;
      localStorage.setItem('token', this.token);
    }
    
    return response;
  }

  async register(userData) {
    return this.request('/api/v1/auth/register', {
      method: 'POST',
      body: JSON.stringify(userData),
    });
  }
}

// 使用示例
const api = new EasyPeekAPI();

// 获取热点事件
api.getHotEvents(5).then(response => {
  if (response.code === 200) {
    console.log('热点事件:', response.data);
  }
});

// 用户登录
api.login('user@example.com', 'password123').then(response => {
  if (response.code === 200) {
    console.log('登录成功:', response.data.user);
  }
});
```

---

## 📝 更新日志

### v1.0.0 (2025-06-27)
- ✅ 实现用户认证系统
- ✅ 实现事件CRUD操作
- ✅ 支持事件分类和搜索
- ✅ 添加热点事件功能
- ✅ 实现浏览次数统计
- ✅ 添加管理员用户管理功能

---

## 📞 联系信息

如有API使用问题，请联系开发团队：
- 📧 Email: dev@easypeek.com
- 📱 GitHub: https://github.com/EasyPeek
- 🐛 Bug反馈: https://github.com/EasyPeek/issues

---

*文档最后更新时间: 2025年6月27日*
