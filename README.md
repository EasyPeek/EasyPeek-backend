# EasyPeek Backend

[![Go Version](https://img.shields.io/badge/Go-1.24.3-blue.svg)](https://golang.org/)
[![Gin Framework](https://img.shields.io/badge/Gin-1.10.1-green.svg)](https://gin-gonic.com/)
[![PostgreSQL](https://img.shields.io/badge/PostgreSQL-12+-blue.svg)](https://www.postgresql.org/)
[![Redis](https://img.shields.io/badge/Redis-7.0+-red.svg)](https://redis.io/)
[![License](https://img.shields.io/badge/License-MIT-yellow.svg)](LICENSE)

EasyPeek 是一个现代化的新闻聚合和事件管理平台后端系统，采用 Go 语言和 Gin 框架开发，支持 RSS 源自动抓取、智能新闻聚合、事件管理和用户系统。

## 🌟 功能特性

### 核心功能
- **新闻管理系统** - 支持手动创建和RSS自动抓取的新闻管理
- **RSS源管理** - 智能RSS源配置、定时抓取和内容去重
- **事件管理** - 新闻事件聚合、热度计算和趋势分析
- **用户系统** - 完整的用户注册、登录、权限管理
- **智能调度** - 基于cron的自动化任务调度系统

### 高级特性
- **自动种子数据** - 首次启动自动导入初始数据
- **热度算法** - 基于浏览量、点赞、评论的智能热度计算
- **缓存系统** - Redis缓存提升系统性能
- **JWT认证** - 安全的用户身份验证和授权
- **RESTful API** - 标准化的API接口设计
- **优雅关闭** - 支持服务优雅停止和资源清理

## 🏗️ 技术栈

### 后端框架
- **[Go](https://golang.org/)** `1.24.3` - 主编程语言
- **[Gin](https://gin-gonic.com/)** `1.10.1` - HTTP Web框架
- **[GORM](https://gorm.io/)** `1.30.0` - ORM数据库操作

### 数据库
- **[PostgreSQL](https://www.postgresql.org/)** `12+` - 主数据库
- **[Redis](https://redis.io/)** `7.0+` - 缓存和会话存储

### 核心依赖
- **[JWT](https://github.com/golang-jwt/jwt)** `5.2.2` - 用户认证
- **[Viper](https://github.com/spf13/viper)** `1.20.1` - 配置管理
- **[Cron](https://github.com/robfig/cron)** `3.0.1` - 定时任务
- **[GoFeed](https://github.com/mmcdole/gofeed)** `1.3.0` - RSS解析
- **[bcrypt](https://golang.org/x/crypto)** - 密码加密

## 📁 项目结构

```
EasyPeek-backend/
├── cmd/                    # 应用程序入口
│   └── main.go            # 主程序文件
├── internal/              # 内部包（私有代码）
│   ├── api/               # HTTP处理器和路由
│   │   ├── router.go      # 路由配置
│   │   ├── user_handler.go    # 用户相关API
│   │   ├── news_handler.go    # 新闻相关API
│   │   ├── event_handler.go   # 事件相关API
│   │   ├── rss_handler.go     # RSS相关API
│   │   └── admin_handler.go   # 管理员API
│   ├── cache/             # 缓存服务
│   │   └── redis.go       # Redis缓存实现
│   ├── config/            # 配置管理
│   │   ├── config.go      # 配置结构和加载
│   │   └── config.yaml    # 配置文件
│   ├── database/          # 数据库连接和迁移
│   │   └── database.go    # 数据库初始化
│   ├── middleware/        # 中间件
│   │   ├── auth.go        # 认证中间件
│   │   ├── cors.go        # CORS中间件
│   │   └── role.go        # 权限中间件
│   ├── models/            # 数据模型
│   │   ├── user.go        # 用户模型
│   │   ├── news.go        # 新闻模型
│   │   ├── event.go       # 事件模型
│   │   └── rss_source.go  # RSS源模型
│   ├── scheduler/         # 定时任务调度器
│   │   └── rss_scheduler.go   # RSS调度器
│   ├── services/          # 业务逻辑服务
│   │   ├── user_service.go    # 用户服务
│   │   ├── news_service.go    # 新闻服务
│   │   ├── event_service.go   # 事件服务
│   │   ├── rss_service.go     # RSS服务
│   │   └── seed_service.go    # 种子数据服务
│   └── utils/             # 工具函数
│       ├── jwt.go         # JWT工具
│       ├── validator.go   # 验证工具
│       └── json_helper.go # JSON辅助函数
├── data/                  # 初始化数据文件
│   └── new.json          # 新闻种子数据（2600+条记录）
├── bin/                   # 编译输出目录
├── docs/                  # 文档目录
├── Makefile              # 构建脚本
├── go.mod                # Go模块依赖
├── go.sum                # 依赖校验文件
└── README.md             # 项目说明文档
```

## 🚀 快速开始

### 环境要求

- **Go** 1.21 或更高版本
- **PostgreSQL** 12 或更高版本
- **Redis** 6.0 或更高版本（可选，用于缓存）
- **Git** 版本控制

### 1. 克隆项目

```bash
git clone https://github.com/EasyPeek/EasyPeek-backend.git
cd EasyPeek-backend
```

### 2. 安装依赖

```bash
go mod tidy
```

### 3. 数据库准备

#### 使用Docker快速启动（推荐）

```bash
# 启动PostgreSQL
make launch_postgres

# 启动Redis（可选）
make launch_redis
```

#### 手动安装
1. 安装并启动PostgreSQL
2. 创建数据库：
   ```sql
   CREATE DATABASE easypeekdb;
   ```

### 4. 配置应用

编辑 `internal/config/config.yaml` 文件：

```yaml
database:
  host: localhost
  port: 5432
  user: postgres
  password: PostgresPassword
  db_name: easypeekdb
  ssl_mode: disable
  max_idle_conns: 10
  max_open_conns: 10

redis:
  address: localhost:6379
  password: ""
  database: 0

jwt:
  secret_key: "your-secret-key-here-change-in-production"
  expire_hours: 24

cors:
  allow_origins:
    - "http://localhost:3000"
    - "http://localhost:8080"
    - "*"

admin:
  email: "admin@easypeek.com"
  username: "admin"
  password: "admin123456"
```

> ⚠️ **重要**：生产环境中请务必修改默认的JWT密钥和管理员密码！

### 5. 运行应用

```bash
# 方式1：直接运行
go run cmd/main.go

# 方式2：编译后运行
go build -o bin/easypeek cmd/main.go
./bin/easypeek
```

### 6. 验证安装

应用启动后，访问以下端点验证：

- **健康检查**：http://localhost:8080/health
- **API文档**：http://localhost:8080/swagger/index.html（如果已配置）

## 📱 API 接口

### 认证接口
```
POST   /api/v1/auth/register     # 用户注册
POST   /api/v1/auth/login        # 用户登录
```

### 用户管理
```
GET    /api/v1/user/profile      # 获取用户信息
PUT    /api/v1/user/profile      # 更新用户信息
POST   /api/v1/user/change-password  # 修改密码
DELETE /api/v1/user/me           # 删除账户
```

### 新闻接口
```
GET    /api/v1/rss/news          # 获取新闻列表
GET    /api/v1/rss/news/hot      # 获取热门新闻
GET    /api/v1/rss/news/latest   # 获取最新新闻
GET    /api/v1/rss/news/:id      # 获取新闻详情
GET    /api/v1/rss/news/category/:category  # 按分类获取新闻
```

### 事件管理
```
GET    /api/v1/events            # 获取事件列表
GET    /api/v1/events/hot        # 获取热门事件
GET    /api/v1/events/:id        # 获取事件详情
POST   /api/v1/events            # 创建事件（需认证）
PUT    /api/v1/events/:id        # 更新事件（需认证）
DELETE /api/v1/events/:id        # 删除事件（需认证）
```

### RSS管理（管理员）
```
GET    /api/v1/rss/sources       # 获取RSS源列表
POST   /api/v1/rss/sources       # 创建RSS源
PUT    /api/v1/rss/sources/:id   # 更新RSS源
DELETE /api/v1/rss/sources/:id   # 删除RSS源
POST   /api/v1/rss/sources/:id/fetch  # 手动抓取RSS源
POST   /api/v1/rss/fetch-all     # 抓取所有RSS源
```

### 管理员接口
```
GET    /api/v1/admin/stats       # 系统统计
GET    /api/v1/admin/users       # 用户管理
GET    /api/v1/admin/events      # 事件管理
GET    /api/v1/admin/news        # 新闻管理
```

## ⚙️ 配置说明

### 数据库配置
- `host`: 数据库主机地址
- `port`: 数据库端口
- `user`: 数据库用户名
- `password`: 数据库密码
- `db_name`: 数据库名称
- `ssl_mode`: SSL模式（disable/require）
- `max_idle_conns`: 最大空闲连接数
- `max_open_conns`: 最大打开连接数

### JWT配置
- `secret_key`: JWT签名密钥（生产环境必须修改）
- `expire_hours`: Token过期时间（小时）

### CORS配置
- `allow_origins`: 允许的跨域来源列表

### 管理员配置
- `email`: 默认管理员邮箱
- `username`: 默认管理员用户名
- `password`: 默认管理员密码

## 🔄 自动化功能

### RSS调度器
系统内置智能RSS调度器，自动执行以下任务：

- **📡 RSS抓取** - 每30分钟自动抓取所有活跃RSS源
- **🧹 数据清理** - 每小时清理过期和重复数据
- **🔥 热度计算** - 每6小时重新计算内容热度分数
- **📊 统计更新** - 实时更新浏览量、点赞数等统计信息

### 种子数据初始化
- 首次启动自动检测数据库状态
- 自动导入 `data/new.json` 中的2600+条新闻数据
- 创建默认管理员账户
- 防重复导入机制

## 🛠️ 开发指南

### 代码结构
项目采用清晰的分层架构：
- **Handler层** - HTTP请求处理和路由
- **Service层** - 业务逻辑处理
- **Model层** - 数据模型定义
- **Repository层** - 数据访问（通过GORM）

### 添加新功能
1. 在 `models/` 中定义数据模型
2. 在 `services/` 中实现业务逻辑
3. 在 `api/` 中添加HTTP处理器
4. 在 `router.go` 中注册路由

### 数据库迁移
```bash
# 项目会在启动时自动执行迁移
# 新增模型后重启应用即可自动创建表结构
```

### 添加RSS源
```bash
# 通过API添加
curl -X POST http://localhost:8080/api/v1/rss/sources \
  -H "Authorization: Bearer <token>" \
  -H "Content-Type: application/json" \
  -d '{
    "name": "示例RSS源",
    "url": "https://example.com/rss.xml",
    "category": "科技",
    "language": "zh",
    "priority": 1,
    "update_freq": 60
  }'
```

## 🐳 Docker部署

### 使用Docker Compose（推荐）

```yaml
version: '3.8'
services:
  easypeek-backend:
    build: .
    ports:
      - "8080:8080"
    environment:
      - DB_HOST=postgres
      - DB_USER=postgres
      - DB_PASSWORD=password
      - DB_NAME=easypeekdb
      - REDIS_ADDR=redis:6379
    depends_on:
      - postgres
      - redis

  postgres:
    image: postgres:15
    environment:
      - POSTGRES_DB=easypeekdb
      - POSTGRES_USER=postgres
      - POSTGRES_PASSWORD=password
    volumes:
      - postgres_data:/var/lib/postgresql/data

  redis:
    image: redis:7-alpine
    volumes:
      - redis_data:/data

volumes:
  postgres_data:
  redis_data:
```

### 构建Docker镜像

```bash
docker build -t easypeek-backend .
```

## 📊 监控和日志

### 系统日志
应用启动时会输出详细的日志信息：
```
数据库连接成功
数据库迁移完成
开始初始化种子数据...
RSS调度器启动成功
服务器启动在端口 :8080
```

### RSS抓取日志
```
[RSS SCHEDULER] Starting scheduled RSS fetch...
RSS fetch summary - New: 15, Updated: 3, Errors: 0
[RSS SCHEDULER] Scheduled RSS fetch completed
```

### 健康检查
```bash
curl http://localhost:8080/health
```

## 🔒 安全特性

- **JWT认证** - 无状态的用户身份验证
- **角色权限** - 用户、管理员、系统三级权限
- **密码加密** - bcrypt安全哈希
- **CORS保护** - 跨域请求安全控制
- **SQL注入防护** - GORM ORM安全查询
- **输入验证** - 请求参数严格验证

## 🤝 贡献指南

我们欢迎社区贡献！请遵循以下步骤：

1. **Fork** 项目到你的GitHub账户
2. **创建** 功能分支：`git checkout -b feature/amazing-feature`
3. **提交** 你的更改：`git commit -m 'Add some amazing feature'`
4. **推送** 到分支：`git push origin feature/amazing-feature`
5. **创建** Pull Request

### 代码规范
- 遵循Go官方代码风格
- 添加适当的注释和文档
- 编写单元测试
- 确保所有测试通过

## 📝 更新日志

### v1.0.0 (当前版本)
- ✨ 完整的新闻管理系统
- ✨ RSS源自动抓取功能
- ✨ 事件聚合和管理
- ✨ 用户认证和权限系统
- ✨ 智能热度算法
- ✨ 自动种子数据导入
- ✨ RESTful API接口

## 📄 许可证

本项目采用 [MIT License](LICENSE) 开源协议。

## 📞 联系我们

- **项目主页**: https://github.com/EasyPeek/EasyPeek-backend
- **问题反馈**: https://github.com/EasyPeek/EasyPeek-backend/issues
- **讨论区**: https://github.com/EasyPeek/EasyPeek-backend/discussions

## 🙏 致谢

感谢所有为这个项目做出贡献的开发者和开源社区的支持！

---

⭐ 如果这个项目对你有帮助，请给我们一个Star！